export const CardType = {
  Defuse: 'DEFUSE',
  ExplodingKitten: 'EXPLODING_KITTEN',
  Skip: 'SKIP',
  Attack: 'ATTACK',
  TargetedAttack: 'TARGETED_ATTACK',
  Cat1: 'CAT1',
  Cat2: 'CAT2',
  Cat3: 'CAT3',
  Cat4: 'CAT4',
  Cat5: 'CAT5',
  FeralCat: 'FERAL_CAT',
  SeeTheFuture: 'SEE_THE_FUTURE',
  AlterTheFuture: 'ALTER_THE_FUTURE',
  Shuffle: 'SHUFFLE',
  DrawFromBottom: 'DRAW_FROM_BOTTOM',
  Favor: 'FAVOR',
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
  GameOver: 'GAME_OVER',
} as const;
export type TurnState = (typeof TurnState)[keyof typeof TurnState];

export const CardDisplayName: Record<CardType, string> = {
  DEFUSE: 'Defuse',
  EXPLODING_KITTEN: 'Exploding Kitten',
  SKIP: 'Skip',
  ATTACK: 'Attack',
  TARGETED_ATTACK: 'Targeted Attack',
  CAT1: 'Taco Cat',
  CAT2: 'Beard Cat',
  CAT3: 'Hairy Potato Cat',
  CAT4: 'Cattermelon',
  CAT5: 'Rainbow-Ralphing Cat',
  FERAL_CAT: 'Feral Cat',
  SEE_THE_FUTURE: 'See the Future',
  ALTER_THE_FUTURE: 'Alter the Future',
  SHUFFLE: 'Shuffle',
  DRAW_FROM_BOTTOM: 'Draw from Bottom',
  FAVOR: 'Favor',
};

export function isCatCard(card: CardType): boolean {
  return (
    card === 'CAT1' ||
    card === 'CAT2' ||
    card === 'CAT3' ||
    card === 'CAT4' ||
    card === 'CAT5' ||
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
