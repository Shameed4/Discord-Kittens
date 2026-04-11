# Game UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the game screen to a round-table layout with a playful purple/galaxy aesthetic, supporting 2–10 players and mobile viewports.

**Architecture:** Pure frontend visual overhaul. A new `PlayerSeat` component replaces `PlayerCard` and renders at CSS clock positions around a circle. A `table-utils.ts` utility computes percentage-based seat coordinates from player count and local player index. All WebSocket, state management, and game logic in `page.tsx` are unchanged — only the JSX is replaced.

**Tech Stack:** React 19, TypeScript 5.9, Tailwind CSS v4, Vite 8. Tests via Vitest (added in Task 1).

---

## File Map

| Status | File | Change |
|---|---|---|
| Modify | `client/vite.config.ts` | Add Vitest config block |
| Modify | `client/package.json` | Add vitest dev dep + test script |
| Modify | `client/src/index.css` | Add `@keyframes` for pulse-dot and glow-pulse |
| Create | `client/src/pages/game/table-utils.ts` | Seat position math |
| Create | `client/src/pages/game/table-utils.test.ts` | Vitest unit tests |
| Create | `client/src/pages/game/components/PlayerSeat.tsx` | New player seat (emoji avatar + card fan) |
| Delete | `client/src/pages/game/components/PlayerCard.tsx` | Replaced by PlayerSeat |
| Modify | `client/src/pages/game/components/HandCard.tsx` | Emoji, gradient colors, new sizing |
| Modify | `client/src/pages/game/components/ActionBar.tsx` | Pill buttons, purple/blue theme |
| Modify | `client/src/pages/game/components/LastActionBanner.tsx` | Purple pill restyle |
| Modify | `client/src/pages/game/page.tsx` | Round table layout, seat positioning |

---

## Task 1: Vitest setup + CSS animations

**Files:**
- Modify: `client/vite.config.ts`
- Modify: `client/package.json`
- Modify: `client/src/index.css`

- [ ] **Step 1: Install Vitest**

```bash
cd client && npm install -D vitest jsdom
```

Expected: `vitest` and `jsdom` appear in `package.json` devDependencies.

- [ ] **Step 2: Add test config to `vite.config.ts`**

```ts
// client/vite.config.ts
/// <reference types="vitest" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  test: {
    environment: 'jsdom',
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
    },
  },
});
```

- [ ] **Step 3: Add test script to `package.json`**

Add `"test": "vitest run"` to the `scripts` block:

```json
"scripts": {
  "dev": "vite",
  "build": "tsc -b && vite build",
  "lint": "eslint .",
  "preview": "vite preview",
  "test": "vitest run"
}
```

- [ ] **Step 4: Replace `client/src/index.css` with keyframes**

```css
@import "tailwindcss";

@keyframes pulse-dot {
  0%, 100% { opacity: 1; transform: scale(1); }
  50%       { opacity: 0.4; transform: scale(0.7); }
}

@keyframes glow-pulse {
  0%, 100% { box-shadow: 0 4px 12px rgba(29, 78, 216, 0.4); }
  50%      { box-shadow: 0 4px 24px rgba(29, 78, 216, 0.9), 0 0 32px rgba(29, 78, 216, 0.3); }
}

.animate-pulse-dot {
  animation: pulse-dot 1.2s ease-in-out infinite;
}

.animate-glow-pulse {
  animation: glow-pulse 2s ease-in-out infinite;
}
```

- [ ] **Step 5: Verify Vitest runs (no tests yet)**

```bash
cd client && npm test
```

Expected output: `No test files found` or `0 tests passed` — no errors.

- [ ] **Step 6: Commit**

```bash
cd client
git add vite.config.ts package.json package-lock.json src/index.css
git commit -m "chore: add vitest and CSS animation keyframes"
```

---

## Task 2: Seat position utility

**Files:**
- Create: `client/src/pages/game/table-utils.ts`
- Create: `client/src/pages/game/table-utils.test.ts`

- [ ] **Step 1: Write the failing tests**

