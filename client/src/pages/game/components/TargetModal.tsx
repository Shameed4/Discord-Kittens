import { useState } from 'react';
import { CardDisplayName, type CardType } from '../../../models/game-enums';
import type { PlayerState } from '../../../models/game-state';

interface TargetModalProps {
  title: string;
  prompt: string;
  players: PlayerState[];
  currentPlayerId: number;
  /** When true, a card must also be chosen before the move commits (3-card combo). */
  needsCard: boolean;
  onCommit: (targetedPlayer: number, requestedCard?: string) => void;
  onCancel: () => void;
}

export default function TargetModal({
  title,
  prompt,
  players,
  currentPlayerId,
  needsCard,
  onCommit,
  onCancel,
}: TargetModalProps) {
  const [selectedTarget, setSelectedTarget] = useState<number | null>(null);
  const [requestedCard, setRequestedCard] = useState<string>('');

  const targets = players.filter((p) => p.id !== currentPlayerId && p.isAlive);

  const handleTargetClick = (id: number) => {
    // Target-only moves commit immediately on selection; combos that also need a
    // named card wait for the Confirm button.
    if (!needsCard) {
      onCommit(id);
      return;
    }
    setSelectedTarget(id);
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
      onClick={onCancel}
    >
      <div
        className="flex max-w-sm flex-col items-center gap-3 rounded-xl border border-yellow-700 bg-gray-800 p-5"
        onClick={(e) => e.stopPropagation()}
      >
        <span className="text-sm font-bold tracking-widest text-yellow-300 uppercase">
          {title}
        </span>
        <span className="text-center text-xs text-gray-400">{prompt}</span>

        <div className="flex flex-wrap justify-center gap-2">
          {targets.map((p) => (
            <button
              key={p.id}
              onClick={() => handleTargetClick(p.id)}
              className={`rounded-full border px-3 py-1.5 text-sm font-semibold transition-colors ${
                selectedTarget === p.id
                  ? 'border-yellow-300 bg-yellow-400 text-gray-900'
                  : 'border-yellow-600 bg-yellow-900 text-white hover:bg-yellow-800'
              }`}
            >
              {p.name}
            </button>
          ))}
        </div>

        {needsCard && selectedTarget !== null && (
          <div className="flex flex-col items-center gap-2">
            <span className="text-xs font-semibold text-purple-300">
              Card to request
            </span>
            <select
              value={requestedCard}
              onChange={(e) => setRequestedCard(e.target.value)}
              className="rounded-lg border border-purple-700 bg-purple-950 px-2 py-1 text-sm text-white"
            >
              <option value="">Pick a card…</option>
              {(Object.keys(CardDisplayName) as CardType[]).map((c) => (
                <option key={c} value={c}>
                  {CardDisplayName[c]}
                </option>
              ))}
            </select>
            <button
              onClick={() => onCommit(selectedTarget, requestedCard)}
              disabled={requestedCard === ''}
              className={`rounded-full border px-5 py-1.5 text-sm font-bold text-white transition-all ${
                requestedCard !== ''
                  ? 'border-violet-400/40 bg-gradient-to-r from-violet-600 to-purple-800 hover:from-violet-500 hover:to-purple-700'
                  : 'cursor-not-allowed border-gray-700 bg-gray-800 opacity-50'
              }`}
            >
              Steal
            </button>
          </div>
        )}

        <button
          onClick={onCancel}
          className="text-[10px] font-semibold tracking-widest text-gray-500 uppercase hover:text-gray-300"
        >
          Cancel
        </button>
      </div>
    </div>
  );
}
