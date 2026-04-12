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
          className="rounded-full border border-green-400/30 bg-gradient-to-r from-green-600 to-emerald-700 px-8 py-2.5 font-bold text-white shadow-lg transition-all hover:from-green-500 hover:to-emerald-600"
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
      <div className="mt-3 flex justify-center text-xs font-semibold tracking-widest text-purple-800 uppercase">
        {waitingFor ? `Waiting for Player ${waitingFor.id}…` : ''}
      </div>
    );
  }

  const selectedCards = selectedIndices.map((i) => hand[i]);
  const singleAction =
    selectedIndices.length === 1 && isPlayableAlone(hand[selectedIndices[0]]);

  // matching combo: all same type, or exactly one cat type mixed with feral cats
  const isMatchingCombo = (cards: CardType[]): boolean => {
    if (cards.length < 2) return false;
    const types = new Set(cards);
    if (types.size === 1) return true;
    if (
      types.size === 2 &&
      cards.every(isCatCard) &&
      cards.includes('FERAL_CAT')
    )
      return true;
    return false;
  };
  const validCombo =
    (selectedIndices.length === 2 || selectedIndices.length === 3
      ? isMatchingCombo(selectedCards)
      : false) ||
    (selectedIndices.length === 5 && new Set(selectedCards).size === 5);
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
      <div className="flex flex-wrap justify-center gap-3">
        {/* Draw button */}
        <button
          onClick={() => onAction({ action: 'DRAW_CARD' })}
          className="animate-glow-pulse rounded-full border border-blue-400/40 bg-gradient-to-r from-blue-600 to-blue-800 px-6 py-2 font-bold text-white shadow-lg transition-all hover:from-blue-500 hover:to-blue-700"
        >
          Draw Card
        </button>

        {/* Play / Combo button */}
        {(singleAction || validCombo) && (
          <button
            onClick={handlePlay}
            disabled={!canConfirm}
            className={`rounded-full border px-6 py-2 font-bold text-white transition-all ${
              canConfirm
                ? 'border-violet-400/40 bg-gradient-to-r from-violet-600 to-purple-800 shadow-lg hover:from-violet-500 hover:to-purple-700'
                : 'cursor-not-allowed border-gray-700 bg-gray-800 opacity-50'
            }`}
          >
            {singleAction
              ? `Play ${CardDisplayName[hand[selectedIndices[0]]]}`
              : `Play ${selectedIndices.length}-Card Combo`}
          </button>
        )}

        {/* Invalid selection hint */}
        {selectedIndices.length > 0 && !singleAction && !validCombo && (
          <span className="self-center text-xs font-semibold text-purple-700">
            Select 2-3 matching cards, or 5 unique cards for a combo
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
          <span className="text-xs font-semibold text-purple-600">
            Request card:
          </span>
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
