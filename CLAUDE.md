# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

**Backend (Go):**

```bash
cd game
go run .          # Start server on :8080 (single node; no Redis needed)
go build .        # Build binary
go vet ./...      # Lint
go test ./...     # Snapshot round-trip tests
```

**Cluster (3 nodes + Redis + nginx):**

```bash
docker compose up --build       # nginx round-robins :8080 across node1/2/3
cd game && go run ./loadbot -bots 60 -perLobby 4 -duration 120s   # bot swarm
scripts/chaos.sh                # cluster + swarm + kill a node, print survival %
```

**Frontend (React + TypeScript):**

```bash
cd client
npm run dev       # Vite dev server (proxies /api and /ws to :8080)
npm run build     # tsc + vite build
npm run lint      # ESLint
npm run test      # Vitest (jsdom)
```

**Environment:** A repo-root `.env` (loaded by both the Go backend via godotenv and Vite via `envDir: '..'`) provides `VITE_DISCORD_CLIENT_ID` and `DISCORD_CLIENT_SECRET` for the Discord OAuth flow. `config.go` (`LoadConfig` → the global `cfg`) also reads `PORT` (8080), `NODE_ID`/`ADVERTISE_ADDR` (default from the hostname), and `REDIS_URL`. **`REDIS_URL` is the only switch between single-node and clustered mode** — unset means today's in-memory behavior.

## Architecture

A real-time multiplayer Exploding Kittens clone that runs as a Discord activity (embedded app) and also standalone in a browser. The backend is a Go WebSocket server; the frontend is React + TypeScript + Vite. There is no REST API for game actions — all game communication after joining goes through a single WebSocket connection. The backend runs either as a single process or as N identical nodes coordinated through Redis (see **Distributed mode** below); the game logic is identical in both.

### Backend (`game/`)

| File                   | Responsibility                                                                                                                                                                                         |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `server.go`            | HTTP handlers (`POST /api/lobby`, `GET /api/ws`, `POST /api/token`), WebSocket upgrade + ping/pong keepalive, `resolveLobby` (own / proxy / adopt), cross-node WS proxy, graceful drain in `main()`    |
| `lobby.go`             | `Lobby` struct and `run()` event loop, `TurnState` enum, `GameState`/`PlayerAction` types, join/reconnect/disconnect logic, `LobbySnapshot` + `SerializeLobby`/`DeserializeLobby`, lease `heartbeat()` |
| `action.go`            | `receivePlayerAction()` switch dispatch for all 12 action types, seq-number dedup, plus the Nope window (`startNopeTimer`/`handleNopeTimerComplete`/`confirmPlayerAction`/`denyPlayerAction`)          |
| `cards.go`             | `Card` type, all 17 card definitions, tiered deck generation (`ActionCardTotals`/`DefuseTotals` by player count), shuffle/deal                                                                         |
| `turns.go`             | Turn rotation (`setNextPlayerTurn`), `assertTurnAndState()` guard, game log (`recordAction`), `getGameState`/`broadcastGameState` (which also writes the snapshot)/`sendError`                         |
| `discord.go`           | `POST /token` — exchanges a Discord OAuth2 code for an access token using the `.env` client id/secret                                                                                                  |
| `config.go`            | `Config`/`LoadConfig` → the global `cfg` (port, node id, advertise addr, Redis URL, Discord creds)                                                                                                     |
| `coordinator.go`       | The `Coordinator` interface (`Acquire`/`Lookup`/`Delete`/`Release`/`Refresh`/`UpdateState`), lease/state TTL constants, `LocalCoordinator` (single-node no-op impl), `newCoordinator()` selection      |
| `redis-coordinator.go` | `RedisCoordinator` — all five Lua scripts (acquire, delete, release, refresh, update-state) and the `lobby:{name}:{owner,epoch,state}` key layout                                                      |
| `metrics.go`           | Prometheus collectors + `/metrics` and `/healthz`                                                                                                                                                      |
| `wire/wire.go`         | Shared wire types (`GameState`, `ActionRequest`) and action/card/turn-state string constants, so `loadbot` can't drift from the server                                                                 |
| `loadbot/`             | Standalone bot swarm (`go run ./loadbot`): fake players speaking the real WS protocol, plus the run report (survival %, recovery p50/p99, action latency)                                              |
| `Dockerfile`           | Static Go build → alpine, used by `docker-compose.yml`                                                                                                                                                 |

