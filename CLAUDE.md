# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

**Backend (Go):**

```bash
cd game
go run .          # Start server on :8080
go build .        # Build binary
go vet ./...      # Lint
```

**Frontend (React + TypeScript):**

```bash
cd client
npm run dev       # Vite dev server (proxies /api and /ws to :8080)
npm run build     # tsc + vite build
npm run lint      # ESLint
npm run test      # Vitest (jsdom)
```

**Environment:** A repo-root `.env` (loaded by both the Go backend via godotenv and Vite via `envDir: '..'`) provides `VITE_DISCORD_CLIENT_ID` and `DISCORD_CLIENT_SECRET` for the Discord OAuth flow.

## Architecture

A real-time multiplayer Exploding Kittens clone that runs as a Discord activity (embedded app) and also standalone in a browser. The backend is a Go WebSocket server; the frontend is React + TypeScript + Vite. There is no REST API for game actions — all game communication after joining goes through a single WebSocket connection.

### Backend (`game/`)

| File         | Responsibility                                                                                                          |
| ------------ | ----------------------------------------------------------------------------------------------------------------------- |
| `server.go`  | HTTP handlers (`POST /lobby`, `GET /ws`, `POST /token`), WebSocket upgrade + ping/pong keepalive, global `lobbies` map (guarded by `lobbiesMutex`) |
| `lobby.go`   | `Lobby` struct and `run()` event loop, `TurnState` enum, `GameState`/`PlayerAction` types, join/reconnect/disconnect logic |
| `action.go`  | `takePlayerAction()` switch dispatch for all 9 action types                                                             |
| `cards.go`   | `Card` type, all 16 card definitions, tiered deck generation (`ActionCardTotals`/`DefuseTotals` by player count), shuffle/deal |
| `turns.go`   | Turn rotation (`setNextPlayerTurn`), `assertTurnAndState()` guard, game log (`recordAction`), `getGameState`/`broadcastGameState`/`sendError` |
| `discord.go` | `POST /token` — exchanges a Discord OAuth2 code for an access token using the `.env` client id/secret                   |

**Concurrency model:** Each `Lobby` runs a single goroutine with a `select` loop over two channels — `JoinQueue chan JoinRequest` and `ActionQueue chan PlayerAction`. No mutexes inside a lobby; all state mutation is serialized by the event loop. Each player has a buffered `Send chan GameState`; `sendTo` drops a player whose buffer is full (treated as disconnected).

**State machine:** `TurnState` has 8 values (`Normal`, `NotStarted`, `GameOver`, `AwaitingKittenPlacement`, `SeeingTheFuture`, `AlteringTheFuture`, `AwaitingFavor`, `AwaitingDiscardTake`). Every action calls `assertTurnAndState()` with a list of valid states before mutating anything.

**GameState is personalized:** `getGameState(playerIdx)` builds a snapshot for each player individually — other players' hands are hidden (only `cardCount` exposed); `future`/`discardOptions` are only populated for the active player in the relevant states; and the action log / `lastAction` are resolved per-player (a `LastAction.Private` message overrides the public one, e.g. to reveal a stolen card only to the players involved).

**Reconnection:** Players carry a stable `UserId` (the Discord user id). `handleJoin` reconnects a returning player to their existing seat by `UserId`, even mid-game. An id-less player can reclaim a disconnected id-less seat. The WS query params are `lobby`, `username`, `userId`, and `create=1` (Discord auto-creates the instance lobby on the fly).

**Lobby lifecycle:** A hardcoded `"test"` lobby is created on startup. `POST /lobby` creates named lobbies; `create=1` on `/ws` auto-creates one atomically. On game over the lobby's `turnState` becomes `GameOver` but the lobby entry persists in the `lobbies` map.

### Frontend (`client/src/`)

| Path                   | Responsibility                                                                                          |
| ---------------------- | ------------------------------------------------------------------------------------------------------ |
| `models/`              | TypeScript types mirroring backend: `GameState`, `ActionRequest`, `CardType`, `TurnState`, `ConnectionStatus` |
| `discord/sdk.ts`       | Discord embedded-app SDK setup, OAuth handshake, `getUsername`/`getUserId`/`getInstanceId`              |
| `pages/home/`          | Lobby creation and join UI (browser fallback)                                                           |
| `pages/game/`          | Game screen + WS lifecycle with auto-reconnect/backoff; `components/` for the table, hand, action bar, game log, and every interaction prompt (target, kitten placement, see/alter future, favor, discard pick, game over) |
| `pages/game/table-utils.ts` | Seat layout math (has Vitest coverage)                                                             |
| `App.tsx`              | React Router setup                                                                                      |

The Vite dev server proxies `/api/*` → `http://localhost:8080` (stripping `/api`) and `/ws` → `ws://localhost:8080`. Inside Discord the frontend derives the lobby from the activity instance id and auto-creates/joins; standalone it uses the lobby name from the home page.
