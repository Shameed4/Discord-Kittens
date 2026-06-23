import { useEffect, useRef, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import type { GameState } from '../../models/game-state';
import type { ActionRequest } from '../../models/player-action';
import { ConnectionStatus } from '../../models/connection-status';
import PlayerSeat from './components/PlayerSeat';
import HandCard from './components/HandCard';
import ActionBar from './components/ActionBar';
import LastActionBanner from './components/LastActionBanner';
import FutureViewer from './components/FutureViewer';
import FutureReorder from './components/FutureReorder';
import KittenPlacer from './components/KittenPlacer';
import FavorGiver from './components/FavorGiver';
import DiscardPicker from './components/DiscardPicker';
import GameOverOverlay from './components/GameOverOverlay';
import GameLog from './components/GameLog';
import NopeBanner from './components/NopeBanner';
import { getSeatPositions } from './table-utils';
import { useNopeCountdown } from './use-nope-countdown';
import {
  getUsername,
  getUserId,
  getInstanceId,
  setupDiscordSdk,
} from '../../discord/sdk';

export default function GamePage() {
  const navigate = useNavigate();
  const location = useLocation();
  // In Discord, auto create/join lobby based on instance id
  const discordInstanceId = getInstanceId();
  const lobbyName = location.state?.lobbyName || discordInstanceId || '';
  const autoCreate = !location.state?.lobbyName && Boolean(discordInstanceId);

  const ws = useRef<WebSocket | null>(null);
  const [gameState, setGameState] = useState<GameState | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>(
    ConnectionStatus.Connecting,
  );
  const [selectedIndices, setSelectedIndices] = useState<number[]>([]);

  // Nope window: countdown ticks only while the server is accepting nopes.
  const isAcceptingNopes = gameState?.turnState === 'ACCEPTING_NOPES';
  const isNoped = gameState?.isNoped ?? false;
  const { remaining: nopeRemaining, fraction: nopeFraction } = useNopeCountdown(
    isAcceptingNopes ? gameState?.nopeDeadline : undefined,
  );

  useEffect(() => {
    if (!lobbyName) {
      navigate('/');
      return;
    }
    let cancelled = false; // set on unmount/leave so we stop auto-reconnecting
    let attempt = 0; // reconnect attempt counter, drives exponential backoff
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

    const connect = async () => {
      // Wait for the Discord auth handshake to resolve so getUsername() returns
      // the real name. The backend locks in the name at join time, so connecting
      // before auth completes would permanently leave us as "Player N". Resolves
      // to null (and we join nameless) outside a Discord context.
      await setupDiscordSdk().catch(() => null);
      if (cancelled) return;

      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const params = new URLSearchParams({ lobby: lobbyName });
      if (autoCreate) params.set('create', '1');
      const username = getUsername();
      if (username) params.set('username', username);
      const userId = getUserId();
      if (userId) params.set('userId', userId);
      const socket = new WebSocket(
        `${protocol}//${window.location.host}/ws?${params.toString()}`,
      );
      ws.current = socket;
      socket.onopen = () => {
        attempt = 0; // successful connection — reset backoff
        setConnectionStatus(ConnectionStatus.Connected);
      };
      socket.onmessage = (event) => {
        try {
          setGameState(JSON.parse(event.data) as GameState);
          setSelectedIndices([]);
        } catch (e) {
          console.error('Failed to parse game state:', e);
        }
      };
      socket.onclose = (event) => {
        if (cancelled) return; // we closed it intentionally (unmount/leave)

        // 1000: clean server close (our own quit, or a duplicate-tab takeover —
        //       another connection is now authoritative, so don't fight it).
        // 4000: server rejected the join (e.g. game in progress with no seat to
        //       reclaim). Surface the reason and stop.
        if (event.code === 1000 || event.code === 4000) {
          setConnectionStatus(ConnectionStatus.Disconnected);
          return;
        }

        // Unexpected drop (network loss, code 1006, etc.) — reconnect with
        // exponential backoff capped at 15s, plus jitter to avoid thundering herd.
        const delay =
          Math.min(1000 * 2 ** attempt, 15000) + Math.random() * 300;
        attempt += 1;
        setConnectionStatus(ConnectionStatus.Reconnecting);
        reconnectTimer = setTimeout(() => {
          if (!cancelled) connect();
        }, delay);
      };
    };

    connect();

    return () => {
      cancelled = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      ws.current?.close();
      ws.current = null;
    };
  }, [lobbyName, autoCreate, navigate]);

  function sendAction(action: ActionRequest) {
    if (ws.current?.readyState === WebSocket.OPEN)
      ws.current.send(JSON.stringify(action));
  }

  const handleLeave = () => navigate('/');

  if (!gameState) {
    return (
      <div className="flex h-full items-center justify-center bg-[#0d0720] text-xs font-bold tracking-widest text-purple-500 uppercase">
        {connectionStatus}
      </div>
    );
  }

  const { playerId, turnId, turnState, hand, players, deckSize, inProgress } =
    gameState;
  const isMyTurn = playerId === turnId;
  const amFavorTarget =
    turnState === 'AWAITING_FAVOR' && gameState.targetedPlayer === playerId;
  const handIsPlayable =
    isMyTurn &&
    (turnState === 'NORMAL' || turnState === 'SEEING_THE_FUTURE') &&
    !amFavorTarget;

  const turnPlayerName =
    players.find((p) => p.id === turnId)?.name ?? `Player ${turnId}`;
  const localPlayerIndex = players.findIndex((p) => p.id === playerId);
  const seatPositions = getSeatPositions(
    players.length,
    localPlayerIndex < 0 ? 0 : localPlayerIndex,
  );
  // Scale seats down at high player counts so they fit the perimeter
  const seatScale = players.length <= 4 ? 1 : players.length <= 7 ? 0.85 : 0.7;

  const handleCardClick = (index: number) => {
    const card = hand[index];
    if (card === 'EXPLODING_KITTEN') return;
    if (selectedIndices.includes(index)) {
      setSelectedIndices(selectedIndices.filter((i) => i !== index));
    } else {
      setSelectedIndices([...selectedIndices, index]);
    }
  };

  return (
    <div className="flex h-full overflow-hidden bg-[#0d0720]">
      {/* Nope-window mood tint: amber while the action stands, red once noped.
          Vignette (clear center) keeps the table readable; never blocks clicks. */}
      {isAcceptingNopes && (
        <div
          aria-hidden
          className="pointer-events-none fixed inset-0 z-40 animate-pulse"
          style={{
            background: `radial-gradient(ellipse at center, transparent 45%, ${
              isNoped ? 'rgba(239,68,68,0.30)' : 'rgba(245,158,11,0.24)'
            } 100%)`,
          }}
        />
      )}
      {/* Main game area: table on top, hand at the bottom */}
      <div className="flex min-w-0 flex-1 flex-col items-center justify-center gap-3 p-3">
        {/* ── Round table ── */}
        {/*
        Outer div is a square container. The felt circle is inset by 15% on each
        side, leaving 15% on each edge for seat content to overflow into.
        All player seats are positioned with percentage coords relative to this
        outer container so they land right on the felt's perimeter.
      */}
        <div
          style={{
            position: 'relative',
            width: 'min(62vw, 620px)',
            height: 'min(46vh, 360px)',
          }}
        >
          {/* Felt circle */}
          <div
            style={{
              position: 'absolute',
              inset: '15%',
              borderRadius: '50%',
              background:
                'radial-gradient(ellipse at 38% 38%, #1a5c2a 40%, #0e3d1a 100%)',
              border: '4px solid #8B6914',
              boxShadow:
                '0 0 40px rgba(109,40,217,0.25), inset 0 0 30px rgba(0,0,0,0.4)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            {/* Subtle purple glow overlay */}
            <div
              style={{
                position: 'absolute',
                inset: 0,
                borderRadius: '50%',
                background:
                  'radial-gradient(ellipse, rgba(76,29,149,0.14) 0%, transparent 70%)',
                pointerEvents: 'none',
              }}
            />

            {/* Center content */}
            <div className="relative z-10 flex max-w-full flex-col items-center gap-2 px-3">
              <LastActionBanner lastAction={gameState.lastAction} />

              {/* Nope window countdown + verdict (visible to everyone) */}
              {isAcceptingNopes && (
                <NopeBanner
                  isNoped={isNoped}
                  remaining={nopeRemaining}
                  fraction={nopeFraction}
                />
              )}

              {/* Deck + discard piles */}
              {inProgress && (
                <div className="flex items-end gap-4">
                  <div className="flex flex-col items-center gap-1">
                    <span className="text-[8px] font-bold tracking-widest text-purple-800 uppercase">
                      Deck
                    </span>
                    <button
                      onClick={() => {
                        if (isMyTurn && turnState === 'NORMAL')
                          sendAction({ action: 'DRAW_CARD' });
                      }}
                      style={{
                        position: 'relative',
                        width: 44,
                        height: 62,
                        borderRadius: 6,
                        background:
                          'radial-gradient(ellipse at 35% 35%, #1e1060 0%, #06020f 100%)',
                        border: '2px solid #7c3aed',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontSize: 22,
                        boxShadow:
                          '3px 3px 0 #1e1060, 5px 5px 0 #1e1060, 0 0 16px rgba(109,40,217,0.3)',
                        cursor:
                          isMyTurn && turnState === 'NORMAL'
                            ? 'pointer'
                            : 'default',
                      }}
                    >
                      😼
                      <span
                        style={{
                          position: 'absolute',
                          bottom: 3,
                          right: 3,
                          fontSize: 9,
                          fontWeight: 800,
                          color: '#a78bfa',
                          background: 'rgba(0,0,0,0.6)',
                          padding: '1px 3px',
                          borderRadius: 3,
                        }}
                      >
                        {deckSize}
                      </span>
                    </button>
                  </div>
                  <div className="flex flex-col items-center gap-1">
                    <span className="text-[8px] font-bold tracking-widest text-purple-800 uppercase">
                      Discard
                    </span>
                    <div
                      style={{
                        width: 44,
                        height: 62,
                        borderRadius: 6,
                        background: '#111827',
                        border: '2px solid #374151',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontSize: 22,
                      }}
                    >
                      🃏
                    </div>
                  </div>
                </div>
              )}

              {/* Under-attack notice */}
              {inProgress && gameState.underAttack && (
                <div className="rounded-full border border-orange-700/50 bg-orange-950/50 px-3 py-0.5 text-[9px] font-bold text-orange-400">
                  ⚡ Draw {gameState.turnsToTake} more
                </div>
              )}

              {/* Error message */}
              {gameState.err && (
                <div className="max-w-[180px] rounded-full border border-red-800 bg-red-950/70 px-3 py-0.5 text-center text-[9px] font-semibold text-red-300">
                  {gameState.err}
                </div>
              )}

              {/* Pre-game lobby info */}
              {!inProgress && (
                <div className="text-center text-[10px] leading-snug font-semibold text-purple-800">
                  <div className="font-bold text-purple-600">{lobbyName}</div>
                  <div>
                    {players.length} player{players.length !== 1 ? 's' : ''}{' '}
                    connected
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Player seats — absolutely positioned around the felt perimeter */}
          {players.map((player, idx) => {
            const pos = seatPositions[idx];
            return (
              <PlayerSeat
                key={player.id}
                playerState={player}
                gameState={gameState}
                playerIndex={idx}
                seatScale={seatScale}
                mirrored={pos.x > 50}
                style={{
                  position: 'absolute',
                  left: `${pos.x}%`,
                  top: `${pos.y}%`,
                  transform: 'translate(-50%, -50%)',
                  zIndex: player.id === playerId ? 10 : 5,
                }}
              />
            );
          })}

          {/* State-specific overlays — centered over the entire table area */}
          {turnState === 'SEEING_THE_FUTURE' &&
            isMyTurn &&
            gameState.future && (
              <div className="absolute inset-0 z-50 flex items-center justify-center">
                <FutureViewer cards={gameState.future} />
              </div>
            )}
          {turnState === 'ALTERING_THE_FUTURE' &&
            isMyTurn &&
            gameState.future && (
              <div className="absolute inset-0 z-50 flex items-center justify-center">
                <FutureReorder
                  cards={gameState.future}
                  onConfirm={(newOrder) =>
                    sendAction({
                      action: 'ALTER_FUTURE',
                      alterFutureOrder: newOrder,
                    })
                  }
                />
              </div>
            )}
          {turnState === 'AWAITING_KITTEN_PLACEMENT' && isMyTurn && (
            <div className="absolute inset-0 z-50 flex items-center justify-center">
              <KittenPlacer
                deckSize={deckSize}
                onPlace={(index) =>
                  sendAction({
                    action: 'PLACE_KITTEN',
                    placeKittenIndex: index,
                  })
                }
              />
            </div>
          )}
          {amFavorTarget && (
            <div className="absolute inset-0 z-50 flex items-center justify-center">
              <FavorGiver
                hand={hand}
                requesterName={turnPlayerName}
                onGive={(cardIndex) =>
                  sendAction({ action: 'GIVE_FAVOR', useCardIndex: cardIndex })
                }
              />
            </div>
          )}
          {turnState === 'AWAITING_DISCARD_TAKE' &&
            isMyTurn &&
            gameState.discardOptions && (
              <div className="absolute inset-0 z-50 flex items-center justify-center">
                <DiscardPicker
                  options={gameState.discardOptions}
                  onPick={(card) =>
                    sendAction({
                      action: 'TAKE_FROM_DISCARD',
                      requestedCard: card,
                    })
                  }
                />
              </div>
            )}
        </div>

        {/* ── Hand zone (below table) — cards and actions side by side ── */}
        {/*
        Fixed 2:1 proportions (both flex items have basis 0 via `flex-[2]`/`flex-1`
        + min-w-0), so the card area keeps the same width no matter which buttons or
        hints the action bar is showing — cards never shift around. Cards wrap to a
        second row when the hand is large rather than getting clipped.
      */}
        <div className="flex w-full max-w-4xl items-center justify-center gap-3 px-2">
          <div className="flex min-w-0 flex-[4] flex-wrap content-center justify-center gap-1.5 pb-1">
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
              <span className="self-center text-xs font-semibold text-purple-800">
                No cards in hand
              </span>
            )}
          </div>
          <div className="flex min-w-0 flex-1 justify-center">
            <ActionBar
              gameState={gameState}
              selectedIndices={selectedIndices}
              nopeRemaining={nopeRemaining}
              onAction={(action) => {
                sendAction(action);
                setSelectedIndices([]);
              }}
            />
          </div>
        </div>

        {/* Game over overlay */}
        {turnState === 'GAME_OVER' && (
          <GameOverOverlay players={players} onLeave={handleLeave} />
        )}
      </div>

      {/* Game log sidebar */}
      <GameLog log={gameState.log ?? []} />

      {/* Connection status badge */}
      {connectionStatus !== ConnectionStatus.Connected && (
        <div
          className="fixed rounded-full border border-purple-800 bg-purple-950/90 px-3 py-1 text-xs font-semibold text-purple-300"
          style={{
            top: 'calc(0.5rem + var(--sait))',
            right: 'calc(0.5rem + var(--sair))',
          }}
        >
          {connectionStatus}
        </div>
      )}
    </div>
  );
}