```ts
// client/src/pages/game/table-utils.test.ts
import { describe, it, expect } from 'vitest';
import { getSeatPositions } from './table-utils';

describe('getSeatPositions', () => {
  it('places local player at 6 o\'clock — bottom center (x=50, y=100)', () => {
    const pos = getSeatPositions(2, 0);
    expect(pos[0].x).toBeCloseTo(50, 1);
    expect(pos[0].y).toBeCloseTo(100, 1);
  });

  it('places single opponent at 12 o\'clock — top center (x=50, y=0)', () => {
    const pos = getSeatPositions(2, 0);
    expect(pos[1].x).toBeCloseTo(50, 1);
    expect(pos[1].y).toBeCloseTo(0, 1);
  });

  it('distributes 3 opponents evenly over the 300° arc (4-player game)', () => {
    // local=0 at 180°; opponents at 210°, 360°, 510°
    const pos = getSeatPositions(4, 0);
    // 210° → x≈25, y≈93.3
    expect(pos[1].x).toBeCloseTo(25, 1);
    expect(pos[1].y).toBeCloseTo(93.3, 1);
    // 360° → x=50, y=0
    expect(pos[2].x).toBeCloseTo(50, 1);
    expect(pos[2].y).toBeCloseTo(0, 1);
    // 510° (=150°) → x≈75, y≈93.3
    expect(pos[3].x).toBeCloseTo(75, 1);
    expect(pos[3].y).toBeCloseTo(93.3, 1);
  });

  it('works when local player is not at array index 0', () => {
    const pos = getSeatPositions(2, 1);
    expect(pos[1].x).toBeCloseTo(50, 1);  // local at bottom
    expect(pos[1].y).toBeCloseTo(100, 1);
    expect(pos[0].x).toBeCloseTo(50, 1);  // opponent at top
    expect(pos[0].y).toBeCloseTo(0, 1);
  });

  it('handles 10 players — all positions are within 0–100%', () => {
    const pos = getSeatPositions(10, 0);
    expect(pos).toHaveLength(10);
    pos.forEach(p => {
      expect(p.x).toBeGreaterThanOrEqual(-1);   // allow floating point at edge
      expect(p.x).toBeLessThanOrEqual(101);
      expect(p.y).toBeGreaterThanOrEqual(-1);
      expect(p.y).toBeLessThanOrEqual(101);
    });
  });
});
```

- [ ] **Step 2: Run tests — expect all to fail**

```bash
cd client && npm test
```

Expected: 5 failing tests with `Cannot find module './table-utils'`.

- [ ] **Step 3: Implement `table-utils.ts`**

```ts
// client/src/pages/game/table-utils.ts

export interface SeatPosition {
  /** Percentage left (0–100) within the table container */
  x: number;
  /** Percentage top (0–100) within the table container */
  y: number;
}

// Opponents span a 300° arc from 7:30 (210°) clockwise to 4:30 (510°=150°).
// The local player is pinned at 6 o'clock (180°).
const OPPONENT_START_DEG = 210;
const OPPONENT_ARC_DEG = 300;

function degToPos(angleDeg: number): SeatPosition {
  const rad = (angleDeg * Math.PI) / 180;
  return {
    x: 50 + 50 * Math.sin(rad),
    y: 50 - 50 * Math.cos(rad),
  };
}

/**
 * Returns a SeatPosition for every player, indexed by position in the
 * players array. The local player is always at 6 o'clock (bottom).
 * Opponents are evenly distributed over the remaining 300° arc.
 *
 * @param playerCount  total players (2–10)
 * @param localPlayerIndex  index of the local player in the players array
 */
export function getSeatPositions(
  playerCount: number,
  localPlayerIndex: number,
): SeatPosition[] {
  const positions: SeatPosition[] = new Array(playerCount);

  positions[localPlayerIndex] = degToPos(180);

  const opponentCount = playerCount - 1;
  for (let i = 0; i < opponentCount; i++) {
    const playerIdx = (localPlayerIndex + 1 + i) % playerCount;
    const angleDeg =
      opponentCount === 1
        ? 0  // single opponent → 12 o'clock
        : OPPONENT_START_DEG + (i * OPPONENT_ARC_DEG) / (opponentCount - 1);
    positions[playerIdx] = degToPos(angleDeg);
  }

  return positions;
}
```

- [ ] **Step 4: Run tests — expect all to pass**

```bash
cd client && npm test
```

Expected: `5 tests passed`.

- [ ] **Step 5: Commit**

```bash
git add client/src/pages/game/table-utils.ts client/src/pages/game/table-utils.test.ts
git commit -m "feat: add seat position utility with vitest coverage"
```

---

## Task 3: PlayerSeat component

**Files:**
- Create: `client/src/pages/game/components/PlayerSeat.tsx`
- Delete: `client/src/pages/game/components/PlayerCard.tsx`

- [ ] **Step 1: Create `PlayerSeat.tsx`**

