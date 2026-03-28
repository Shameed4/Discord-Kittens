export const ConnectionStatus = {
  Connecting: 'Connecting...',
  Connected: 'Connected',
  Disconnected: 'Disconnected',
} as const;
export type ConnectionStatus =
  (typeof ConnectionStatus)[keyof typeof ConnectionStatus];