**Concurrency model:** Each `Lobby` runs a single goroutine with a `select` loop over `JoinQueue chan JoinRequest`, `ActionQueue chan PlayerAction`, two internal timers — a `nopeTimer` (the open Nope window) and an `emptyTimer` (the empty-lobby reap countdown) — and two shutdown signals: `leaseLost` (the heartbeat goroutine lost the Redis lease) and the process-wide `serverShutdownChannel` (SIGTERM drain). No mutexes inside a lobby; all state mutation is serialized by the event loop. Each player has a buffered `Send chan GameState`; `sendTo` drops a player whose buffer is full (treated as disconnected). A lobby never spans nodes, so this invariant is untouched by clustering — only the _assignment_ of lobbies to nodes is distributed.

**State machine:** `TurnState` has 9 values (`Normal`, `NotStarted`, `GameOver`, `AwaitingKittenPlacement`, `SeeingTheFuture`, `AlteringTheFuture`, `AwaitingFavor`, `AwaitingDiscardTake`, `AcceptingNopes`). Every action calls `assertTurnAndState()` with a list of valid states before mutating anything.

**Nope window:** Playing a single card or a 2/3-card combo doesn't take effect immediately — it records a `pendingAction` (`PendingNopeableAction`), enters `AcceptingNopes`, and arms a 5s `nopeTimer` (`nopeDelay`). Any living player holding a `Nope` can `PlayNope` to flip the action's `isNoped` state (nope/yup), which restarts the timer. When the timer fires, `handleNopeTimerComplete` either runs `confirmPlayerAction` (apply the effect) or `denyPlayerAction` (cancelled by an odd number of nopes). The current `nopeDeadline` (unix ms) is broadcast so clients can show a countdown. 5-card combos and other actions are not nopeable.

**GameState is personalized:** `getGameState(playerIdx)` builds a snapshot for each player individually — other players' hands are hidden (only `cardCount` exposed); `future`/`discardOptions` are only populated for the active player in the relevant states; and the action log / `lastAction` are resolved per-player (a `LastAction.Private` message overrides the public one, e.g. to reveal a stolen card only to the players involved).

**Reconnection:** Players carry a stable `UserId` (the Discord user id). `handleJoin` reconnects a returning player to their existing seat by `UserId`, even mid-game. An id-less player can reclaim a disconnected id-less seat.

**Spectating:** A newcomer who joins a lobby whose game is already in progress and can't claim/reclaim a seat is admitted as a watch-only spectator instead of being rejected. Spectators live in a separate `lobby.spectators` map (their own id space, not in `playersList`/`playersMap`), receive only the fully public snapshot from `getSpectatorState()` — no hand, future, discard options, or private log/banner reveals — and cannot take any action (the WS read loop only watches for their disconnect). The spectator state carries `isSpectator: true` and `playerId: -1`; the client hides the hand/action bar and shows a "Spectating" banner. The WS query params are `lobby`, `username`, `userId`, `avatar`, and `create=1` (Discord auto-creates the instance lobby on the fly). `username`/`userId`/`avatar` are resolved client-side from the Discord SDK user object (`discord/sdk.ts`); each player's `avatar` is broadcast in `GameState` so seats render the Discord profile image, with an emoji fallback when it's empty or fails to load.

**Lobby lifecycle:** `POST /api/lobby` creates named lobbies; `create=1` on `/api/ws` auto-creates one atomically. On game over the lobby's `turnState` becomes `GameOver` but the lobby entry persists in the `lobbies` map. A lobby with zero live connections (players + spectators) is reaped after `emptyLobbyTTL` (60s): `refreshEmptyTimer` arms the `emptyTimer` when the last client leaves and stops it when one returns, and `destroy()` removes the lobby from the map and closes its `done` channel. `done` lets a WS joiner racing a reap detect the dead goroutine and re-resolve/retry instead of blocking on `JoinQueue` forever (`server.go`).

