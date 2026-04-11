# Game UI Redesign — Design Spec

**Date:** 2026-04-11
**Status:** Approved

## Overview

Redesign the game screen from a flat three-zone layout into a poker-style arc layout with a playful, Exploding Kittens–branded visual identity. The backend and WebSocket communication are unchanged — this is a purely frontend visual overhaul of `client/src/pages/game/`.

---

## Layout

### Arc / Poker View

The screen is divided into three vertical zones:

1. **Opponents zone (top)** — opponents arc across the top of the screen, each with a fanned stack of face-down galaxy card backs and an emoji avatar below.
2. **Table zone (center)** — the main game area: deck pile, discard pile, turn indicator, and last-action banner.
3. **My hand zone (bottom)** — the local player's face-up hand cards and the action bar.

This mirrors the natural poker-table perspective: you're always at the bottom, opponents above.

---

## Visual Style: Playful & Expressive

- **Background:** Deep purple-black (`#0d0720`) with a radial purple glow in the center of the table zone.
- **Accent color:** Purple (`#7c3aed` / `#a78bfa`) for borders, labels, and UI chrome.
- **Active player highlight:** Gold ring + glow (`#f59e0b`) on the avatar of whoever's turn it is.
- **Avatars:** Emoji characters (🐱 😼 😸 😿) instead of "P1/P2" text circles. Emojis are assigned by cycling through a fixed list `[🐱, 😼, 😸, 😿, 🙀]` indexed by player position (0-based index into the `players` array), so the same player always gets the same emoji within a session.
- **Typography:** Tight uppercase labels with letter-spacing for labels; normal weight for names.

---

## Opponents Zone

Each opponent is rendered as a `PlayerSlot` component containing:

- **Fanned card backs** (above the avatar): N galaxy-style card backs fanned at ±16° spread, quantity matching `cardCount`. Capped at 7 visible cards to prevent overflow; additional cards stack behind.
- **Avatar circle**: 44px circle with emoji, purple border. Active player gets a gold ring + glow. Dead players are greyscale + dimmed.
- **Online dot**: Small green/grey dot in the bottom-right of the avatar.
- **Card count badge**: Small purple badge in the top-right of the avatar showing the numeric card count.
- **Player name**: Below the avatar, 10px uppercase. Active player name is gold-colored and appended with ⚡.

Dead players remain visible but desaturated and slightly transparent.

Players are distributed evenly across the top zone. For 2 players the single opponent is centered; for 3–5 players they spread left-to-right.

---

## Galaxy Card Back

The face-down card design for opponents' hands:

- **Size:** 26×38px (small, in fan) / 48×68px (deck pile)
- **Background:** Dark radial gradient (`#1e1060` → `#06020f`)
- **Border:** 1.5px solid `#6d28d9`
- **Stars:** CSS `radial-gradient` dots scattered across the card face at fixed positions
- **Center emoji:** 😼 displayed over the stars with a purple drop-shadow glow
- **Border radius:** 4px (small) / 6px (large)

---

## Table Zone (Center)

- **Felt glow:** A large radial purple gradient overlay (non-interactive) centered in the zone to evoke a table surface.
- **Turn indicator:** Pill badge showing whose turn it is, gold-colored with a pulsing dot. Hidden when game hasn't started.
- **Center piles** (side by side):
  - **Deck:** Large galaxy card back (48×68px) with a small card-count badge in the corner. Clicking it sends `DRAW_CARD`.
  - **Discard:** Shows the emoji and short name of the last discarded card type, with color matching the card's color family. Shows a placeholder when empty.
- **Last action banner:** Existing `LastActionBanner` component, restyled to match (purple border, `#c4b5fd` text).
- **Error message:** Red-bordered pill, same position as now.
- **State-specific overlays** (`FutureViewer`, `FutureReorder`, `KittenPlacer`, `FavorGiver`, `DiscardPicker`) remain as modal-style overlays — their internal styling can be updated to match the purple theme but their logic is untouched.

---

## My Hand Zone (Bottom)

- **"Your hand" label:** Small uppercase label above the cards.
- **Hand cards:** Face-up, colored by card type (existing color mapping). Size 46×68px with 6px border-radius. Show card emoji + short name.
  - **Hover:** Card lifts 8px (`translateY(-8px)`).
  - **Selected:** Card lifts 12px + white border + purple glow shadow. Multiple selection (cat combos) works the same as today.
  - **Disabled:** `opacity: 0.4`, no hover effect, `cursor: not-allowed`.
- **Card emojis:** Each `CardType` gets an assigned emoji (see mapping below).
- **Action bar:** Unchanged in behavior; restyled as pill buttons:
  - **Play** button: Purple gradient, visible when a card is selected.
  - **Draw Card** button: Blue gradient, pulses with a glow animation on the local player's turn.

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

## Files Changed

| File | Change |
|---|---|
| `client/src/pages/game/page.tsx` | Restructure layout zones; apply new background and border styles |
| `client/src/pages/game/components/PlayerCard.tsx` | Full rewrite — emoji avatar, fanned galaxy card backs, gold active ring |
| `client/src/pages/game/components/HandCard.tsx` | Add emoji, resize, update hover/selected styles |
| `client/src/pages/game/components/ActionBar.tsx` | Restyle buttons to pill shape with gradients and glow |
| `client/src/pages/game/components/LastActionBanner.tsx` | Restyle to match purple theme |
| `client/src/index.css` | Add any global CSS needed (e.g. keyframe animations) |

State-specific overlay components (`FutureViewer`, `FutureReorder`, `KittenPlacer`, `FavorGiver`, `DiscardPicker`, `GameOverOverlay`) are **not** restyled in this pass — they remain functional.

---

## Out of Scope

- Backend changes
- WebSocket protocol changes
- Home/lobby page styling
- State-specific overlay component restyling
- Responsive/mobile layout (desktop-first is fine)
