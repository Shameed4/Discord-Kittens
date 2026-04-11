import { CardDisplayName, type CardType } from '../../../models/game-enums';

interface FutureViewerProps {
  cards: CardType[];
}

export default function FutureViewer({ cards }: FutureViewerProps) {
  return (
    <div className="flex flex-col items-center gap-2">
      <span className="text-gray-300 text-sm font-semibold uppercase tracking-widest">Top of Deck</span>
      <div className="flex gap-3">
        {cards.map((card, i) => (
          <div
            key={i}
            className="w-20 h-28 rounded-lg border-2 border-cyan-500 bg-cyan-900 flex flex-col items-center justify-center text-white text-xs font-semibold text-center px-1"
          >
            <span className="text-[10px] text-cyan-300 mb-1">#{i + 1}</span>
            {CardDisplayName[card]}
          </div>
        ))}
      </div>
    </div>
  );
}
