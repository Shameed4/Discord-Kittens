import type { PlayerState } from '../../../models/game-state';

interface TargetSelectorProps {
  players: PlayerState[];
  currentPlayerId: number;
  selected: number | null;
  onSelect: (playerId: number) => void;
}

export default function TargetSelector({ players, currentPlayerId, selected, onSelect }: TargetSelectorProps) {
  const targets = players.filter(p => p.id !== currentPlayerId && p.isAlive);

  if (targets.length === 0) return null;

  return (
    <div className="flex items-center gap-2 flex-wrap justify-center mt-2">
      <span className="text-gray-400 text-xs">Target:</span>
      {targets.map(p => (
        <button
          key={p.id}
          onClick={() => onSelect(p.id)}
          className={`
            px-3 py-1 rounded-full text-sm font-semibold border transition-colors
            ${selected === p.id
              ? 'bg-yellow-400 border-yellow-300 text-gray-900'
              : 'bg-gray-700 border-gray-600 text-white hover:bg-gray-600'}
          `}
        >
          Player {p.id}
        </button>
      ))}
    </div>
  );
}
