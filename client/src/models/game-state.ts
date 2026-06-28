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
  isSpectator:    boolean; // watch-only client that joined mid-game (no seat, public info only)

  future?:         CardType[];
  discardOptions?: CardType[];
  isNoped?:        boolean;
  nopeDeadline?:   number;
  lastAction?:     string;
  log?:            string[];
  err?:            string;
}

export interface PlayerState {
  id:        number;
  name:      string;
  avatar:    string; // avatar image URL; "" when the player has none (falls back to an emoji)
  cardCount: number;
  isAlive:   boolean;
  isOnline:  boolean;
}
