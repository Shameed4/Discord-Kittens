import { useEffect, useState } from 'react';
import type { GameState } from '../../../models/game-state';
import type { ActionRequest } from '../../../models/player-action';
import {
  NOPE_FLIP_DEBOUNCE_MS,
  NOPE_LATENCY_BUFFER_MS,
  formatNopeSeconds,
} from '../use-nope-countdown';

interface NopeButtonProps {
  gameState: GameState;
  nopeRemaining: number;
  onAction: (action: ActionRequest) => void;
}

// Shown during the ACCEPTING_NOPES window. Any alive player holding a Nope card
// can flip the verdict: "Nope!" when the action currently stands, "Yup!" once it
// is noped. The acting player can also Yup to defend their own move.
export default function NopeButton({
  gameState,
  nopeRemaining,
  onAction,
}: NopeButtonProps) {
  const { hand, players, playerId } = gameState;
  const noped = gameState.isNoped ?? false;
  const me = players.find((p) => p.id === playerId);
  const canParticipate = (me?.isAlive ?? false) && hand.includes('NOPE');

  // Disabled on mount, then enabled after the debounce. The parent remounts this
  // component (via a key keyed on the verdict) on each flip, so every nope/yup
  // re-runs this from the disabled state — preventing a mid-click from triggering
  // the freshly relabeled opposite action. Window open is just the first mount.
  const [debouncing, setDebouncing] = useState(true);
  useEffect(() => {
    const id = setTimeout(() => setDebouncing(false), NOPE_FLIP_DEBOUNCE_MS);
    return () => clearTimeout(id);
  }, []);

  const seconds = formatNopeSeconds(nopeRemaining);

  // Players without a Nope (or eliminated) just watch the clock.
  if (!canParticipate) {
    return (
      <div className="flex justify-center text-center text-[10px] font-semibold tracking-wide text-purple-700 uppercase">
        Nope window · {seconds}s
      </div>
    );
  }

  const tooLate = nopeRemaining <= NOPE_LATENCY_BUFFER_MS;
  const disabled = debouncing || tooLate;
  const colors = noped
    ? 'border-green-400/40 from-green-600 to-emerald-700 hover:from-green-500 hover:to-emerald-600'
    : 'border-red-400/40 from-red-600 to-rose-800 hover:from-red-500 hover:to-rose-700';

  return (
    <div className="flex flex-col items-center gap-1">
      <button
        onClick={() => onAction({ action: 'PLAY_NOPE', wantNoped: !noped })}
        disabled={disabled}
        className={[
          'rounded-full border bg-linear-to-r px-4 py-1.5 text-sm font-black text-white shadow-lg transition-all',
          colors,
          disabled ? 'cursor-not-allowed opacity-40' : 'cursor-pointer',
        ].join(' ')}
      >
        {noped ? 'Yup! ✅' : 'Nope! 🚫'}
      </button>
      <span className="text-[10px] font-bold tracking-widest text-purple-700 tabular-nums">
        {seconds}s
      </span>
    </div>
  );
}