```tsx
// client/src/pages/game/components/PlayerSeat.tsx
import type { CSSProperties } from 'react';
import type { GameState, PlayerState } from '../../../models/game-state';

// One emoji per player slot, cycled by array index. Local player always gets 😎.
const PLAYER_EMOJIS = ['🐱', '😼', '😸', '😿', '🙀', '😺', '🐈', '🐈‍⬛', '😻', '🙈'];
const LOCAL_EMOJI = '😎';

export interface PlayerSeatProps {
  playerState: PlayerState;
  gameState: GameState;
  /** Index in gameState.players — used for emoji assignment */
  playerIndex: number;
  /** 1.0 for ≤4 players, 0.85 for 5–7, 0.7 for 8–10 */
  seatScale: number;
  /** Absolute positioning injected by the parent (left/top/transform) */
  style?: CSSProperties;
}

export default function PlayerSeat({
  playerState,
  gameState,
  playerIndex,
  seatScale,
  style,
}: PlayerSeatProps) {
  const { id, cardCount, isAlive, isOnline } = playerState;
  const isLocal = id === gameState.playerId;
  const isTurn = id === gameState.turnId && isAlive && gameState.inProgress;

  const avatarPx  = Math.round(44 * seatScale);
  const emojiFpx  = Math.round(22 * seatScale);
  const nameFpx   = Math.round(10 * seatScale);
  const dotPx     = Math.round(10 * seatScale);
  const badgePx   = Math.round(17 * seatScale);
  const gapPx     = Math.round(4 * seatScale);

  const emoji = isLocal
    ? LOCAL_EMOJI
    : PLAYER_EMOJIS[playerIndex % PLAYER_EMOJIS.length];

  const borderColor = isTurn ? '#f59e0b' : isLocal ? '#818cf8' : '#6d28d9';
  const boxShadow = isTurn
    ? '0 0 0 3px rgba(245,158,11,0.35), 0 0 16px rgba(245,158,11,0.4)'
    : isLocal
    ? '0 0 0 2px rgba(129,140,248,0.3)'
    : 'none';

  const nameColor = !isAlive
    ? '#4b5563'
    : isTurn
    ? '#f59e0b'
    : isLocal
    ? '#818cf8'
    : '#a78bfa';

  return (
    <div style={{
      ...style,
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      gap: gapPx,
      opacity: isAlive ? 1 : 0.5,
    }}>
      {/* Galaxy card fan — opponents only, game in progress */}
      {!isLocal && gameState.inProgress && (
        <CardFan cardCount={cardCount} seatScale={seatScale} />
      )}

      {/* Avatar + online dot + card count badge */}
      <div style={{ position: 'relative' }}>
        <div style={{
          width: avatarPx,
          height: avatarPx,
          borderRadius: '50%',
          background: 'radial-gradient(circle at 35% 35%, #2e1065, #0d0520)',
          border: `2px solid ${borderColor}`,
          boxShadow,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: emojiFpx,
          filter: !isAlive ? 'grayscale(1) brightness(0.5)' : 'none',
        }}>
          {emoji}
        </div>

        {/* Online indicator */}
        <div style={{
          position: 'absolute',
          bottom: 0,
          right: 0,
          width: dotPx,
          height: dotPx,
          borderRadius: '50%',
          background: isOnline ? '#22c55e' : '#6b7280',
          border: `${Math.round(2 * seatScale)}px solid #0d0720`,
        }} />

        {/* Card count badge (opponents only) */}
        {!isLocal && isAlive && gameState.inProgress && (
          <div style={{
            position: 'absolute',
            top: -Math.round(6 * seatScale),
            right: -Math.round(6 * seatScale),
            width: badgePx,
            height: badgePx,
            borderRadius: '50%',
            background: '#7c3aed',
            border: `${Math.round(1.5 * seatScale)}px solid #a78bfa`,
            color: 'white',
            fontSize: Math.round(9 * seatScale),
            fontWeight: 800,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}>
            {cardCount}
          </div>
        )}
      </div>

      {/* Player name */}
      <div style={{
        fontSize: nameFpx,
        fontWeight: 700,
        color: nameColor,
        textAlign: 'center',
        whiteSpace: 'nowrap',
        letterSpacing: '0.3px',
      }}>
        {isLocal ? 'You' : `Player ${id}`}
        {isTurn ? ' ⚡' : ''}
      </div>
    </div>
  );
}

// ─── Galaxy card fan ──────────────────────────────────────────────────────────

function CardFan({ cardCount, seatScale }: { cardCount: number; seatScale: number }) {
  const visible = Math.min(cardCount, 7);

  const cardW = Math.round(26 * seatScale);
  const cardH = Math.round(38 * seatScale);
  const maxAngleDeg = 16;
  const maxAngleRad = (maxAngleDeg * Math.PI) / 180;

  // Container wide enough to hold fanned cards without clipping
  const spread = Math.round(cardH * Math.sin(maxAngleRad));
  const containerW = cardW + 2 * spread + Math.round(4 * seatScale);
  const containerH = cardH + Math.round(6 * seatScale);

  if (visible === 0) return <div style={{ height: containerH }} />;

  const angles = Array.from({ length: visible }, (_, i) =>
    visible === 1 ? 0 : -maxAngleDeg + (i * 2 * maxAngleDeg) / (visible - 1),
  );

  return (
    <div style={{ position: 'relative', width: containerW, height: containerH }}>
      {angles.map((angle, i) => (
        <div
          key={i}
          style={{
            position: 'absolute',
            bottom: 0,
            left: '50%',
            marginLeft: -cardW / 2,
            width: cardW,
            height: cardH,
            transform: `rotate(${angle}deg)`,
            transformOrigin: 'bottom center',
            borderRadius: Math.round(4 * seatScale),
            background: 'radial-gradient(ellipse at 35% 35%, #1e1060 0%, #06020f 100%)',
            border: '1.5px solid #6d28d9',
            boxShadow: '1px 1px 4px rgba(0,0,0,0.6)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: Math.round(13 * seatScale),
            zIndex: i,
          }}
        >
          😼
        </div>
      ))}
    </div>
  );
}
```

- [ ] **Step 2: Delete `PlayerCard.tsx`**

```bash
git rm client/src/pages/game/components/PlayerCard.tsx
```

- [ ] **Step 3: TypeScript check**

```bash
cd client && npx tsc --noEmit
```

Expected: no errors (PlayerCard import in page.tsx will error — leave it for Task 7).

- [ ] **Step 4: Commit**

```bash
git add client/src/pages/game/components/PlayerSeat.tsx
git commit -m "feat: add PlayerSeat with emoji avatar, galaxy card fan, scale support"
```

---

## Task 4: HandCard updates

**Files:**
- Modify: `client/src/pages/game/components/HandCard.tsx`

- [ ] **Step 1: Rewrite `HandCard.tsx`**

```tsx
// client/src/pages/game/components/HandCard.tsx
import { isCatCard, type CardType } from '../../../models/game-enums';

