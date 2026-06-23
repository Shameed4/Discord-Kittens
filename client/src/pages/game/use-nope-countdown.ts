import { useEffect, useState } from 'react';

// prevent nope clicks after this time since latency will probably prevent it anyways
export const NOPE_LATENCY_BUFFER_MS = 300;
// delay before allowing user to click button to prevent accidental nope as soon as nope switches to yup
export const NOPE_FLIP_DEBOUNCE_MS = 500;
// shave some time off actual countdown to give appearance of leeway
const NOPE_DISPLAY_SKEW_MS = 250;

export interface NopeCountdown {
  remaining: number; // raw ms left until the deadline (0 once expired)
  fraction: number; // remaining / window length, in [0, 1] — for progress bars
}

// Ticks down to the given deadline (server unix-ms) while it is active. Pass
// undefined to disable. Each fresh deadline restarts the countdown, anchoring
// the "full window" to the first observed remaining so a bar can show a fraction.
export function useNopeCountdown(deadline: number | undefined): NopeCountdown {
  const [countdown, setCountdown] = useState<NopeCountdown>({
    remaining: 0,
    fraction: 0,
  });

  useEffect(() => {
    if (!deadline) return;
    const total = Math.max(1, deadline - Date.now());
    const tick = () => {
      const ms = Math.max(0, deadline - Date.now());
      setCountdown({ remaining: ms, fraction: Math.min(1, ms / total) });
    };
    tick();
    const id = setInterval(tick, 100);
    return () => clearInterval(id);
  }, [deadline]);

  return countdown;
}

// Formats the remaining time for display, applying the conservative skew.
export function formatNopeSeconds(remaining: number): string {
  return (Math.max(0, remaining - NOPE_DISPLAY_SKEW_MS) / 1000).toFixed(1);
}
