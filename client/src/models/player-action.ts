export const ActionType = {
  StartGame: 'START_GAME',
  PlayCard: 'PLAY_CARD',
  DrawCard: 'DRAW_CARD',
  PlaceKitten: 'PLACE_KITTEN',
  Disconnect: 'DISCONNECT',
} as const;

export type ActionType = (typeof ActionType)[keyof typeof ActionType];

interface SimpleAction {
  action:
    | typeof ActionType.StartGame
    | typeof ActionType.DrawCard
    | typeof ActionType.Disconnect;
}

interface IndexedAction {
  action: typeof ActionType.PlayCard | typeof ActionType.PlaceKitten;
  index: number;
}

export type ActionRequest = SimpleAction | IndexedAction;