interface HandCardProps {
  card: CardType;
  index: number;
  isSelected: boolean;
  isPlayable: boolean;
  onClick: (index: number) => void;
}

const CARD_BG: Record<CardType, string> = {
  DEFUSE:           'linear-gradient(135deg, #166534, #14532d)',
  EXPLODING_KITTEN: 'linear-gradient(135deg, #991b1b, #7f1d1d)',
  SKIP:             'linear-gradient(135deg, #1d4ed8, #1e3a8a)',
  ATTACK:           'linear-gradient(135deg, #c2410c, #9a3412)',
  TARGETED_ATTACK:  'linear-gradient(135deg, #b45309, #92400e)',
  CAT1:             'linear-gradient(135deg, #7e22ce, #4c1d95)',
  CAT2:             'linear-gradient(135deg, #7e22ce, #4c1d95)',
  CAT3:             'linear-gradient(135deg, #7e22ce, #4c1d95)',
  CAT4:             'linear-gradient(135deg, #7e22ce, #4c1d95)',
  CAT5:             'linear-gradient(135deg, #7e22ce, #4c1d95)',
  FERAL_CAT:        'linear-gradient(135deg, #6b21a8, #3b0764)',
  SEE_THE_FUTURE:   'linear-gradient(135deg, #0e7490, #164e63)',
  ALTER_THE_FUTURE: 'linear-gradient(135deg, #0369a1, #1e3a8a)',
  SHUFFLE:          'linear-gradient(135deg, #a16207, #713f12)',
  DRAW_FROM_BOTTOM: 'linear-gradient(135deg, #0f766e, #134e4a)',
  FAVOR:            'linear-gradient(135deg, #be185d, #9d174d)',
};

const CARD_EMOJI: Record<CardType, string> = {
  DEFUSE:           '🔧',
  EXPLODING_KITTEN: '💥',
  SKIP:             '⏭️',
  ATTACK:           '⚡',
  TARGETED_ATTACK:  '🎯',
  CAT1:             '🐾',
  CAT2:             '🌮',
  CAT3:             '🥔',
  CAT4:             '🐟',
  CAT5:             '🌈',
  FERAL_CAT:        '🦄',
  SEE_THE_FUTURE:   '🔮',
  ALTER_THE_FUTURE: '✨',
  SHUFFLE:          '🔀',
  DRAW_FROM_BOTTOM: '⬇️',
  FAVOR:            '🎁',
};

const SHORT_NAME: Record<CardType, string> = {
  DEFUSE:           'Defuse',
  EXPLODING_KITTEN: 'Bomb!',
  SKIP:             'Skip',
  ATTACK:           'Attack',
  TARGETED_ATTACK:  'Target',
  CAT1:             'Taco',
  CAT2:             'Beard',
  CAT3:             'Potato',
  CAT4:             'Melon',
  CAT5:             'Rainbow',
  FERAL_CAT:        'Feral',
  SEE_THE_FUTURE:   'Future',
  ALTER_THE_FUTURE: 'Alter',
  SHUFFLE:          'Shuffle',
  DRAW_FROM_BOTTOM: 'Bottom',
  FAVOR:            'Favor',
};

