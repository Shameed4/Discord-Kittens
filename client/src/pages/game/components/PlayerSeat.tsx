// client/src/pages/game/components/PlayerSeat.tsx
import { useState, type CSSProperties } from 'react';
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
  /** True for seats on the right half — flips layout so content grows toward center */
  mirrored: boolean;
  /** Absolute positioning injected by the parent (left/top/transform) */
  style?: CSSProperties;
}

export default function PlayerSeat({
  playerState,
  gameState,
  playerIndex,
  seatScale,
  mirrored,
  style,
}: PlayerSeatProps) {
  const { id, name, avatar, cardCount, isAlive, isOnline } = playerState;
  const isLocal = id === gameState.playerId;
  // Fall back to the emoji if the avatar URL is missing or fails to load.
  const [avatarFailed, setAvatarFailed] = useState(false);
  const showAvatar = Boolean(avatar) && !avatarFailed;
  const isTurn = gameState.inProgress
    ? id === gameState.turnId && isAlive
    : gameState.turnState === 'NOT_STARTED' && playerIndex === 0;

  const avatarPx = Math.round(44 * seatScale);
  const emojiFpx = Math.round(22 * seatScale);
  const nameFpx = Math.round(10 * seatScale);
  const dotPx = Math.round(10 * seatScale);
  const badgePx = Math.round(17 * seatScale);
  const gapPx = Math.round(4 * seatScale);

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

  const avatarBlock = (
    <div style={{ position: 'relative' }}>
      <div className={isTurn ? 'animate-turn-flash' : undefined} style={{
        width: avatarPx,
        height: avatarPx,
        borderRadius: '50%',
        background: 'radial-gradient(circle at 35% 35%, #2e1065, #0d0520)',
        border: `${Math.round(2 * seatScale)}px solid ${borderColor}`,
        boxShadow,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: emojiFpx,
        filter: !isAlive ? 'grayscale(1) brightness(0.5)' : 'none',
        overflow: 'hidden',
      }}>
        {showAvatar ? (
          <img
            src={avatar}
            alt={name}
            onError={() => setAvatarFailed(true)}
            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
          />
        ) : (
          emoji
        )}
      </div>

      {/* Online indicator */}
      <div
        title={isOnline ? 'Online' : 'Offline'}
        style={{
          position: 'absolute',
          bottom: 0,
          right: 0,
          width: dotPx,
          height: dotPx,
          borderRadius: '50%',
          background: isOnline ? '#22c55e' : '#6b7280',
          border: `${Math.round(2 * seatScale)}px solid #0d0720`,
        }}
      />

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
  );

  const nameBlock = (
    <div style={{
      fontSize: nameFpx,
      fontWeight: 700,
      color: nameColor,
      textAlign: mirrored ? 'right' : 'left',
      whiteSpace: 'nowrap',
      letterSpacing: '0.3px',
    }}>
      {isLocal ? `${name} (You)` : name}
      {isTurn ? ' ⚡' : ''}
    </div>
  );

  // Local player: simple vertical avatar + name (hand is shown below the table).
  if (isLocal) {
    return (
      <div style={{
        ...style,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: gapPx,
        opacity: isAlive ? 1 : 0.5,
      }}>
        {avatarBlock}
        {nameBlock}
      </div>
    );
  }

  // Opponent: horizontal — avatar beside a stacked name + card fan.
  return (
    <div style={{
      ...style,
      display: 'flex',
      flexDirection: mirrored ? 'row-reverse' : 'row',
      alignItems: 'center',
      gap: gapPx * 2,
      opacity: isAlive ? 1 : 0.5,
    }}>
      {avatarBlock}
      <div style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: mirrored ? 'flex-end' : 'flex-start',
        gap: gapPx,
      }}>
        {nameBlock}
        {gameState.inProgress && (
          <CardFan cardCount={cardCount} seatScale={seatScale} />
        )}
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
            border: `${Math.max(1, Math.round(1.5 * seatScale))}px solid #6d28d9`,
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
