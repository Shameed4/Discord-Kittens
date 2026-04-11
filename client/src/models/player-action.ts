export const ActionType = {
  StartGame:       'START_GAME',
  PlayCard:        'PLAY_CARD',
  DrawCard:        'DRAW_CARD',
  PlaceKitten:     'PLACE_KITTEN',
  Disconnect:      'DISCONNECT',
  AlterFuture:     'ALTER_FUTURE',
  GiveFavor:       'GIVE_FAVOR',
  Combo:           'COMBO',
  TakeFromDiscard: 'TAKE_FROM_DISCARD',
} as const;
export type ActionType = (typeof ActionType)[keyof typeof ActionType];

interface StartGameAction  { action: 'START_GAME' }
interface DrawCardAction   { action: 'DRAW_CARD' }
interface DisconnectAction { action: 'DISCONNECT' }

interface PlayCardAction {
  action:          'PLAY_CARD';
  useCardIndex:    number;
  targetedPlayer?: number;
}

interface PlaceKittenAction {
  action:           'PLACE_KITTEN';
  placeKittenIndex: number;
}

interface AlterFutureAction {
  action:           'ALTER_FUTURE';
  alterFutureOrder: number[];
}

interface GiveFavorAction {
  action:       'GIVE_FAVOR';
  useCardIndex: number;
}

interface ComboAction {
  action:          'COMBO';
  comboIndices:    number[];
  targetedPlayer?: number;
  requestedCard?:  string;
}

interface TakeFromDiscardAction {
  action:        'TAKE_FROM_DISCARD';
  requestedCard: string;
}

export type ActionRequest =
  | StartGameAction
  | DrawCardAction
  | DisconnectAction
  | PlayCardAction
  | PlaceKittenAction
  | AlterFutureAction
  | GiveFavorAction
  | ComboAction
  | TakeFromDiscardAction;