export default function HandCard({ card, index, isSelected, isPlayable, onClick }: HandCardProps) {
  const isCat = isCatCard(card);

  return (
    <button
      onClick={() => isPlayable && onClick(index)}
      disabled={!isPlayable}
      style={{
        background: CARD_BG[card] ?? 'linear-gradient(135deg, #374151, #1f2937)',
        transform: isSelected ? 'translateY(-12px)' : undefined,
        boxShadow: isSelected
          ? '0 8px 20px rgba(167,139,250,0.5), 0 0 0 2px white'
          : '0 4px 12px rgba(0,0,0,0.5)',
      }}
      className={[
        'relative flex flex-col items-center justify-center gap-0.5',
        'w-[46px] h-[66px] rounded-lg border-2 text-white shrink-0 select-none',
        'transition-all duration-150',
        isSelected ? 'border-white' : 'border-white/20',
        isPlayable
          ? 'cursor-pointer hover:-translate-y-2'
          : 'opacity-40 cursor-not-allowed',
      ].join(' ')}
    >
      <span className="text-lg leading-none">{CARD_EMOJI[card]}</span>
      <span className="hidden sm:block text-[7px] font-black uppercase tracking-tight px-0.5 leading-tight text-center text-white/90">
        {SHORT_NAME[card]}
      </span>
      {isCat && (
        <span className="absolute bottom-0.5 text-[6px] font-black uppercase tracking-widest opacity-60">
          cat
        </span>
      )}
    </button>
  );
}
```

- [ ] **Step 2: TypeScript check**

```bash
cd client && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add client/src/pages/game/components/HandCard.tsx
git commit -m "feat: update HandCard with emoji, gradient bg, mobile-friendly sizing"
```

---

## Task 5: ActionBar restyle

**Files:**
- Modify: `client/src/pages/game/components/ActionBar.tsx`

**Steps:**

- [ ] **Step 1: Replace only the three `return` statements in `ActionBar.tsx` — all logic above remains identical**

The three returns are at lines ~31, ~53, and ~114 in the original. Replace each:

**Not-started return (replace the block starting `if (!inProgress)`):**

```tsx
if (!inProgress) {
  return (
    <div className="mt-4 flex justify-center">
      <button
        onClick={() => onAction({ action: 'START_GAME' })}
        className="rounded-full bg-gradient-to-r from-green-600 to-emerald-700 px-8 py-2.5 font-bold text-white shadow-lg border border-green-400/30 hover:from-green-500 hover:to-emerald-600 transition-all"
      >
        Start Game
      </button>
    </div>
  );
}
```

**Waiting return (replace the block starting `if (!isMyTurn || ...)`):**

```tsx
if (!isMyTurn || (turnState !== 'NORMAL' && turnState !== 'SEEING_THE_FUTURE')) {
  const waitingFor = players.find((p) => p.id === turnId);
  return (
    <div className="mt-3 flex justify-center text-xs font-semibold uppercase tracking-widest text-purple-800">
      {waitingFor ? `Waiting for Player ${waitingFor.id}…` : ''}
    </div>
  );
}
```

**Main return (replace from `return (` at the end of the function):**

```tsx
return (
  <div className="mt-3 flex flex-col items-center gap-2">
    <div className="flex gap-3 flex-wrap justify-center">
      {/* Draw button */}
      <button
        onClick={() => onAction({ action: 'DRAW_CARD' })}
        className="rounded-full bg-gradient-to-r from-blue-600 to-blue-800 px-6 py-2 font-bold text-white border border-blue-400/40 shadow-lg animate-glow-pulse hover:from-blue-500 hover:to-blue-700 transition-all"
      >
        Draw Card
      </button>

      {/* Play / Combo button */}
      {(singleAction || validCombo) && (
        <button
          onClick={handlePlay}
          disabled={!canConfirm}
          className={`rounded-full px-6 py-2 font-bold text-white border transition-all ${
            canConfirm
              ? 'bg-gradient-to-r from-violet-600 to-purple-800 border-violet-400/40 shadow-lg hover:from-violet-500 hover:to-purple-700'
              : 'bg-gray-800 border-gray-700 opacity-50 cursor-not-allowed'
          }`}
        >
          {singleAction
            ? `Play ${CardDisplayName[hand[selectedIndices[0]]]}`
            : `Play ${selectedIndices.length}-Card Combo`}
        </button>
      )}

      {/* Invalid selection hint */}
      {selectedIndices.length > 0 && !singleAction && !validCombo && (
        <span className="self-center text-xs text-purple-700 font-semibold">
          Select 2, 3, or 5 cat cards for a combo
        </span>
      )}
    </div>

    {/* Target selector */}
    {(needsTarget || comboNeedsTarget) && (
      <TargetSelector
        players={players}
        currentPlayerId={playerId}
        selected={targetedPlayer}
        onSelect={setTargetedPlayer}
      />
    )}

    {/* 3-card combo: card name picker */}
    {comboNeeds3CardPick && targetedPlayer !== null && (
      <div className="mt-1 flex items-center gap-2">
        <span className="text-xs text-purple-600 font-semibold">Request card:</span>
        <select
          value={requestedCard}
          onChange={(e) => setRequestedCard(e.target.value)}
          className="rounded-lg border border-purple-800 bg-purple-950 px-2 py-1 text-sm text-white"
        >
          <option value="">Pick a card…</option>
          {(Object.keys(CardDisplayName) as CardType[]).map((c) => (
            <option key={c} value={c}>
              {CardDisplayName[c]}
            </option>
          ))}
        </select>
      </div>
    )}
  </div>
);
```

- [ ] **Step 2: TypeScript check**

```bash
cd client && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add client/src/pages/game/components/ActionBar.tsx
git commit -m "feat: restyle ActionBar with pill buttons and glow effects"
```

---

## Task 6: LastActionBanner restyle

**Files:**
- Modify: `client/src/pages/game/components/LastActionBanner.tsx`

- [ ] **Step 1: Rewrite the component**

```tsx
// client/src/pages/game/components/LastActionBanner.tsx
interface LastActionBannerProps {
  lastAction?: string;
}

export default function LastActionBanner({ lastAction }: LastActionBannerProps) {
  if (!lastAction) return null;
  return (
    <div className="rounded-full border border-purple-800/60 bg-purple-950/60 px-4 py-1 text-[10px] font-semibold text-purple-200 text-center max-w-[200px] leading-snug">
      {lastAction}
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add client/src/pages/game/components/LastActionBanner.tsx
git commit -m "feat: restyle LastActionBanner with purple pill theme"
```

---

## Task 7: page.tsx — round table layout

**Files:**
- Modify: `client/src/pages/game/page.tsx`

- [ ] **Step 1: Rewrite `page.tsx` — WebSocket/state logic is identical, only JSX changes**

```tsx
// client/src/pages/game/page.tsx
import { useEffect, useRef, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import type { GameState } from '../../models/game-state';
import type { ActionRequest } from '../../models/player-action';
import { ConnectionStatus } from '../../models/connection-status';
import PlayerSeat from './components/PlayerSeat';
import HandCard from './components/HandCard';
import ActionBar from './components/ActionBar';
import LastActionBanner from './components/LastActionBanner';
import FutureViewer from './components/FutureViewer';
import FutureReorder from './components/FutureReorder';
import KittenPlacer from './components/KittenPlacer';
import FavorGiver from './components/FavorGiver';
import DiscardPicker from './components/DiscardPicker';
import GameOverOverlay from './components/GameOverOverlay';
import { isCatCard, isPlayableAlone } from '../../models/game-enums';
import { getSeatPositions } from './table-utils';

export default function GamePage() {
  const navigate = useNavigate();
  const location = useLocation();
  const lobbyName = location.state?.lobbyName || '';

  const ws = useRef<WebSocket | null>(null);
  const [gameState, setGameState] = useState<GameState | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>(ConnectionStatus.Connecting);
  const [selectedIndices, setSelectedIndices] = useState<number[]>([]);

  useEffect(() => {
    if (!lobbyName) { navigate('/'); return; }
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const socket = new WebSocket(
      `${protocol}//${window.location.host}/ws?lobby=${encodeURIComponent(lobbyName)}`
    );
    ws.current = socket;
    socket.onopen = () => setConnectionStatus(ConnectionStatus.Connected);
    socket.onmessage = (event) => {
      try {
        setGameState(JSON.parse(event.data) as GameState);
        setSelectedIndices([]);
      } catch (e) { console.error('Failed to parse game state:', e); }
    };
    socket.onclose = () => setConnectionStatus(ConnectionStatus.Disconnected);
    return () => { socket.close(); ws.current = null; };
  }, [lobbyName, navigate]);

  function sendAction(action: ActionRequest) {
    if (ws.current?.readyState === WebSocket.OPEN) ws.current.send(JSON.stringify(action));
  }

  const handleLeave = () => navigate('/');

  if (!gameState) {
    return (
      <div className="flex h-screen items-center justify-center bg-[#0d0720] text-purple-500 text-xs font-bold tracking-widest uppercase">
        {connectionStatus}
      </div>
    );
  }

  const { playerId, turnId, turnState, hand, players, deckSize, inProgress } = gameState;
  const isMyTurn = playerId === turnId;
  const amFavorTarget = turnState === 'AWAITING_FAVOR' && gameState.targetedPlayer === playerId;
  const handIsPlayable = isMyTurn && (turnState === 'NORMAL' || turnState === 'SEEING_THE_FUTURE') && !amFavorTarget;

  const localPlayerIndex = players.findIndex(p => p.id === playerId);
  const seatPositions = getSeatPositions(players.length, localPlayerIndex < 0 ? 0 : localPlayerIndex);
  // Scale seats down at high player counts so they fit the perimeter
  const seatScale = players.length <= 4 ? 1 : players.length <= 7 ? 0.85 : 0.7;

  const handleCardClick = (index: number) => {
    const card = hand[index];
    const isAlreadySelected = selectedIndices.includes(index);
    if (isAlreadySelected) { setSelectedIndices(selectedIndices.filter(i => i !== index)); return; }
    if (isPlayableAlone(card)) { setSelectedIndices([index]); return; }
    if (isCatCard(card)) {
      const currentlySelected = selectedIndices.map(i => hand[i]);
      if (currentlySelected.some(isPlayableAlone)) setSelectedIndices([index]);
      else setSelectedIndices([...selectedIndices, index]);
    }
  };

  return (
    <div className="min-h-screen bg-[#0d0720] flex flex-col items-center justify-center gap-3 p-3 overflow-hidden">

      {/* ── Round table ── */}
      {/*
        Outer div is a square container. The felt circle is inset by 15% on each
        side, leaving 15% on each edge for seat content to overflow into.
        All player seats are positioned with percentage coords relative to this
        outer container so they land right on the felt's perimeter.
      */}
      <div style={{
        position: 'relative',
        width: 'min(min(90vw, 65vh), 560px)',
        height: 'min(min(90vw, 65vh), 560px)',
      }}>

        {/* Felt circle */}
        <div style={{
          position: 'absolute',
          inset: '15%',
          borderRadius: '50%',
          background: 'radial-gradient(ellipse at 38% 38%, #1a5c2a 40%, #0e3d1a 100%)',
          border: '4px solid #8B6914',
          boxShadow: '0 0 40px rgba(109,40,217,0.25), inset 0 0 30px rgba(0,0,0,0.4)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}>
          {/* Subtle purple glow overlay */}
          <div style={{
            position: 'absolute',
            inset: 0,
            borderRadius: '50%',
            background: 'radial-gradient(ellipse, rgba(76,29,149,0.14) 0%, transparent 70%)',
            pointerEvents: 'none',
          }} />

          {/* Center content */}
          <div className="relative z-10 flex flex-col items-center gap-2 px-3 max-w-full">

            {/* Turn indicator */}
            {inProgress && (
              <div className="flex items-center gap-1.5 rounded-full border border-yellow-700/50 bg-yellow-950/60 px-3 py-1 text-[10px] font-bold uppercase tracking-widest text-yellow-400">
                <span className="w-1.5 h-1.5 rounded-full bg-yellow-400 animate-pulse-dot" />
                {isMyTurn ? 'Your turn' : `Player ${turnId}'s turn`}
              </div>
            )}

            <LastActionBanner lastAction={gameState.lastAction} />

            {/* Deck + discard piles */}
            {inProgress && (
              <div className="flex gap-4 items-end">
                <div className="flex flex-col items-center gap-1">
                  <span className="text-[8px] font-bold uppercase tracking-widest text-purple-800">Deck</span>
                  <button
                    onClick={() => {
                      if (isMyTurn && turnState === 'NORMAL') sendAction({ action: 'DRAW_CARD' });
                    }}
                    style={{
                      position: 'relative',
                      width: 44,
                      height: 62,
                      borderRadius: 6,
                      background: 'radial-gradient(ellipse at 35% 35%, #1e1060 0%, #06020f 100%)',
                      border: '2px solid #7c3aed',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 22,
                      boxShadow: '3px 3px 0 #1e1060, 5px 5px 0 #1e1060, 0 0 16px rgba(109,40,217,0.3)',
                      cursor: isMyTurn && turnState === 'NORMAL' ? 'pointer' : 'default',
                    }}
                  >
                    😼
                    <span style={{
                      position: 'absolute',
                      bottom: 3,
                      right: 3,
                      fontSize: 9,
                      fontWeight: 800,
                      color: '#a78bfa',
                      background: 'rgba(0,0,0,0.6)',
                      padding: '1px 3px',
                      borderRadius: 3,
                    }}>
                      {deckSize}
                    </span>
                  </button>
                </div>
                <div className="flex flex-col items-center gap-1">
                  <span className="text-[8px] font-bold uppercase tracking-widest text-purple-800">Discard</span>
                  <div style={{
                    width: 44,
                    height: 62,
                    borderRadius: 6,
                    background: '#111827',
                    border: '2px solid #374151',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: 22,
                  }}>
                    🃏
                  </div>
                </div>
              </div>
            )}

            {/* Under-attack notice */}
            {inProgress && gameState.underAttack && (
              <div className="rounded-full border border-orange-700/50 bg-orange-950/50 px-3 py-0.5 text-[9px] font-bold text-orange-400">
                ⚡ Draw {gameState.turnsToTake} more
              </div>
            )}

            {/* Error message */}
            {gameState.err && (
              <div className="rounded-full border border-red-800 bg-red-950/70 px-3 py-0.5 text-[9px] font-semibold text-red-300 text-center max-w-[180px]">
                {gameState.err}
              </div>
            )}

            {/* Pre-game lobby info */}
            {!inProgress && (
              <div className="text-center text-[10px] font-semibold text-purple-800 leading-snug">
                <div className="font-bold text-purple-600">{lobbyName}</div>
                <div>{players.length} player{players.length !== 1 ? 's' : ''} connected</div>
              </div>
            )}
          </div>
        </div>

        {/* Player seats — absolutely positioned around the felt perimeter */}
        {players.map((player, idx) => {
          const pos = seatPositions[idx];
          return (
            <PlayerSeat
              key={player.id}
              playerState={player}
              gameState={gameState}
              playerIndex={idx}
              seatScale={seatScale}
              style={{
                position: 'absolute',
                left: `${pos.x}%`,
                top: `${pos.y}%`,
                transform: 'translate(-50%, -50%)',
                zIndex: player.id === playerId ? 10 : 5,
              }}
            />
          );
        })}

        {/* State-specific overlays — centered over the entire table area */}
        {turnState === 'SEEING_THE_FUTURE' && isMyTurn && gameState.future && (
          <div className="absolute inset-0 flex items-center justify-center z-50">
            <FutureViewer cards={gameState.future} />
          </div>
        )}
        {turnState === 'ALTERING_THE_FUTURE' && isMyTurn && gameState.future && (
          <div className="absolute inset-0 flex items-center justify-center z-50">
            <FutureReorder
              cards={gameState.future}
              onConfirm={(newOrder) => sendAction({ action: 'ALTER_FUTURE', alterFutureOrder: newOrder })}
            />
          </div>
        )}
        {turnState === 'AWAITING_KITTEN_PLACEMENT' && isMyTurn && (
          <div className="absolute inset-0 flex items-center justify-center z-50">
            <KittenPlacer
              deckSize={deckSize}
              onPlace={(index) => sendAction({ action: 'PLACE_KITTEN', placeKittenIndex: index })}
            />
          </div>
        )}
        {amFavorTarget && (
          <div className="absolute inset-0 flex items-center justify-center z-50">
            <FavorGiver
              hand={hand}
              requesterPlayerId={turnId}
              onGive={(cardIndex) => sendAction({ action: 'GIVE_FAVOR', useCardIndex: cardIndex })}
            />
          </div>
        )}
        {turnState === 'AWAITING_DISCARD_TAKE' && isMyTurn && gameState.discardOptions && (
          <div className="absolute inset-0 flex items-center justify-center z-50">
            <DiscardPicker
              options={gameState.discardOptions}
              onPick={(card) => sendAction({ action: 'TAKE_FROM_DISCARD', requestedCard: card })}
            />
          </div>
        )}
      </div>

      {/* ── Hand zone (below table) ── */}
      <div className="flex flex-col items-center gap-2 w-full max-w-lg px-2">
        <div className="flex gap-1.5 overflow-x-auto pb-1 justify-center flex-wrap">
          {hand.map((card, i) => (
            <HandCard
              key={i}
              card={card}
              index={i}
              isSelected={selectedIndices.includes(i)}
              isPlayable={handIsPlayable}
              onClick={handleCardClick}
            />
          ))}
          {hand.length === 0 && inProgress && (
            <span className="text-purple-800 text-xs font-semibold self-center">No cards in hand</span>
          )}
        </div>
        <ActionBar
          gameState={gameState}
          selectedIndices={selectedIndices}
          onAction={(action) => { sendAction(action); setSelectedIndices([]); }}
        />
      </div>

      {/* Game over overlay */}
      {turnState === 'GAME_OVER' && (
        <GameOverOverlay players={players} onLeave={handleLeave} />
      )}

      {/* Connection status badge */}
      {connectionStatus !== ConnectionStatus.Connected && (
        <div className="fixed top-2 right-2 rounded-full border border-purple-800 bg-purple-950/90 px-3 py-1 text-xs text-purple-300 font-semibold">
          {connectionStatus}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: TypeScript check — fix any errors**

```bash
cd client && npx tsc --noEmit
```

Expected: no errors. If there are errors, fix them before continuing.

- [ ] **Step 3: Run all tests**

```bash
cd client && npm test
```

Expected: `5 tests passed`.

- [ ] **Step 4: Start both servers and verify visually**

```bash
# Terminal 1
cd game && go run .

# Terminal 2
cd client && npm run dev
```

Open `http://localhost:5173` and join the `test` lobby. Check:
- [ ] Green felt circle renders, gold border visible
- [ ] Player seat appears at bottom (6 o'clock) with 😎 avatar
- [ ] Opponents arc around the top half with galaxy card fans
- [ ] Active player has gold glow ring + ⚡ in name
- [ ] Dead players are greyscale and dimmed
- [ ] Card count badge appears top-right of opponent avatar
- [ ] Deck shows 😼 back + card count badge
- [ ] Draw Card button has blue glow pulse
- [ ] Hand cards show emoji + gradient backgrounds
- [ ] Selecting a card lifts it with white border + purple glow
- [ ] Resize to narrow width (< 480px): table scales down, card names hide on mobile

- [ ] **Step 5: Commit**

```bash
git add client/src/pages/game/page.tsx
git commit -m "feat: round table layout — arc seats, galaxy deck, purple theme"
```

---

## Self-Review

**Spec coverage check:**

| Spec requirement | Task |
|---|---|
| Arc/poker → round table layout | Task 7 (page.tsx) |
| Players at clock positions | Task 2 (table-utils) + Task 7 |
| You always at 6 o'clock | Task 2 |
| Galaxy card backs for opponents | Task 3 (PlayerSeat > CardFan) |
| Emoji avatars, distinct per player | Task 3 (PLAYER_EMOJIS array) |
| Gold ring on active player | Task 3 |
| Online dot | Task 3 |
| Card count badge | Task 3 |
| Dead player greyscale | Task 3 |
| Supports 2–10 players | Task 2 (math) + Task 3 (seatScale) |
| seatScale 1.0/0.85/0.7 breakpoints | Task 7 (computed in page.tsx) |
| Card emojis all 16 types inc. CAT5 | Task 4 |
| Draw button glow pulse | Task 1 (CSS) + Task 5 |
| Turn indicator pill + pulse dot | Task 7 |
| LastActionBanner purple | Task 6 |
| ActionBar pill buttons | Task 5 |
| Mobile: table scales to viewport | Task 7 (`min(min(90vw, 65vh), 560px)`) |
| Mobile: card names hidden | Task 4 (`hidden sm:block`) |
| State overlays unchanged in logic | Task 7 (overlays kept, repositioned) |

**No placeholders found.**

**Type consistency check:** `SeatPosition` defined in Task 2, consumed in Task 7. `PlayerSeatProps.seatScale` defined in Task 3, passed in Task 7. `CARD_BG`/`CARD_EMOJI`/`SHORT_NAME` keys match all 16 `CardType` values including `CAT5`. ✓
