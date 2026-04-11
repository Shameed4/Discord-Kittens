// client/src/pages/game/components/HandCard.tsx
import { isCatCard, type CardType } from '../../../models/game-enums';

interface HandCardProps {
  card: CardType;
  index: number;
  isSelected: boolean;
  isPlayable: boolean;
  onClick: (index: number) => void;
}

const CARD_BG: Record<CardType, string> = {
  DEFUSE:           'linear-gradient(135deg, #166534, #14532d)',
  EXPLODING_KITTEN: 'linear-gradient(135deg, #991b1b, #7f1d1d)',
  SKIP:             'linear-gradient(135deg, #1d4ed8, #1e3a8a)',
  ATTACK:           'linear-gradient(135deg, #c2410c, #9a3412)',
  TARGETED_ATTACK:  'linear-gradient(135deg, #b45309, #92400e)',
  CAT1:             'linear-gradient(135deg, #7e22ce, #4c1d95)',
  CAT2:             'linear-gradient(135deg, #7e22ce, #4c1d95)',
  CAT3:             'linear-gradient(135deg, #7e22ce, #4c1d95)',
  CAT4:             'linear-gradient(135deg, #7e22ce, #4c1d95)',
  CAT5:             'linear-gradient(135deg, #7e22ce, #4c1d95)',
  FERAL_CAT:        'linear-gradient(135deg, #6b21a8, #3b0764)',
  SEE_THE_FUTURE:   'linear-gradient(135deg, #0e7490, #164e63)',
  ALTER_THE_FUTURE: 'linear-gradient(135deg, #0369a1, #1e3a8a)',
  SHUFFLE:          'linear-gradient(135deg, #a16207, #713f12)',
  DRAW_FROM_BOTTOM: 'linear-gradient(135deg, #0f766e, #134e4a)',
  FAVOR:            'linear-gradient(135deg, #be185d, #9d174d)',
};

const CARD_EMOJI: Record<CardType, string> = {
  DEFUSE:           '🔧',
  EXPLODING_KITTEN: '💥',
  SKIP:             '⏭️',
  ATTACK:           '⚡',
  TARGETED_ATTACK:  '🎯',
  CAT1:             '🐾',
  CAT2:             '🌮',
  CAT3:             '🥔',
  CAT4:             '🐟',
  CAT5:             '🌈',
  FERAL_CAT:        '🦄',
  SEE_THE_FUTURE:   '🔮',
  ALTER_THE_FUTURE: '✨',
  SHUFFLE:          '🔀',
  DRAW_FROM_BOTTOM: '⬇️',
  FAVOR:            '🎁',
};

const SHORT_NAME: Record<CardType, string> = {
  DEFUSE:           'Defuse',
  EXPLODING_KITTEN: 'Bomb!',
  SKIP:             'Skip',
  ATTACK:           'Attack',
  TARGETED_ATTACK:  'Target',
  CAT1:             'Taco',
  CAT2:             'Beard',
  CAT3:             'Potato',
  CAT4:             'Melon',
  CAT5:             'Rainbow',
  FERAL_CAT:        'Feral',
  SEE_THE_FUTURE:   'Future',
  ALTER_THE_FUTURE: 'Alter',
  SHUFFLE:          'Shuffle',
  DRAW_FROM_BOTTOM: 'Bottom',
  FAVOR:            'Favor',
};

export default function HandCard({ card, index, isSelected, isPlayable, onClick }: HandCardProps) {
  const isCat = isCatCard(card);

  return (
    <button
      onClick={() => isPlayable && onClick(index)}
      disabled={!isPlayable}
      style={{
        background: CARD_BG[card] ?? 'linear-gradient(135deg, #374151, #1f2937)',
        transform: isSelected ? 'translateY(-12px)' : undefined,
        boxShadow: isSelected
          ? '0 8px 20px rgba(167,139,250,0.5), 0 0 0 2px white'
          : '0 4px 12px rgba(0,0,0,0.5)',
      }}
      className={[
        'relative flex flex-col items-center justify-center gap-0.5',
        'w-[46px] h-[66px] rounded-lg border-2 text-white shrink-0 select-none',
        'transition-all duration-150',
        isSelected ? 'border-white' : 'border-white/20',
        isPlayable
          ? 'cursor-pointer hover:-translate-y-2'
          : 'opacity-40 cursor-not-allowed',
      ].join(' ')}
    >
      <span className="text-lg leading-none">{CARD_EMOJI[card]}</span>
      <span className="hidden sm:block text-[7px] font-black uppercase tracking-tight px-0.5 leading-tight text-center text-white/90">
        {SHORT_NAME[card]}
      </span>
      {isCat && (
        <span className="absolute bottom-0.5 text-[6px] font-black uppercase tracking-widest opacity-60">
          cat
        </span>
      )}
    </button>
  );
}
