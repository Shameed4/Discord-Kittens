interface RestartConfirmProps {
  onConfirm: () => void;
  onCancel: () => void;
}

export default function RestartConfirm({
  onConfirm,
  onCancel,
}: RestartConfirmProps) {
  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/70">
      <div className="flex max-w-xs flex-col items-center gap-4 rounded-2xl border border-purple-700 bg-gray-900 p-6 text-center shadow-2xl">
        <span className="text-lg font-black text-white">Restart Lobby?</span>
        <p className="text-xs leading-snug font-semibold text-purple-300">
          This ends the current game and returns everyone to the lobby.
        </p>
        <div className="flex gap-2">
          <button
            onClick={onCancel}
            className="rounded-lg border border-gray-600 bg-gray-800 px-4 py-1.5 text-sm font-bold text-gray-200 transition-colors hover:bg-gray-700"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className="rounded-lg border border-red-400/40 bg-gradient-to-r from-red-600 to-rose-700 px-4 py-1.5 text-sm font-bold text-white transition-all hover:from-red-500 hover:to-rose-600"
          >
            Restart
          </button>
        </div>
      </div>
    </div>
  );
}
