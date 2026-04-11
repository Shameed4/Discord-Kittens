import type { CardType, TurnState } from './game-enums';

export interface GameState {
  playerId:       number;
  turnId:         number;
  deckSize:       number;
  players:        PlayerState[];
  turnState:      TurnState;
  hand:           CardType[];
  inProgress:     boolean;
  underAttack:    boolean;
  turnsToTake:    number;
  targetedPlayer: number;

  future?:         CardType[];
  discardOptions?: CardType[];
  lastAction?:     string;
  err?:            string;
}

export interface PlayerState {
  id:        number;
  cardCount: number;
  isAlive:   boolean;
  isOnline:  boolean;
}
