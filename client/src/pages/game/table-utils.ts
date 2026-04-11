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
