import type { CardType } from './game-enums';

export interface GameState {
  playerId: number;
  turnId: number;
  deckSize: number;
  players: [];
  turnState: 'NORMAL';
  hand: [];

  future?: CardType[];
  err?: string;
}

export interface PlayerState {
  cardCount: number;
  isAlive: boolean;
  isOnline: boolean;
}
