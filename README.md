# Discord Kittens

A real-time multiplayer Exploding Kittens clone built with a **Go WebSocket backend** and a **React + TypeScript frontend**. It runs as a **Discord activity** (embedded app) and also works standalone in a browser.

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│  Client (React + TypeScript)                                 │
│  Discord embedded-app SDK · Vite dev server ──proxy──► /api/* │
└──────────────┬───────────────────────────────┬───────────────┘
               │ HTTP (lobby, OAuth token)     │ WebSocket (game actions + state)
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
│  │ /token  │    │  run() event loop (select on chans)   │    │
│  └─────────┘    │       │                               │    │
│                 │       ├─ action.go  (card logic)      │    │
│                 │       ├─ cards.go   (deck/dealing)    │    │
│                 │       └─ turns.go   (state broadcast) │    │
│                 └───────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────┘
```

The `/token` endpoint (in `discord.go`) exchanges a Discord OAuth2 authorization code for an access token, so the client can identify the player without exposing the client secret.

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
- **Resilient connection** -- auto-reconnect with exponential backoff + jitter, reconnecting to the same seat by Discord user id
- **Browser fallback** -- create or join a lobby by name when running outside Discord

## Tech Stack

| Layer     | Technology                   |
| --------- | ---------------------------- |
| Backend   | Go 1.25                      |
| WebSocket | Gorilla WebSocket 1.5        |
| Frontend  | React 19, TypeScript 5.9     |
| Discord   | Discord Embedded App SDK 2.5 |
| Build     | Vite 8                       |
| Styling   | TailwindCSS 4                |
| Routing   | React Router 7               |

## Getting Started

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

## Project Structure

```
game/
  server.go      HTTP endpoints (lobby, ws, token) + WebSocket upgrade & keepalive
  lobby.go       Lobby state, join/reconnect/disconnect, event loop
  action.go      Card action dispatch and game logic
  cards.go       Card definitions, tiered deck generation, dealing
  turns.go       Turn rotation, game log, state broadcasting & serialization
  discord.go     Discord OAuth2 token exchange (/token)

client/
  src/
    discord/sdk.ts    Discord embedded-app SDK setup + OAuth handshake
    pages/home/       Browser-fallback lobby creation and join UI
    pages/game/       Game screen, WS lifecycle, table layout, and interaction components
    models/           TypeScript types mirroring backend state
```
