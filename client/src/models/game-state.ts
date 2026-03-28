import type { CardType, TurnStateType } from './game-enums';

export interface GameState {
  playerId: number;
  turnId: number;
  deckSize: number;
  players: PlayerState[];
  turnState: TurnStateType;
  hand: CardType[];

  future?: CardType[3];
  err?: string;
}

export interface PlayerState {
  id: number;
  cardCount: number;
  isAlive: boolean;
  isOnline: boolean;
}
