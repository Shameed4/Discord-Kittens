import { useState } from 'react';
import { CardDisplayName, type CardType } from '../../../models/game-enums';

interface FutureReorderProps {
  cards: CardType[];
  onConfirm: (newOrder: number[]) => void;
}

export default function FutureReorder({ cards, onConfirm }: FutureReorderProps) {
  const [order, setOrder] = useState<number[]>(cards.map((_, i) => i));

  const swap = (i: number, j: number) => {
    const next = [...order];
    [next[i], next[j]] = [next[j], next[i]];
    setOrder(next);
  };

  return (
    <div className="flex flex-col items-center gap-3">
      <span className="text-gray-300 text-sm font-semibold uppercase tracking-widest">Reorder the top cards</span>
      <div className="flex gap-3">
        {order.map((originalIdx, slot) => (
          <div key={slot} className="flex flex-col items-center gap-1">
            <span className="text-[10px] text-gray-400">#{slot + 1}</span>
            <div className="w-20 h-28 rounded-lg border-2 border-cyan-500 bg-cyan-900 flex items-center justify-center text-white text-xs font-semibold text-center px-1">
              {CardDisplayName[cards[originalIdx]]}
            </div>
            <div className="flex gap-1">
              <button
                onClick={() => slot > 0 && swap(slot - 1, slot)}
                disabled={slot === 0}
                className="px-2 py-0.5 text-xs bg-gray-700 hover:bg-gray-600 disabled:opacity-30 rounded"
              >
                ←
              </button>
              <button
                onClick={() => slot < order.length - 1 && swap(slot, slot + 1)}
                disabled={slot === order.length - 1}
                className="px-2 py-0.5 text-xs bg-gray-700 hover:bg-gray-600 disabled:opacity-30 rounded"
              >
                →
              </button>
            </div>
          </div>
        ))}
      </div>
      <button
        onClick={() => onConfirm(order)}
        className="px-5 py-2 bg-cyan-600 hover:bg-cyan-500 text-white font-bold rounded-lg transition-colors"
      >
        Confirm Order
      </button>
    </div>
  );
}
