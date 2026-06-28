import type { PlayerState } from '../../../models/game-state';

interface GameOverOverlayProps {
  players: PlayerState[];
  isSpectator: boolean;
  // Hidden inside Discord: the activity is bound to the instance and there's no
  // home screen to return to, so "Leave Lobby" would be a dead end.
  hideLeave: boolean;
  onLeave: () => void;
  onRestart: () => void;
}

export default function GameOverOverlay({
  players,
  isSpectator,
  hideLeave,
  onLeave,
  onRestart,
}: GameOverOverlayProps) {
  const winner = players.find((p) => p.isAlive);

  return (
    <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
      <div className="bg-gray-900 border border-gray-700 rounded-2xl p-10 flex flex-col items-center gap-5 shadow-2xl">
        <span className="text-4xl font-black text-white">
          {winner ? `${winner.name} wins!` : 'Draw!'}
        </span>
        <div className="flex gap-3">
          {!isSpectator && (
            <button
              onClick={onRestart}
              className="px-6 py-2 bg-emerald-600 hover:bg-emerald-500 text-white font-bold rounded-lg transition-colors"
            >
              Restart Lobby
            </button>
          )}
          {!hideLeave && (
            <button
              onClick={onLeave}
              className="px-6 py-2 bg-indigo-600 hover:bg-indigo-500 text-white font-bold rounded-lg transition-colors"
            >
              Leave Lobby
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