**Restart & order:** The `RandomizeOrder` action (only valid in `NotStarted`) shuffles the seating order before the game begins. The `RestartLobby` action runs `resetToLobby`: it clears all game state back to `NotStarted`, drops players who went offline mid-game, and promotes every spectator into a seated player in place (same id + socket), so a finished game can be replayed with the watchers now playing.

### Distributed mode (`game/coordinator.go`, `redis-coordinator.go`)

N identical nodes sit behind a round-robin load balancer; any node accepts any connection (required anyway — the Discord activity proxy forces one origin). A lobby lives entirely on **one** node; Redis coordinates who owns what.

**Coordinator interface.** `newCoordinator()` picks the implementation from `cfg.RedisURL`:

- `LocalCoordinator` (unset): `Acquire` always returns this node's own address, everything else is a no-op — byte-for-byte the old single-process behavior, no new dependencies.
- `RedisCoordinator` (set): leases, epochs, and snapshots, every operation a Lua script so it's atomic.

**Keys** (all per lobby name): `lobby:{name}:owner` — the lease, `SET` with `leaseTTL` (15s), holding the owner's `ADVERTISE_ADDR`. `lobby:{name}:epoch` — a fencing token `INCR`ed on every acquisition. `lobby:{name}:state` — the latest `LobbySnapshot` JSON (`stateTTL`, 10min).

**Everything is epoch-guarded, never address-guarded.** `Refresh`, `Delete`, `Release`, and `UpdateState` all compare the caller's epoch against `lobby:{name}:epoch` before acting, so a zombie owner (paused past its TTL, replaced, then woken) can't renew a lease it lost or overwrite a newer owner's snapshot. `Release` also `INCR`s the epoch, which fences the outgoing owner's in-flight heartbeat.

