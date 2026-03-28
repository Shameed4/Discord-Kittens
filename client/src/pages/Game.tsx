import { useEffect, useRef, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import type { GameState } from '../models/game-state';
import { ConnectionStatus } from '../models/connection-status';

export default function GamePage() {
  const navigate = useNavigate();
  const location = useLocation();

  const lobbyName = location.state?.lobbyName || '';

  // 1. Swap useState for useRef here
  const ws = useRef<WebSocket | null>(null);

  const [gameState, setGameState] = useState<GameState | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>(
    ConnectionStatus.Connecting,
  );

  useEffect(() => {
    if (!lobbyName) {
      navigate('/');
      return;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws?lobby=${encodeURIComponent(lobbyName)}`;

    const socket = new WebSocket(wsUrl);

    // 2. Assign the socket to the ref's .current property (synchronous, but doesn't trigger a render!)
    ws.current = socket;

    socket.onopen = () => {
      setConnectionStatus(ConnectionStatus.Connected);
    };

    socket.onmessage = (event) => {
      try {
        const data: GameState = JSON.parse(event.data);
        setGameState(data);
      } catch (e) {
        console.error('Failed to parse JSON:', e);
      }
    };

    socket.onclose = () => {
      setConnectionStatus(ConnectionStatus.Disconnected);
    };

    return () => {
      socket.close();
      ws.current = null;
    };
  }, [lobbyName, navigate]);

  const handleLeave = () => {
    navigate('/');
  };

  const handleClick = () => {
    // 3. Access the socket using ws.current
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send('click');
    }
  };

  return (
    <div className="flex flex-col">
      <h2>{connectionStatus}</h2>
      <div>{JSON.stringify(gameState)}</div>
    </div>
  );
}
