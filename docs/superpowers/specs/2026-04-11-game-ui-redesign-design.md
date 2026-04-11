# Game UI Redesign — Design Spec

**Date:** 2026-04-11
**Status:** Approved

## Overview

Redesign the game screen from a flat three-zone layout into a round-table layout with a playful, Exploding Kittens–branded visual identity. Supports 2–10 players and mobile viewports. The backend and WebSocket communication are unchanged — this is a purely frontend visual overhaul of `client/src/pages/game/`.

---

## Layout

### Round Table

The screen is a single full-viewport canvas. A circular felt table is centered on screen and expands to fill the available space (capped at a sensible max on large displays). Players sit at clock positions around the table perimeter:

- **You (local player):** always at 6 o'clock, outside the table edge at the bottom. Your hand cards fan out below your avatar.
- **Opponents:** distributed evenly around the remaining 300° of the circle (roughly 7:30 through 4:30, skipping 6 o'clock). With 1 opponent they sit at 12 o'clock; with 9 opponents they space ~33° apart.
- **Deck + discard piles:** centered on the table felt.
- **Turn indicator + last-action banner:** also centered, above the piles.

On mobile the table circle shrinks to fit the viewport width with adequate padding. Seat elements (avatar, card fan, name) scale down proportionally via a CSS custom property `--seat-scale` that is set based on viewport width and player count.

---

## Visual Style: Playful & Expressive

- **Background:** Deep purple-black (`#0d0720`).
- **Table felt:** Green radial gradient (`#1a5c2a` → `#0e3d1a`) with a gold border (`#8B6914`), circular.
- **Accent color:** Purple (`#7c3aed` / `#a78bfa`) for borders, labels, and UI chrome.
- **Active player highlight:** Gold ring + glow (`#f59e0b`) on the avatar of whoever's turn it is.
- **Avatars:** Emoji characters instead of "P1/P2" text circles. Emojis are assigned by cycling through `[🐱, 😼, 😸, 😿, 🙀, 😺, 🐈, 🐈‍⬛, 😻, 🙈]` indexed by position in the `players` array (0-based), giving each player a consistent emoji within a session. The local player always shows 😎.
- **Typography:** Tight uppercase labels with letter-spacing for UI labels; normal weight for player names.

---

## Player Seat Component

Each player (including you) is a `PlayerSeat` component. All sizing uses `--seat-scale` so the entire component shrinks uniformly at high player counts or on small screens.

Base sizes (at `--seat-scale: 1`):

| Element | Size |
|---|---|
| Avatar circle | 44px |
| Card back (in fan) | 26×38px |
| Font size (name) | 10px |

Scale breakpoints:

| Player count | `--seat-scale` |
|---|---|
| 2–4 | 1.0 |
| 5–7 | 0.85 |
| 8–10 | 0.7 |

On mobile (viewport < 480px), `--seat-scale` is additionally multiplied by `0.8`.

### Opponent seat (all players except you)

- **Fanned card backs** (above the avatar): N galaxy-style card backs fanned at ±16° spread, quantity matching `cardCount`. Capped at 7 visible cards; additional cards stack behind without increasing the fan width.
- **Avatar circle:** Emoji, purple border. Active player gets gold ring + glow. Dead players are greyscale + dimmed (`opacity: 0.5`).
- **Online dot:** Small green/grey dot bottom-right of avatar.
- **Card count badge:** Small purple badge top-right of avatar showing the numeric `cardCount`.
- **Player name:** Below avatar, uppercase. Active player name is gold + ⚡ appended.

### Your seat (6 o'clock)

- Avatar sits just outside the table edge.
- **Hand cards** fan out in a row below the avatar (not above like opponents). Cards are face-up, full size (46×68px at scale 1), horizontally scrollable if they overflow on narrow screens.
- **Action bar** sits below the hand row.

---

## Galaxy Card Back

The face-down card shared by opponent fans and the deck pile:

- **Background:** Dark radial gradient (`#1e1060` → `#06020f`)
- **Border:** 1.5px solid `#6d28d9`
- **Stars:** CSS `radial-gradient` dots at fixed percentage positions across the face
- **Center emoji:** 😼 with a purple `drop-shadow` glow
- **Border radius:** 4px (fan size) / 6px (deck pile size)

---

## Table Center

- **Green felt circle** with gold border — the table background.
- **Subtle radial purple glow** overlay on top of the felt (non-interactive), centered, adds depth.
- **Turn indicator:** Gold pill badge with pulsing dot and player name. Hidden before game starts.
- **Center piles** (side by side, centered on table):
  - **Deck:** Galaxy card back at 48×68px, card-count badge in corner. Clicking sends `DRAW_CARD`.
  - **Discard:** Shows emoji + short name of the top discard, colored by card type. Placeholder when empty.
- **Last action banner:** Purple-bordered pill, `#c4b5fd` text. Positioned above the piles.
- **Error message:** Red-bordered pill.
- **State-specific overlays** (`FutureViewer`, `FutureReorder`, `KittenPlacer`, `FavorGiver`, `DiscardPicker`) render as centered overlays on top of the table — their logic is unchanged, styling may be lightly updated to match the purple theme.

---

## My Hand Zone

- **Hand cards:** Face-up, colored by card type. Size 46×68px, 6px border-radius. Show card emoji + short name.
  - **Hover:** Lifts 8px.
  - **Selected:** Lifts 12px + white border + purple glow shadow.
  - **Disabled:** `opacity: 0.4`, no hover, `cursor: not-allowed`.
- **Action bar:** Pill-shaped buttons below the hand:
  - **Play** button: Purple gradient, shown when a card is selected.
  - **Draw Card** button: Blue gradient, pulses with glow animation on your turn.

### Card Emoji Mapping

| CardType | Emoji |
|---|---|
| DEFUSE | 🔧 |
| EXPLODING_KITTEN | 💥 |
| SKIP | ⏭️ |
| ATTACK | ⚡ |
| TARGETED_ATTACK | 🎯 |
| CAT1 | 🐾 |
| CAT2 | 🌮 |
| CAT3 | 🥔 |
| CAT4 | 🐟 |
| FERAL_CAT | 🌈 |
| SEE_THE_FUTURE | 🔮 |
| ALTER_THE_FUTURE | ✨ |
| SHUFFLE | 🔀 |
| DRAW_FROM_BOTTOM | ⬇️ |
| FAVOR | 🎁 |

---

## Responsive / Mobile

- The table circle is sized as `min(90vw, 90vh - hand-height)` so it always fits without scrolling.
- `--seat-scale` is computed from player count × viewport width (see breakpoints above).
- On narrow screens (< 480px), card names in the hand are hidden (emoji only) to save space.
- No horizontal page scroll — everything fits within the viewport.

---

## Files Changed

| File | Change |
|---|---|
| `client/src/pages/game/page.tsx` | Full layout rewrite — round table canvas, seat positioning math |
| `client/src/pages/game/components/PlayerCard.tsx` | Rename → `PlayerSeat.tsx`; full rewrite with scale support, fanned card backs |
| `client/src/pages/game/components/HandCard.tsx` | Add emoji, update hover/selected styles, hide name on mobile |
| `client/src/pages/game/components/ActionBar.tsx` | Restyle buttons to pill shape with gradients and glow |
| `client/src/pages/game/components/LastActionBanner.tsx` | Restyle to match purple theme |
| `client/src/index.css` | Keyframe animations (`pulse`, `glow-pulse`), CSS custom property defaults |

State-specific overlay components (`FutureViewer`, `FutureReorder`, `KittenPlacer`, `FavorGiver`, `DiscardPicker`, `GameOverOverlay`) are **not** structurally changed — light purple theme polish only if time allows.

---

## Out of Scope

- Backend changes
- WebSocket protocol changes
- Home/lobby page styling
- Full restyle of state-specific overlay components
