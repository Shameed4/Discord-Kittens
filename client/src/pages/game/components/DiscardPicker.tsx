import { CardDisplayName, type CardType } from '../../../models/game-enums';

interface DiscardPickerProps {
  options: CardType[];
  onPick: (card: CardType) => void;
}

export default function DiscardPicker({ options, onPick }: DiscardPickerProps) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-xl border border-yellow-700 bg-gray-800 p-5">
      <span className="text-sm font-bold tracking-widest text-yellow-300 uppercase">
        5-Card Combo
      </span>
      <span className="text-xs text-gray-400">
        Take any card from the discard pile
      </span>
      <div className="flex flex-wrap justify-center gap-2">
        {options.map((card) => (
          <button
            key={card}
            onClick={() => onPick(card)}
            className="rounded-lg border border-yellow-600 bg-yellow-900 px-3 py-2 text-sm font-semibold text-white transition-colors hover:bg-yellow-800"
          >
            {CardDisplayName[card]}
          </button>
        ))}
      </div>
    </div>
  );
}
