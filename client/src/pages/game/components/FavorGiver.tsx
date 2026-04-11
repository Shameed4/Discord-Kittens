import { CardDisplayName, type CardType } from '../../../models/game-enums';

interface FavorGiverProps {
  hand: CardType[];
  requesterPlayerId: number;
  onGive: (cardIndex: number) => void;
}

export default function FavorGiver({ hand, requesterPlayerId, onGive }: FavorGiverProps) {
  return (
    <div className="flex flex-col items-center gap-3 bg-gray-800 border border-pink-700 rounded-xl p-5">
      <span className="text-pink-300 font-bold text-sm uppercase tracking-widest">
        Player {requesterPlayerId} wants a favor
      </span>
      <span className="text-gray-400 text-xs">Choose a card to give</span>
      <div className="flex gap-2 flex-wrap justify-center">
        {hand.map((card, i) => (
          <button
            key={i}
            onClick={() => onGive(i)}
            className="w-20 h-28 rounded-lg border-2 border-pink-500 bg-pink-900 hover:bg-pink-800 flex items-center justify-center text-white text-xs font-semibold text-center px-1 transition-colors"
          >
            {CardDisplayName[card]}
          </button>
        ))}
      </div>
    </div>
  );
}
