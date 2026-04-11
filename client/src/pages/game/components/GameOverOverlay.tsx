import type { PlayerState } from '../../../models/game-state';

interface GameOverOverlayProps {
  players: PlayerState[];
  onLeave: () => void;
}

export default function GameOverOverlay({ players, onLeave }: GameOverOverlayProps) {
  const winner = players.find(p => p.isAlive);

  return (
    <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
      <div className="bg-gray-900 border border-gray-700 rounded-2xl p-10 flex flex-col items-center gap-5 shadow-2xl">
        <span className="text-4xl font-black text-white">
          {winner ? `Player ${winner.id} wins!` : 'Draw!'}
        </span>
        <button
          onClick={onLeave}
          className="px-6 py-2 bg-indigo-600 hover:bg-indigo-500 text-white font-bold rounded-lg transition-colors"
        >
          Leave Lobby
        </button>
      </div>
    </div>
  );
}
