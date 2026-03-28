export const CardType = {
  Defuse: 'DEFUSE',
  ExplodingKitten: 'EXPLODING_KITTEN',
  Skip: 'SKIP',
  Cat: 'CAT',
  SeeTheFuture: 'SEE_THE_FUTURE',
} as const;
export type CardType = (typeof CardType)[keyof typeof CardType];

export const TurnStateType = {
  NotStarted: 'NOT_STARTED',
  Normal: 'NORMAL',
  GameOver: 'GAME_OVER',
  AwaitingKittenPlacement: 'AWAITING_KITTEN_PLACEMENT',
  SeeingTheFuture: 'SEEING_THE_FUTURE',
} as const;
export type TurnStateType = (typeof TurnStateType)[keyof typeof TurnStateType];
