# Discord Kittens

A real-time multiplayer Exploding Kittens clone built with a **Go WebSocket backend** and a **React + TypeScript frontend**. It runs as a **Discord activity** (embedded app) and also works standalone in a browser.

It runs as a single process, or as a **cluster of identical nodes coordinated through Redis** — where a node can be killed mid-game and another one picks the game up, nope countdown and all, without anyone losing their hand. Under a scripted chaos run, killing a node mid-game cost **zero games out of 16** ([results](#results)).

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│  Client (React + TypeScript)                                 │
│  Discord embedded-app SDK · Vite dev server ──proxy──► /api/* │
└──────────────┬───────────────────────────────┬───────────────┘
               │ HTTP (lobby, OAuth token)     │ WebSocket (game actions + state)
               ▼                               ▼
┌──────────────────────────────────────────────────────────────┐
│  Server (Go)  ×N behind a load balancer                      │
│                                                              │
│  ┌─────────┐    ┌───────────────────────────────────────┐    │
│  │ HTTP    │    │  Lobby (per-game instance)            │    │
│  │ Router  │───►│                                       │    │
│  │         │    │  ActionQueue ◄── player actions       │    │
│  │ /lobby  │    │  JoinQueue   ◄── new connections      │    │
│  │ /ws     │    │                                       │    │
│  │ /token  │    │  run() event loop (select on chans)   │    │
│  │ /metrics│    │       │                               │    │
│  └─────────┘    │       ├─ action.go  (card logic)      │    │
│       │         │       ├─ cards.go   (deck/dealing)    │    │
│       │         │       └─ turns.go   (state broadcast) │    │
│       │         └───────────────────────────────────────┘    │
└───────┼──────────────────────────────────────────────────────┘
        │ don't own this lobby? ──► proxy the socket to the node that does
        │ leases · fencing epochs · game snapshots
        ▼
   ┌─────────┐
   │  Redis  │   (optional — omit it and you get today's single process)
   └─────────┘
```

The `/token` endpoint (in `discord.go`) exchanges a Discord OAuth2 authorization code for an access token, so the client can identify the player without exposing the client secret.

## Backend (Go)

The backend is a concurrent WebSocket game server built on Go's standard library and [Gorilla WebSocket](https://github.com/gorilla/websocket). No frameworks -- just channels, goroutines, and a clean state machine. (Clustering adds go-redis and the Prometheus client; neither is on the path of a single-process run.)

### Concurrency Model

Each lobby runs its own goroutine with a **channel-based event loop**:

```go
select {
case joinReq := <-lobby.JoinQueue:   // new player connected
case action  := <-lobby.ActionQueue: // player performed an action
}
```

All state mutations flow through these channels, so no mutexes are needed -- the event loop serializes access naturally. Each connected player gets their own buffered `Send chan GameState`, and the server pushes personalized state snapshots (hiding other players' hands) after every action. A player whose send buffer fills up is treated as disconnected and dropped.

### Reconnection & Keepalive

Players carry a stable `UserId` (their Discord user id). When a player rejoins -- even mid-game -- the lobby reconnects them to their existing seat by `UserId` instead of creating a new one, so a dropped connection or a refreshed tab doesn't lose your hand. An id-less player can reclaim a disconnected id-less seat. The server also pings each WebSocket on an interval (and drops clients that stop ponging) so reverse proxies like Cloudflare don't sever idle connections.

### Spectating & Lobby Reaping

Someone who joins after the game has already started -- and can't reclaim a seat -- is admitted as a **watch-only spectator**: they receive a fully public game-state snapshot (no hands, future, or private reveals) and can't take actions. On a lobby restart, spectators are promoted into real seats. A lobby with no live connections (players or spectators) is automatically **reaped after 60 seconds**, and a join request racing that teardown transparently re-creates and retries against a fresh lobby.

### State Machine

The game uses a 9-state turn machine to enforce valid transitions:

```
NotStarted ──► Normal ──► SeeingTheFuture
                  │          AlteringTheFuture
                  │          AwaitingKittenPlacement
                  │          AwaitingFavor
                  │          AwaitingDiscardTake
                  │          AcceptingNopes
                  └──────► GameOver
```

Every action is guarded by `assertTurnAndState()`, which validates both whose turn it is and which state transitions are legal. This prevents invalid game states at the protocol level.

### Nope Window

Single cards and 2/3-card combos don't resolve instantly. Playing one stashes a **pending action**, enters `AcceptingNopes`, and starts a 5-second timer. Any living player holding a **Nope** can play it to flip the action between noped and "yuped" (a nope on a nope), each play restarting the timer. When the window closes, the action is either applied or cancelled depending on how many nopes landed. The closing deadline is broadcast to clients so they can render a live countdown. (5-card combos and forced responses like favors are not nopeable.)

### Card System

17 card types with a deck whose composition scales by player-count tier (`GetDeckConfig` / `GetExtraDefuses` in `cards.go`):

| Category | Cards                                                                                                   | Count                                                                        |
| -------- | ------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| Danger   | Exploding Kitten, Defuse                                                                                | n-1 kittens; everyone gets 1 defuse, with extras dealt into the deck by tier |
| Action   | Skip, Attack, Targeted Attack, Shuffle, Draw From Bottom, See The Future, Alter The Future, Favor, Nope | tiered (more copies as the table grows)                                      |
| Combo    | Tacocat, Hairy Potato Cat, Cattermelon, Rainbow-Ralphing Cat, Rage Cat, Feral Cat                      | tiered (more copies as the table grows)                                      |

The three tiers are **2-3 players (small)**, **4-6 (medium)**, and **7-10 (large)** -- each card type has a per-tier count rather than a flat multiplier.

### Action Handling

The action handler (`action.go`) processes 12 distinct action types through a switch dispatch:

- **StartGame** -- initialize deck, deal hands, insert kittens
- **RandomizeOrder** -- shuffle the seating order before the game begins (pre-game only)
- **RestartLobby** -- reset a finished/in-progress game back to the lobby, promoting spectators to seated players
- **DrawCard** -- draw the top card, with Exploding Kitten / Defuse resolution
- **PlayCard** -- queue a single card's effect (Skip, Attack, Targeted Attack, Shuffle, Draw From Bottom, See/Alter The Future, Favor) for the nope window
- **PlayNope** -- nope (or "yup") the pending action during the nope window
- **Combo** -- 2-card (steal random), 3-card (steal named), and 5-card (take from discard) combinations
- **PlaceKitten** -- reinsert a defused Exploding Kitten at any deck position
- **AlterFuture** -- submit a rearranged ordering of the top 3 cards
- **GiveFavor** -- respond to a favor request by handing over a chosen card
- **TakeFromDiscard** -- complete a 5-card combo by taking a named card from the discard pile
- **Disconnect** -- handle a player leaving (reconciled against reconnects)

### Combo Validation

The combo system supports three tiers with strict validation:

- **2 matching cards** -- steal a random card from a target player
- **3 matching cards** -- name a specific card to steal from a target
- **5 unique cards** -- take any card from the discard pile

Feral Cat acts as a wildcard that can substitute for any cat type in 2 and 3-card combos.

## Distributed Mode

Set `REDIS_URL` and the same binary becomes a cluster node. Leave it unset and nothing changes -- the coordinator falls back to an in-memory implementation whose methods are no-ops, so the single-process deploy runs exactly as it always did, with no new dependencies.

### Lobbies don't span nodes

A lobby lives entirely on **one** node, so the single-goroutine event loop -- the thing that makes the game logic simple and correct -- is untouched. What's distributed is the *assignment* of lobbies to nodes. Every node is identical and accepts any connection (it has to be: the Discord activity proxy forces a single origin), and a node that doesn't own the requested lobby **proxies the WebSocket to the node that does**. No separate gateway service.

The proxy is transparent to ping/pong -- it relays control frames rather than answering them -- so the owner's keepalive reaches the real client end-to-end and a dead client is still detected by the node that actually cares.

### Ownership: leases and fencing epochs

Three Redis keys per lobby, every operation a Lua script so it's atomic:

| Key                    | Role                                                                        |
| ---------------------- | --------------------------------------------------------------------------- |
| `lobby:{name}:owner`   | 15s lease holding the owner's address, renewed by a per-lobby heartbeat every 5s |
| `lobby:{name}:epoch`   | Fencing token, incremented on every acquisition                              |
| `lobby:{name}:state`   | The latest game snapshot                                                     |

Every write -- renew, release, delete, snapshot -- is **guarded by the epoch, never by the address**. That's what kills the zombie-owner failure mode: a node that pauses past its lease TTL, gets replaced, then wakes up still believing it owns the lobby finds its epoch stale and every write bounces. A renewal that fails (as opposed to a Redis error, which is retried) means someone else has taken over, so the old owner tears its copy down rather than running a second authority for the same game.

The lease's `SET NX` also does double duty as the cross-node creation lock, which makes Discord's `create=1` auto-join atomic cluster-wide for free.

### Snapshots and failover

The full game -- deck, discard, every hand, turn index, attack state, the pending nopeable action, and the log -- serializes to a compact JSON snapshot, written synchronously at the end of every event-loop iteration that mutated state. Sockets and channels are deliberately excluded: connections are per-node ephemera.

**Timers persist as absolute deadlines, never remaining durations**, so a nope window that had 2.6 seconds left when its node died still has 2.6 seconds left when another node picks it up.

Acquiring a lease returns any snapshot the previous owner left behind, so winning ownership and adopting the game are one atomic step. From there, recovery is the reconnect path that already existed: clients auto-reconnect with backoff, land on the new owner, and reclaim their seats by Discord user id. A graceful teardown (game over, empty-lobby reap) deletes the state key so a future lobby of the same name can't adopt a stale game; a crash leaves it behind on purpose, for exactly that adoption.

### Exactly-once moves across a reconnect

Within one socket the WebSocket is ordered, but the reconnect boundary isn't: a client that dropped mid-send can't know whether its action landed. So every action carries a client-assigned sequence number, the client holds un-acked actions in an outbox and replays them on reconnect, and the server drops anything at or below the last sequence it applied for that player (a value that rides along in the snapshot, so it survives failover too). At-least-once delivery, idempotent application -- retrying a Skip can't cost you two turns.

### Graceful drain

On SIGTERM a node stops accepting new lobbies, snapshots and releases each one (keeping the state key for its successor), and disconnects its clients. Players blip through a single reconnect onto surviving nodes, so a rolling deploy costs nobody a game.

### Proving it

`/metrics` exposes Prometheus counters for actions applied, lobbies adopted after a failover, snapshot write latency, and active lobbies; `/healthz` backs the load balancer's readiness check.

The load harness (`game/loadbot`) runs N bots that speak the real WebSocket protocol and play random legal moves, sharing wire types with the server so the harness can't drift from it. It reports games survived, client recovery p50/p99/max, action latency, and reconnect counts.

`scripts/chaos.sh` is the whole story end to end: bring up the 3-node cluster, throw a bot swarm at it, `docker kill` a node mid-game, and print both sides of the result -- how many lobbies each survivor adopted, and how many games the bots (who know nothing about Redis) actually kept playing. Target is 100%.

```bash
docker compose up --build     # 3 nodes + Redis + nginx on :8080
scripts/chaos.sh              # the full kill-a-node run
```

### Results

60 bots across 15 lobbies, 3 nodes behind nginx, 120-second run, `node2` killed outright at t+40s:

| Measure                          | Result                              |
| -------------------------------- | ----------------------------------- |
| **Games survived**               | **16 / 16 (100%)**                  |
| Lobbies adopted by survivors     | 6 (node1: 3, node3: 3)              |
| Client recovery after the kill   | p50 11.4s · p99 13.6s · max 15.0s   |
| Action latency                   | p50 2.2ms · p99 4.5ms · max 154ms   |
| Actions applied                  | 885 (7.4/s)                         |
| Connections                      | 224 (164 reconnects)                |

Every game whose node was killed came back on a survivor with its hand, deck, and turn order intact -- no bot lost a game, and no action was applied twice across the reconnects.

Recovery time is dominated by the 15-second lease TTL: nobody may adopt a lobby until the dead owner's lease actually lapses in Redis, and the client may be mid-backoff when it does. That's the knob to turn if you want faster failover -- a shorter TTL recovers quicker at the cost of more heartbeat traffic and less tolerance for a node that's merely slow.

## Frontend (React + TypeScript)

A complete, playable game UI rendered around a virtual table, plus a browser fallback lobby screen.

**Stack:** React 19, TypeScript, Vite, TailwindCSS, React Router v7, Discord Embedded App SDK

What's implemented:

- **Discord activity integration** -- the embedded-app SDK handshake, OAuth (`identify`) via the backend `/token` exchange, and auto-create/join of the lobby from the activity instance id
- **Round-table game screen** -- player seats around a felt, deck/discard piles, turn indicator, under-attack notice, and an error banner
- **Hand & action bar** -- click cards to select, play singles or combos, draw, and start the game
- **Interaction prompts** -- target picker, kitten placement, See/Alter The Future viewer & drag-reorder, favor giver, discard picker, and a game-over overlay
- **Nope window** -- a Nope button and banner with a live countdown of the nope deadline, so anyone holding a Nope can cancel (or re-allow) a pending play
- **Restart flow** -- a confirm prompt to reset a finished game back to the lobby (spectators included)
- **Spectator mode** -- watch-only clients that joined mid-game get a public view with the hand and action bar hidden behind a "Spectating" banner
- **Live game log** -- a running, per-player feed of every action (with private reveals to the players involved)
- **Resilient connection** -- auto-reconnect with exponential backoff + jitter, reconnecting to the same seat by Discord user id, with un-acked actions held in an outbox and replayed once the socket is back (the server dedups them, so nothing double-plays)
- **Browser fallback** -- create or join a lobby by name when running outside Discord

## Tech Stack

| Layer         | Technology                        |
| ------------- | --------------------------------- |
| Backend       | Go 1.25                           |
| WebSocket     | Gorilla WebSocket 1.5             |
| Coordination  | Redis 7 (go-redis v9), Lua scripts |
| Observability | Prometheus client_golang          |
| Cluster       | Docker Compose + nginx            |
| Frontend      | React 19, TypeScript 5.9          |
| Discord       | Discord Embedded App SDK 2.5      |
| Build         | Vite 8                            |
| Styling       | TailwindCSS 4                     |
| Routing       | React Router 7                    |

## Getting Started

> Running it as a **Discord activity** (creating the Discord app, tunnels/hosting, and URL
> mappings)? See **[SETUP.md](SETUP.md)** for the full step-by-step guide. The quick start
> below is for running the backend and frontend locally.

**Environment:** Create a `.env` in the repo root (shared by the backend and Vite):

```bash
VITE_DISCORD_CLIENT_ID=your_discord_app_client_id
DISCORD_CLIENT_SECRET=your_discord_app_client_secret
```

**Backend:**

```bash
cd game
go run .
# Serves on :8080
```

**Frontend:**

```bash
cd client
npm install
npm run dev
# Vite dev server proxies /api (including the /api/ws upgrade) to :8080
```

Standalone, open the frontend, create a lobby, and share the lobby name with other players. As a Discord activity, launch it from a voice channel -- the lobby is created and joined automatically from the activity instance.

**Cluster (optional):**

```bash
docker compose up --build
# nginx on :8080 round-robins across node1/node2/node3, all sharing one Redis
```

Point the frontend at `:8080` as usual -- which node you land on doesn't matter. Set `REDIS_URL` (plus `NODE_ID` / `ADVERTISE_ADDR`) to run a node outside compose; leave `REDIS_URL` unset for the single-process mode.

## Project Structure

```
game/
  server.go             HTTP endpoints (lobby, ws, token) + WS upgrade, keepalive,
                        cross-node proxying, and graceful drain
  lobby.go              Lobby state, join/reconnect/disconnect, event loop,
                        snapshot serialization, lease heartbeat
  action.go             Card action dispatch, game logic, action dedup
  cards.go              Card definitions, tiered deck generation, dealing
  turns.go              Turn rotation, game log, state broadcasting & persistence
  discord.go            Discord OAuth2 token exchange (/token)
  config.go             Env-driven config (port, node id, advertise addr, Redis URL)
  coordinator.go        Coordinator interface + in-memory single-node implementation
  redis-coordinator.go  Redis leases, fencing epochs, and snapshot writes (Lua)
  metrics.go            Prometheus metrics (/metrics) and /healthz
  wire/                 Wire types shared by the server and the load harness
  loadbot/              Bot swarm that plays the real protocol, with a run report

scripts/chaos.sh        Kill-a-node chaos run against the compose cluster
docker-compose.yml      3 nodes + Redis + nginx
nginx.conf              Round-robin front door with WebSocket upgrade support

client/
  src/
    discord/sdk.ts    Discord embedded-app SDK setup + OAuth handshake
    pages/home/       Browser-fallback lobby creation and join UI
    pages/game/       Game screen, WS lifecycle, table layout, and interaction components
    models/           TypeScript types mirroring backend state
```
