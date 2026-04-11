import { useEffect, useRef, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import type { GameState } from '../../models/game-state';
import type { ActionRequest } from '../../models/player-action';
import { ConnectionStatus } from '../../models/connection-status';
import PlayerCard from './components/PlayerCard';
import HandCard from './components/HandCard';
import ActionBar from './components/ActionBar';
import LastActionBanner from './components/LastActionBanner';
import FutureViewer from './components/FutureViewer';
import FutureReorder from './components/FutureReorder';
import KittenPlacer from './components/KittenPlacer';
import FavorGiver from './components/FavorGiver';
import DiscardPicker from './components/DiscardPicker';
import GameOverOverlay from './components/GameOverOverlay';
import { isCatCard, isPlayableAlone } from '../../models/game-enums';

export default function GamePage() {
  const navigate = useNavigate();
  const location = useLocation();
  const lobbyName = location.state?.lobbyName || '';

  const ws = useRef<WebSocket | null>(null);
  const [gameState, setGameState] = useState<GameState | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>(ConnectionStatus.Connecting);
  const [selectedIndices, setSelectedIndices] = useState<number[]>([]);

  useEffect(() => {
    if (!lobbyName) {
      navigate('/');
      return;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws?lobby=${encodeURIComponent(lobbyName)}`;
    const socket = new WebSocket(wsUrl);
    ws.current = socket;

    socket.onopen = () => setConnectionStatus(ConnectionStatus.Connected);
    socket.onmessage = (event) => {
      try {
        setGameState(JSON.parse(event.data) as GameState);
        setSelectedIndices([]);
      } catch (e) {
        console.error('Failed to parse game state:', e);
      }
    };
    socket.onclose = () => setConnectionStatus(ConnectionStatus.Disconnected);

    return () => {
      socket.close();
      ws.current = null;
    };
  }, [lobbyName, navigate]);

  function sendAction(action: ActionRequest) {
    if (ws.current?.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify(action));
    }
  }

  const handleLeave = () => navigate('/');

  if (!gameState) {
    return (
      <div className="flex h-screen items-center justify-center bg-gray-900 text-gray-400">
        {connectionStatus}
      </div>
    );
  }

  const { playerId, turnId, turnState, hand, players, deckSize, inProgress } = gameState;
  const isMyTurn = playerId === turnId;
  const amFavorTarget = turnState === 'AWAITING_FAVOR' && gameState.targetedPlayer === playerId;
  const opponents = players.filter(p => p.id !== playerId);

  // A card is selectable if it's my turn, in a playable state, and not in favor-giver mode
  const handIsPlayable = isMyTurn && (turnState === 'NORMAL' || turnState === 'SEEING_THE_FUTURE') && !amFavorTarget;

  const handleCardClick = (index: number) => {
    const card = hand[index];
    const isAlreadySelected = selectedIndices.includes(index);

    if (isAlreadySelected) {
      setSelectedIndices(selectedIndices.filter(i => i !== index));
      return;
    }

    // Action cards: select exclusively
    if (isPlayableAlone(card)) {
      setSelectedIndices([index]);
      return;
    }

    // Cat cards: add to selection
    if (isCatCard(card)) {
      // Don't mix with action card selections
      const currentlySelectedCards = selectedIndices.map(i => hand[i]);
      if (currentlySelectedCards.some(isPlayableAlone)) {
        setSelectedIndices([index]);
      } else {
        setSelectedIndices([...selectedIndices, index]);
      }
    }
  };

  return (
    <div className="flex flex-col h-screen bg-gray-900 text-white">
      {/* Zone 1: Opponents */}
      <div className="flex justify-center gap-6 p-4 bg-gray-800 border-b border-gray-700 flex-wrap min-h-[140px]">
        {opponents.length === 0 ? (
          <span className="text-gray-500 text-sm self-center">Waiting for other players...</span>
        ) : (
          opponents.map(p => (
            <PlayerCard key={p.id} playerState={p} gameState={gameState} />
          ))
        )}
      </div>

      {/* Zone 2: Center info */}
      <div className="flex-1 flex flex-col items-center justify-center gap-4 p-4 overflow-y-auto">
        <LastActionBanner lastAction={gameState.lastAction} />

        {inProgress && (
          <div className="text-gray-400 text-sm">
            Deck: <span className="text-white font-bold">{deckSize}</span> cards
            {gameState.underAttack && (
              <span className="ml-3 text-orange-400 font-semibold">
                ⚡ Under attack — {gameState.turnsToTake} turn{gameState.turnsToTake !== 1 ? 's' : ''} left
              </span>
            )}
          </div>
        )}

        {gameState.err && (
          <div className="bg-red-900 border border-red-600 text-red-200 rounded-lg px-4 py-2 text-sm">
            {gameState.err}
          </div>
        )}

        {/* State-specific UIs */}
        {turnState === 'SEEING_THE_FUTURE' && isMyTurn && gameState.future && (
          <FutureViewer cards={gameState.future} />
        )}

        {turnState === 'ALTERING_THE_FUTURE' && isMyTurn && gameState.future && (
          <FutureReorder
            cards={gameState.future}
            onConfirm={(newOrder) => sendAction({ action: 'ALTER_FUTURE', alterFutureOrder: newOrder })}
          />
        )}

        {turnState === 'AWAITING_KITTEN_PLACEMENT' && isMyTurn && (
          <KittenPlacer
            deckSize={deckSize}
            onPlace={(index) => sendAction({ action: 'PLACE_KITTEN', placeKittenIndex: index })}
          />
        )}

        {amFavorTarget && (
          <FavorGiver
            hand={hand}
            requesterPlayerId={turnId}
            onGive={(cardIndex) => sendAction({ action: 'GIVE_FAVOR', useCardIndex: cardIndex })}
          />
        )}

        {turnState === 'AWAITING_DISCARD_TAKE' && isMyTurn && gameState.discardOptions && (
          <DiscardPicker
            options={gameState.discardOptions}
            onPick={(card) => sendAction({ action: 'TAKE_FROM_DISCARD', requestedCard: card })}
          />
        )}

        {!inProgress && (
          <div className="text-gray-400 text-sm">
            Lobby: <span className="text-white font-bold">{lobbyName}</span>
            &nbsp;·&nbsp; {players.length} player{players.length !== 1 ? 's' : ''} connected
          </div>
        )}
      </div>

      {/* Zone 3: Hand + action bar */}
      <div className="bg-gray-800 border-t border-gray-700 p-4">
        <div className="flex gap-2 overflow-x-auto pb-2">
          {hand.map((card, i) => (
            <HandCard
              key={i}
              card={card}
              index={i}
              isSelected={selectedIndices.includes(i)}
              isPlayable={handIsPlayable}
              onClick={handleCardClick}
            />
          ))}
          {hand.length === 0 && inProgress && (
            <span className="text-gray-500 text-sm self-center">No cards in hand</span>
          )}
        </div>

        <ActionBar
          gameState={gameState}
          selectedIndices={selectedIndices}
          onAction={(action) => {
            sendAction(action);
            setSelectedIndices([]);
          }}
        />
      </div>

      {turnState === 'GAME_OVER' && (
        <GameOverOverlay players={players} onLeave={handleLeave} />
      )}

      {/* Connection status indicator */}
      {connectionStatus !== ConnectionStatus.Connected && (
        <div className="fixed top-2 right-2 bg-gray-800 border border-gray-600 text-gray-300 text-xs rounded-full px-3 py-1">
          {connectionStatus}
        </div>
      )}
    </div>
  );
}
