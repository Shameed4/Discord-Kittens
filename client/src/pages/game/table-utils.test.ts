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
      expect(p.x).toBeGreaterThanOrEqual(0);
      expect(p.x).toBeLessThanOrEqual(100);
      expect(p.y).toBeGreaterThanOrEqual(0);
      expect(p.y).toBeLessThanOrEqual(100);
    });
  });
});
