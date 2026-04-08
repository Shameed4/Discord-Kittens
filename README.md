# Discord Kittens

A real-time multiplayer Exploding Kittens clone built with a **Go WebSocket backend** and a **React + TypeScript frontend**, designed for integration with the Discord API.

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│  Client (React + TypeScript)                                 │
│  Vite dev server ──proxy──► /api/*, /ws                      │
└──────────────┬───────────────────────────────┬───────────────┘
               │ HTTP (create/join lobby)      │ WebSocket (game actions + state)
               ▼                               ▼
┌──────────────────────────────────────────────────────────────┐
│  Server (Go)                                                 │
│                                                              │
│  ┌─────────┐    ┌───────────────────────────────────────┐    │
│  │ HTTP    │    │  Lobby (per-game instance)            │    │
│  │ Router  │───►│                                       │    │
│  │         │    │  ActionQueue ◄── player actions       │    │
│  │ /lobby  │    │  JoinQueue   ◄── new connections      │    │
│  │ /ws     │    │                                       │    │
│  └─────────┘    │  run() event loop (select on chans)   │    │
│                 │       │                               │    │
│                 │       ├─ action.go  (card logic)      │    │
│                 │       ├─ cards.go   (deck/dealing)    │    │
│                 │       └─ turns.go   (state broadcast) │    │
│                 └───────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────┘
```

## Backend (Go)

The backend is a concurrent WebSocket game server built entirely with Go's standard library and [Gorilla WebSocket](https://github.com/gorilla/websocket). No frameworks -- just channels, goroutines, and a clean state machine.

### Concurrency Model

Each lobby runs its own goroutine with a **channel-based event loop**:

```go
select {
case joinReq := <-lobby.JoinQueue:   // new player connected
case action  := <-lobby.ActionQueue: // player performed an action
}
```

All state mutations flow through these channels, so no mutexes are needed -- the event loop serializes access naturally. Each connected player gets their own `Send chan GameState`, and the server pushes personalized state snapshots (hiding other players' hands) after every action.

### State Machine

The game uses an 8-state turn machine to enforce valid transitions:

```
NotStarted ──► Normal ──► SeeingTheFuture
                  │          AlteringTheFuture
                  │          AwaitingKittenPlacement
                  │          AwaitingFavor
                  │          AwaitingDiscardTake
                  └──────► GameOver
```

Every action is guarded by `assertTurnAndState()`, which validates both whose turn it is and which state transitions are legal. This prevents invalid game states at the protocol level.

### Card System

16 card types with a dynamic deck scaled to player count:

| Category | Cards                                                                                             | Multiplier                                  |
| -------- | ------------------------------------------------------------------------------------------------- | ------------------------------------------- |
| Danger   | Exploding Kitten, Defuse                                                                          | n-1 kittens, 1 per player + 2 extra defuses |
| Action   | Skip, Attack, Targeted Attack, Shuffle, Draw From Bottom, See The Future, Alter The Future, Favor | 2x per player                               |
| Combo    | Cat 1-4, Feral Cat                                                                                | 4x per player                               |

### Action Handling

The action handler (`action.go`) processes 9 distinct action types through a switch dispatch:

- **StartGame** -- initialize deck, deal hands, insert kittens
- **DrawCard / DrawFromBottom** -- draw with Exploding Kitten resolution
- **PlayCard** -- execute card effects (skip turns, force attacks, peek/rearrange deck, etc.)
- **Combo** -- 2-card (steal random), 3-card (steal named), and 5-card (take from discard) combinations
- **PlaceKitten** -- reinsert a defused Exploding Kitten at any deck position
- **AlterFuture** -- submit a rearranged ordering of the top 3 cards
- **GiveFavor / Disconnect** -- respond to favor requests, handle disconnections

### Combo Validation

The combo system supports three tiers with strict validation:

- **2 matching cards** -- steal a random card from a target player
- **3 matching cards** -- name a specific card to steal from a target
- **5 unique cards** -- take any card from the discard pile

Feral Cat acts as a wildcard that can substitute for any cat type in 2 and 3-card combos.

## Frontend (React + TypeScript)

> **Status: Early development.** The lobby flow and real-time WebSocket connection are working. Game UI for playing cards, selecting combos, and special interactions is not yet built.

**Stack:** React 19, TypeScript, Vite, TailwindCSS, React Router v7

What's implemented:

- **Lobby screen** -- create or join a game by name
- **WebSocket connection** -- established on game page mount, receives live `GameState` updates
- **Player display** -- shows each player's card count, online status, turn indicator, and elimination state
- **Type definitions** -- `GameState`, `ActionRequest`, `CardType`, `TurnState` models mirroring the backend

What's next:

- Card hand display and play UI
- Combo card selection interface
- Exploding Kitten placement picker
- Alter The Future drag-to-reorder
- Favor card selection prompt
- Discord OAuth and bot command integration

## Tech Stack

| Layer     | Technology               |
| --------- | ------------------------ |
| Backend   | Go 1.25                  |
| WebSocket | Gorilla WebSocket 1.5    |
| Frontend  | React 19, TypeScript 5.9 |
| Build     | Vite 8                   |
| Styling   | TailwindCSS 4            |
| Routing   | React Router 7           |

## Getting Started

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
# Vite dev server proxies /api and /ws to :8080
```

Open the frontend, create a lobby, and share the lobby name with other players to join.

## Project Structure

```
game/
  server.go      HTTP endpoints + WebSocket upgrade
  lobby.go       Lobby state, player management, event loop
  action.go      Card action dispatch and game logic
  cards.go       Card definitions, deck generation, dealing
  turns.go       Turn rotation, state broadcasting, game state serialization

client/
  src/
    pages/home/       Lobby creation and join UI
    pages/game/       Game screen + PlayerCard component
    models/           TypeScript types mirroring backend state
```
