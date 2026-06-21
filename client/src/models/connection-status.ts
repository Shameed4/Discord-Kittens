export const ConnectionStatus = {
  Connecting: 'Connecting...',
  Connected: 'Connected',
  Reconnecting: 'Reconnecting...',
  Disconnected: 'Disconnected',
} as const;
export type ConnectionStatus =
  (typeof ConnectionStatus)[keyof typeof ConnectionStatus];
