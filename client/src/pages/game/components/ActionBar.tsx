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
      <div className="mt-3 flex justify-center">
        <button
          onClick={() => onAction({ action: 'START_GAME' })}
          className="rounded-lg bg-green-600 px-6 py-2 font-bold text-white transition-colors hover:bg-green-500"
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
      <div className="mt-3 flex justify-center text-sm text-gray-400">
        {waitingFor ? `Waiting for Player ${waitingFor.id}...` : ''}
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
      <div className="flex gap-3">
        {/* Draw button — always available on your turn */}
        <button
          onClick={() => onAction({ action: 'DRAW_CARD' })}
          className="rounded-lg bg-indigo-600 px-5 py-2 font-bold text-white transition-colors hover:bg-indigo-500"
        >
          Draw Card
        </button>

        {/* Play / Combo button */}
        {(singleAction || validCombo) && (
          <button
            onClick={handlePlay}
            disabled={!canConfirm}
            className={`rounded-lg px-5 py-2 font-bold text-white transition-colors ${canConfirm ? 'bg-yellow-600 hover:bg-yellow-500' : 'cursor-not-allowed bg-gray-600 opacity-60'}`}
          >
            {singleAction
              ? `Play ${CardDisplayName[hand[selectedIndices[0]]]}`
              : `Play ${selectedIndices.length}-Card Combo`}
          </button>
        )}

        {/* Hint for invalid selection */}
        {selectedIndices.length > 0 && !singleAction && !validCombo && (
          <span className="self-center text-sm text-gray-400">
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
          <span className="text-xs text-gray-400">Request card:</span>
          <select
            value={requestedCard}
            onChange={(e) => setRequestedCard(e.target.value)}
            className="rounded border border-gray-600 bg-gray-700 px-2 py-1 text-sm text-white"
          >
            <option value="">Pick a card...</option>
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
