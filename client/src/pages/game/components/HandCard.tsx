import { CardDisplayName, isCatCard, type CardType } from '../../../models/game-enums';

interface HandCardProps {
  card: CardType;
  index: number;
  isSelected: boolean;
  isPlayable: boolean;
  onClick: (index: number) => void;
}

const cardColors: Record<string, string> = {
  DEFUSE:           'bg-green-700 border-green-500',
  EXPLODING_KITTEN: 'bg-red-800 border-red-500',
  SKIP:             'bg-blue-700 border-blue-500',
  ATTACK:           'bg-orange-700 border-orange-500',
  TARGETED_ATTACK:  'bg-orange-800 border-orange-600',
  CAT1:             'bg-purple-700 border-purple-500',
  CAT2:             'bg-purple-700 border-purple-500',
  CAT3:             'bg-purple-700 border-purple-500',
  CAT4:             'bg-purple-700 border-purple-500',
  FERAL_CAT:        'bg-purple-800 border-purple-600',
  SEE_THE_FUTURE:   'bg-cyan-700 border-cyan-500',
  ALTER_THE_FUTURE: 'bg-cyan-800 border-cyan-600',
  SHUFFLE:          'bg-yellow-700 border-yellow-500',
  DRAW_FROM_BOTTOM: 'bg-teal-700 border-teal-500',
  FAVOR:            'bg-pink-700 border-pink-500',
};

export default function HandCard({ card, index, isSelected, isPlayable, onClick }: HandCardProps) {
  const color = cardColors[card] ?? 'bg-gray-700 border-gray-500';
  const isCat = isCatCard(card);

  return (
    <button
      onClick={() => isPlayable && onClick(index)}
      disabled={!isPlayable}
      className={`
        relative flex flex-col items-center justify-center
        w-20 h-28 rounded-lg border-2 text-white text-center text-xs font-semibold
        transition-all duration-150 shrink-0 select-none
        ${color}
        ${isSelected ? 'ring-2 ring-white ring-offset-2 ring-offset-gray-900 -translate-y-3' : ''}
        ${isPlayable ? 'cursor-pointer hover:-translate-y-1' : 'opacity-40 cursor-not-allowed'}
      `}
    >
      <span className="px-1 leading-tight">{CardDisplayName[card]}</span>
      {isCat && (
        <span className="absolute bottom-1 text-[9px] font-black uppercase tracking-widest opacity-70">
          cat
        </span>
      )}
    </button>
  );
}
