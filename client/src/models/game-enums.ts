export const CardType = {
  Defuse: 'DEFUSE',
  ExplodingKitten: 'EXPLODING_KITTEN',
  Skip: 'SKIP',
  Attack: 'ATTACK',
  TargetedAttack: 'TARGETED_ATTACK',
  Tacocat: 'TACOCAT',
  HairyPotatoCat: 'HAIRY_POTATO_CAT',
  Cattermelon: 'CATTERMELON',
  RainbowRalphingCat: 'RAINBOW_RALPHING_CAT',
  RageCat: 'RAGE_CAT',
  FeralCat: 'FERAL_CAT',
  SeeTheFuture: 'SEE_THE_FUTURE',
  AlterTheFuture: 'ALTER_THE_FUTURE',
  Shuffle: 'SHUFFLE',
  DrawFromBottom: 'DRAW_FROM_BOTTOM',
  Favor: 'FAVOR',
  Nope: 'NOPE',
} as const;
export type CardType = (typeof CardType)[keyof typeof CardType];

export const TurnState = {
  NotStarted: 'NOT_STARTED',
  Normal: 'NORMAL',
  SeeingTheFuture: 'SEEING_THE_FUTURE',
  AlteringTheFuture: 'ALTERING_THE_FUTURE',
  AwaitingKittenPlacement: 'AWAITING_KITTEN_PLACEMENT',
  AwaitingFavor: 'AWAITING_FAVOR',
  AwaitingDiscardTake: 'AWAITING_DISCARD_TAKE',
  AcceptingNopes: 'ACCEPTING_NOPES',
  GameOver: 'GAME_OVER',
} as const;
export type TurnState = (typeof TurnState)[keyof typeof TurnState];

export const CardDisplayName: Record<CardType, string> = {
  DEFUSE: 'Defuse',
  EXPLODING_KITTEN: 'Exploding Kitten',
  SKIP: 'Skip',
  ATTACK: 'Attack',
  TARGETED_ATTACK: 'Targeted Attack',
  TACOCAT: 'Tacocat',
  HAIRY_POTATO_CAT: 'Hairy Potato Cat',
  CATTERMELON: 'Cattermelon',
  RAINBOW_RALPHING_CAT: 'Rainbow-Ralphing Cat',
  RAGE_CAT: 'Rage Cat',
  FERAL_CAT: 'Feral Cat',
  SEE_THE_FUTURE: 'See the Future',
  ALTER_THE_FUTURE: 'Alter the Future',
  SHUFFLE: 'Shuffle',
  DRAW_FROM_BOTTOM: 'Draw from Bottom',
  FAVOR: 'Favor',
  NOPE: 'Nope',
};

export function isCatCard(card: CardType): boolean {
  return (
    card === 'TACOCAT' ||
    card === 'HAIRY_POTATO_CAT' ||
    card === 'CATTERMELON' ||
    card === 'RAINBOW_RALPHING_CAT' ||
    card === 'RAGE_CAT' ||
    card === 'FERAL_CAT'
  );
}

export function isPlayableAlone(card: CardType): boolean {
  return (
    card === 'SKIP' ||
    card === 'ATTACK' ||
    card === 'TARGETED_ATTACK' ||
    card === 'SEE_THE_FUTURE' ||
    card === 'ALTER_THE_FUTURE' ||
    card === 'SHUFFLE' ||
    card === 'DRAW_FROM_BOTTOM' ||
    card === 'FAVOR'
  );
}
