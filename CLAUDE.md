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
```

## Architecture

A real-time multiplayer Exploding Kittens clone. The backend is a Go WebSocket server; the frontend is React + TypeScript + Vite. There is no REST API for game actions — all game communication after joining goes through a single WebSocket connection.

### Backend (`game/`)

| File        | Responsibility                                                                                                          |
| ----------- | ----------------------------------------------------------------------------------------------------------------------- |
| `server.go` | HTTP handlers (`POST /lobby`, `GET /ws`), WebSocket upgrade, global `lobbies` map (guarded by `lobbiesMutex`)           |
| `lobby.go`  | `Lobby` struct and `run()` event loop, `TurnState` enum, `GameState`/`PlayerAction` types, player join/disconnect logic |
| `action.go` | `takePlayerAction()` switch dispatch for all 9 action types                                                             |
| `cards.go`  | `Card` type, all 16 card definitions, deck generation (`multipliers`, `ExtraDefuses`), shuffle/deal                     |
| `turns.go`  | Turn rotation (`setNextPlayerTurn`), `assertTurnAndState()` guard, `broadcastGameState()` / `sendError()`               |

**Concurrency model:** Each `Lobby` runs a single goroutine with a `select` loop over two channels — `JoinQueue chan JoinRequest` and `ActionQueue chan PlayerAction`. No mutexes inside a lobby; all state mutation is serialized by the event loop. Each player has a `Send chan GameState` that the server goroutine writes to.

**State machine:** `TurnState` has 8 values (`NotStarted`, `Normal`, `SeeingTheFuture`, `AlteringTheFuture`, `AwaitingKittenPlacement`, `AwaitingFavor`, `AwaitingDiscardTake`, `GameOver`). Every action calls `assertTurnAndState()` with a list of valid states before mutating anything.

**GameState is personalized:** `getGameState(playerIdx)` builds a snapshot for each player individually — other players' hands are hidden (only `cardCount` exposed), and `future`/`discardOptions` are only populated for the active player in the relevant states.

**Lobby lifecycle:** A hardcoded `"test"` lobby is created on startup. `POST /lobby` creates named lobbies. On game over (all players disconnect or one player remains), the lobby's `turnState` becomes `GameOver` but the lobby entry persists in the `lobbies` map.

### Frontend (`client/src/`)

| Path          | Responsibility                                                                                                |
| ------------- | ------------------------------------------------------------------------------------------------------------- |
| `models/`     | TypeScript types mirroring backend: `GameState`, `ActionRequest`, `CardType`, `TurnState`, `ConnectionStatus` |
| `pages/home/` | Lobby creation and join UI                                                                                    |
| `pages/game/` | Game screen, WebSocket connection lifecycle, `PlayerCard` component                                           |
| `App.tsx`     | React Router setup                                                                                            |

The Vite dev server proxies `/api/*` → `http://localhost:8080` (stripping `/api`) and `/ws` → `ws://localhost:8080`. The frontend connects to the WebSocket at `/ws?lobby=<name>`.

**Frontend status:** Early development. Lobby flow and live `GameState` display work. Card play UI, combo selection, and special interaction prompts are not yet built.