**Routing (`resolveLobby` in `server.go`).** Lobby present locally → serve it (today's path). Otherwise `Lookup` (join) or `Acquire` (`create=1`): if the winner is this node, start the lobby here; if it's another node, `proxyWebSocket` dials that node's `/api/ws` with the `X-Kittens-Proxied` loop-guard header and pumps frames both directions. The proxy is **transparent to ping/pong** (`forwardControl` relays control frames rather than answering them), so the owner's keepalive reaches the real client end-to-end. The lease `SET NX` inside `acquireScript` is what makes `create=1` atomic cluster-wide — it replaces `lobbiesMutex` for cross-node creation.

**Snapshots.** `SerializeLobby`/`DeserializeLobby` (`lobby.go`) round-trip the whole game: deck, discard, per-player hand/`DiscordUserId`/name/avatar/`IsAlive`/`LastAcked`, turn index, `turnsToTake`, `underAttack`, `turnState`, `pendingAction`, `nopeDeadline`, and the log. `Send` channels and sockets are excluded by design — connections are per-node ephemera. **Timers persist as absolute deadlines**: `nopeDeadline` is stored as a timestamp and the `nopeTimer` re-armed from `time.Until(...)` on restore, so a nope window survives failover with the countdown intact. The write happens synchronously in `broadcastGameState` — i.e. at the end of every mutating event-loop iteration — timed into `kittens_snapshot_write_seconds`. `snapshot_test.go` covers the round trip.

**Failover.** `Acquire` returns any snapshot the previous owner left behind (empty when there's nothing to inherit), so winning the lease and adopting the game are one atomic step. The new owner deserializes with all players `IsOnline: false`, bumps `kittens_failovers_total`, and starts `run()`. Recovery from there is the **existing** reconnect machinery: clients auto-reconnect with backoff, land on the new owner, and `handleJoin` reclaims each seat by `DiscordUserId`. `Delete` (graceful reap / game over) drops the state key so a re-created lobby of the same name can't adopt a stale snapshot; a crashed owner never runs it, so its state survives on TTL for adoption.

**Losing the lease.** Each lobby runs a `heartbeat()` goroutine renewing every `heartbeatInterval` (5s). A failed _renewal_ (not a Redis error — those are retried) means another node has taken over, so it closes `leaseLost`; the event loop disconnects everyone and destroys the local copy rather than running a second authority for the same game.

**Action dedup.** `ActionRequest` carries a client-assigned `seqNumber`; `receivePlayerAction` drops anything `<= player.LastAcked` and the value rides along in the snapshot. Every `GameState` echoes `lastAcked`. The client keeps an outbox of un-acked actions and replays it on reconnect (`pages/game/page.tsx`), so a move that landed pre-drop is a no-op and one that didn't lands now — at-least-once delivery, idempotent application.

**Graceful drain.** SIGTERM: stop the HTTP server, set `shuttingDown` (so `resolveLobby` refuses to adopt lobbies it's about to abandon), close `serverShutdownChannel`. Each lobby then runs `gracefullyShutdown` — disconnect clients, `Release` the lease while **keeping** the state key — and `main` waits on `lobbyWG` with a 10s cap. Clients blip through one reconnect onto surviving nodes; deploys lose zero games.

**Observability.** `/metrics` exposes `kittens_actions_total`, `kittens_failovers_total`, `kittens_snapshot_write_seconds`, and `kittens_active_lobbies` (a `GaugeFunc` reading the live map); `/healthz` returns 200 for load-balancer and compose readiness checks.

**Proving it.** `game/loadbot/` runs N bots playing random legal moves over the real WS protocol (types shared via `game/wire`, so the harness can't drift), reporting games survived, recovery p50/p99/max, action latency, and reconnect counts. `scripts/chaos.sh` is the end-to-end run: bring up the compose cluster, launch the swarm, `docker compose kill` a node mid-game, then print each survivor's `kittens_failovers_total` alongside the bots' own survival line (target: 100%).

### Frontend (`client/src/`)

| Path                        | Responsibility                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| --------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `models/`                   | TypeScript types mirroring backend: `GameState`, `ActionRequest`, `CardType`, `TurnState`, `ConnectionStatus`                                                                                                                                                                                                                                                                                                                                               |
| `discord/sdk.ts`            | Discord embedded-app SDK setup, OAuth handshake, `getUsername`/`getUserId`/`getInstanceId`                                                                                                                                                                                                                                                                                                                                                                  |
| `pages/home/`               | Lobby creation and join UI (browser fallback)                                                                                                                                                                                                                                                                                                                                                                                                               |
| `pages/game/`               | Game screen + WS lifecycle with auto-reconnect/backoff and the seq-numbered action outbox replayed on reconnect (`page.tsx`); `components/` for the table, hand, action bar, game log, and every interaction prompt (target, kitten placement, see/alter future, favor, discard pick, game over). Nope flow lives in `NopeBanner`/`NopeButton` + the `use-nope-countdown` hook (live countdown of `nopeDeadline`); restart confirmation in `RestartConfirm` |
| `pages/game/table-utils.ts` | Seat layout math (has Vitest coverage)                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `App.tsx`                   | React Router setup                                                                                                                                                                                                                                                                                                                                                                                                                                          |

The backend serves every route under `/api` (`/api/lobby`, `/api/ws`, `/api/token`) so one prefix works unchanged across all three environments: the Vite dev proxy (`/api/*` → `http://localhost:8080`, with the `/api/ws` upgrade), the Vercel rewrite (`/api/:path*` → the Render backend), and the Discord activity URL mapping — none of them strip the prefix. The WebSocket base is context-aware (`pages/game/page.tsx`): same-origin inside Discord so its proxy can forward it (CSP blocks direct connections), otherwise `VITE_WS_URL` (set at build time on Vercel, since Vercel can't proxy WebSockets) or same-origin in dev. Inside Discord the frontend derives the lobby from the activity instance id and auto-creates/joins; standalone it uses the lobby name from the home page.

### Local cluster (`docker-compose.yml`, `nginx.conf`)

`docker compose up --build` brings up `node1`/`node2`/`node3` (each built from `game/Dockerfile`, each with `REDIS_URL`, `NODE_ID`, and `ADVERTISE_ADDR`), a Redis 7 instance, and an nginx front door on `:8080` that round-robins across the nodes with WebSocket upgrade headers set and a long `proxy_read_timeout`. Discord credentials come from the repo-root `.env` via `env_file`. The nodes are deliberately not port-mapped — hitting nginx is what exercises the "you probably didn't land on the owner" path.
