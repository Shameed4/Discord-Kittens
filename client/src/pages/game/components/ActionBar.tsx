import { useState } from 'react';
import {
  CardDisplayName,
  isCatCard,
  isPlayableAlone,
  type CardType,
} from '../../../models/game-enums';
import type { GameState } from '../../../models/game-state';
import type { ActionRequest } from '../../../models/player-action';
import TargetSelector from './TargetSelector';

interface ActionBarProps {
  gameState: GameState;
  selectedIndices: number[];
  onAction: (action: ActionRequest) => void;
}

export default function ActionBar({
  gameState,
  selectedIndices,
  onAction,
}: ActionBarProps) {
  const [targetedPlayer, setTargetedPlayer] = useState<number | null>(null);
  const [requestedCard, setRequestedCard] = useState<string>('');

  const { hand, players, playerId, turnId, inProgress, turnState } = gameState;
  const isMyTurn = playerId === turnId;

  // Not started
  if (!inProgress) {
    return (
      <div className="mt-4 flex justify-center">
        <button
          onClick={() => onAction({ action: 'START_GAME' })}
          className="rounded-full bg-gradient-to-r from-green-600 to-emerald-700 px-8 py-2.5 font-bold text-white shadow-lg border border-green-400/30 hover:from-green-500 hover:to-emerald-600 transition-all"
        >
          Start Game
        </button>
      </div>
    );
  }

  if (
    !isMyTurn ||
    (turnState !== 'NORMAL' && turnState !== 'SEEING_THE_FUTURE')
  ) {
    const waitingFor = players.find((p) => p.id === turnId);
    return (
      <div className="mt-3 flex justify-center text-xs font-semibold uppercase tracking-widest text-purple-800">
        {waitingFor ? `Waiting for Player ${waitingFor.id}…` : ''}
      </div>
    );
  }

  const selectedCards = selectedIndices.map((i) => hand[i]);
  const allCats = selectedCards.length > 0 && selectedCards.every(isCatCard);
  const singleAction =
    selectedIndices.length === 1 && isPlayableAlone(hand[selectedIndices[0]]);
  const validCombo =
    allCats &&
    (selectedIndices.length === 2 ||
      selectedIndices.length === 3 ||
      selectedIndices.length === 5);
  const needsTarget =
    singleAction &&
    (hand[selectedIndices[0]] === 'TARGETED_ATTACK' ||
      hand[selectedIndices[0]] === 'FAVOR');
  const comboNeedsTarget = validCombo && selectedIndices.length !== 5;
  const comboNeeds3CardPick = validCombo && selectedIndices.length === 3;

  const canConfirm = (() => {
    if (singleAction) return !needsTarget || targetedPlayer !== null;
    if (validCombo && selectedIndices.length === 2)
      return targetedPlayer !== null;
    if (validCombo && selectedIndices.length === 3)
      return targetedPlayer !== null && requestedCard !== '';
    if (validCombo && selectedIndices.length === 5) return true;
    return false;
  })();

  const handlePlay = () => {
    if (singleAction) {
      const action: ActionRequest =
        needsTarget && targetedPlayer !== null
          ? {
              action: 'PLAY_CARD',
              useCardIndex: selectedIndices[0],
              targetedPlayer,
            }
          : { action: 'PLAY_CARD', useCardIndex: selectedIndices[0] };
      onAction(action);
    } else if (validCombo) {
      if (selectedIndices.length === 5) {
        onAction({ action: 'COMBO', comboIndices: selectedIndices });
      } else if (selectedIndices.length === 3 && targetedPlayer !== null) {
        onAction({
          action: 'COMBO',
          comboIndices: selectedIndices,
          targetedPlayer,
          requestedCard,
        });
      } else if (selectedIndices.length === 2 && targetedPlayer !== null) {
        onAction({
          action: 'COMBO',
          comboIndices: selectedIndices,
          targetedPlayer,
        });
      }
    }
    setTargetedPlayer(null);
    setRequestedCard('');
  };

  return (
    <div className="mt-3 flex flex-col items-center gap-2">
      <div className="flex gap-3 flex-wrap justify-center">
        {/* Draw button */}
        <button
          onClick={() => onAction({ action: 'DRAW_CARD' })}
          className="rounded-full bg-gradient-to-r from-blue-600 to-blue-800 px-6 py-2 font-bold text-white border border-blue-400/40 shadow-lg animate-glow-pulse hover:from-blue-500 hover:to-blue-700 transition-all"
        >
          Draw Card
        </button>

        {/* Play / Combo button */}
        {(singleAction || validCombo) && (
          <button
            onClick={handlePlay}
            disabled={!canConfirm}
            className={`rounded-full px-6 py-2 font-bold text-white border transition-all ${
              canConfirm
                ? 'bg-gradient-to-r from-violet-600 to-purple-800 border-violet-400/40 shadow-lg hover:from-violet-500 hover:to-purple-700'
                : 'bg-gray-800 border-gray-700 opacity-50 cursor-not-allowed'
            }`}
          >
            {singleAction
              ? `Play ${CardDisplayName[hand[selectedIndices[0]]]}`
              : `Play ${selectedIndices.length}-Card Combo`}
          </button>
        )}

        {/* Invalid selection hint */}
        {selectedIndices.length > 0 && !singleAction && !validCombo && (
          <span className="self-center text-xs text-purple-700 font-semibold">
            Select 2, 3, or 5 cat cards for a combo
          </span>
        )}
      </div>

      {/* Target selector */}
      {(needsTarget || comboNeedsTarget) && (
        <TargetSelector
          players={players}
          currentPlayerId={playerId}
          selected={targetedPlayer}
          onSelect={setTargetedPlayer}
        />
      )}

      {/* 3-card combo: card name picker */}
      {comboNeeds3CardPick && targetedPlayer !== null && (
        <div className="mt-1 flex items-center gap-2">
          <span className="text-xs text-purple-600 font-semibold">Request card:</span>
          <select
            value={requestedCard}
            onChange={(e) => setRequestedCard(e.target.value)}
            className="rounded-lg border border-purple-800 bg-purple-950 px-2 py-1 text-sm text-white"
          >
            <option value="">Pick a card…</option>
            {(Object.keys(CardDisplayName) as CardType[]).map((c) => (
              <option key={c} value={c}>
                {CardDisplayName[c]}
              </option>
            ))}
          </select>
        </div>
      )}
    </div>
  );
}
