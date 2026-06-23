// client/src/pages/game/components/HandCard.tsx
import { isCatCard, type CardType } from '../../../models/game-enums';

interface HandCardProps {
  card: CardType;
  index: number;
  isSelected: boolean;
  isPlayable: boolean;
  onClick: (index: number) => void;
}

interface CardInfo {
  bg: string;
  emoji: string;
  name: string;
}

const CARD_INFO: Record<CardType, CardInfo> = {
  DEFUSE: {
    bg: 'linear-gradient(135deg, #166534, #14532d)',
    emoji: '🔧',
    name: 'Defuse',
  },
  EXPLODING_KITTEN: {
    bg: 'linear-gradient(135deg, #991b1b, #7f1d1d)',
    emoji: '💥',
    name: 'Bomb!',
  },
  SKIP: {
    bg: 'linear-gradient(135deg, #1d4ed8, #1e3a8a)',
    emoji: '⏭️',
    name: 'Skip',
  },
  ATTACK: {
    bg: 'linear-gradient(135deg, #c2410c, #9a3412)',
    emoji: '⚡',
    name: 'Attack',
  },
  TARGETED_ATTACK: {
    bg: 'linear-gradient(135deg, #b45309, #92400e)',
    emoji: '🎯',
    name: 'Target',
  },
  CAT1: {
    bg: 'linear-gradient(135deg, #7e22ce, #4c1d95)',
    emoji: '🌮',
    name: 'Taco',
  },
  CAT2: {
    bg: 'linear-gradient(135deg, #7e22ce, #4c1d95)',
    emoji: '😡',
    name: 'Rage',
  },
  CAT3: {
    bg: 'linear-gradient(135deg, #7e22ce, #4c1d95)',
    emoji: '🥔',
    name: 'Potato',
  },
  CAT4: {
    bg: 'linear-gradient(135deg, #7e22ce, #4c1d95)',
    emoji: '🍉',
    name: 'Melon',
  },
  CAT5: {
    bg: 'linear-gradient(135deg, #7e22ce, #4c1d95)',
    emoji: '🌈',
    name: 'Rainbow',
  },
  FERAL_CAT: {
    bg: 'linear-gradient(135deg, #6b21a8, #3b0764)',
    emoji: '🦄',
    name: 'Feral',
  },
  SEE_THE_FUTURE: {
    bg: 'linear-gradient(135deg, #0e7490, #164e63)',
    emoji: '🔮',
    name: 'Future',
  },
  ALTER_THE_FUTURE: {
    bg: 'linear-gradient(135deg, #0369a1, #1e3a8a)',
    emoji: '✨',
    name: 'Alter',
  },
  SHUFFLE: {
    bg: 'linear-gradient(135deg, #a16207, #713f12)',
    emoji: '🔀',
    name: 'Shuffle',
  },
  DRAW_FROM_BOTTOM: {
    bg: 'linear-gradient(135deg, #0f766e, #134e4a)',
    emoji: '⬇️',
    name: 'Bottom',
  },
  FAVOR: {
    bg: 'linear-gradient(135deg, #be185d, #9d174d)',
    emoji: '🎁',
    name: 'Favor',
  },
  NOPE: {
    bg: 'linear-gradient(135deg, #b91c1c, #450a0a)',
    emoji: '🚫',
    name: 'Nope',
  },
};

const DEFAULT_BG = 'linear-gradient(135deg, #374151, #1f2937)';

export default function HandCard({
  card,
  index,
  isSelected,
  isPlayable,
  onClick,
}: HandCardProps) {
  const isCat = isCatCard(card);
  const info = CARD_INFO[card];

  return (
    <button
      onClick={() => isPlayable && onClick(index)}
      disabled={!isPlayable}
      style={{
        background: info?.bg ?? DEFAULT_BG,
        transform: isSelected ? 'translateY(-12px)' : undefined,
        boxShadow: isSelected
          ? '0 8px 20px rgba(167,139,250,0.5), 0 0 0 2px white'
          : '0 4px 12px rgba(0,0,0,0.5)',
      }}
      className={[
        'relative flex flex-col items-center justify-center gap-0.5',
        'h-[66px] w-[46px] shrink-0 rounded-lg border-2 text-white select-none',
        'transition-all duration-150',
        isSelected ? 'border-white' : 'border-white/20',
        isPlayable
          ? 'cursor-pointer hover:-translate-y-2'
          : 'cursor-not-allowed opacity-40',
      ].join(' ')}
    >
      <span className="text-lg leading-none">{info?.emoji}</span>
      <span className="block px-0.5 text-center text-[7px] leading-tight font-black tracking-tight text-white/90 uppercase">
        {info?.name}
      </span>
      {isCat && (
        <span className="absolute bottom-0.5 text-[6px] font-black tracking-widest uppercase opacity-60">
          cat
        </span>
      )}
    </button>
  );
}
