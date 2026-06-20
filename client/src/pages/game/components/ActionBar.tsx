import { useState } from 'react';
import {
  CardDisplayName,
  isCatCard,
  isPlayableAlone,
  type CardType,
} from '../../../models/game-enums';
import type { GameState } from '../../../models/game-state';
import type { ActionRequest } from '../../../models/player-action';
import TargetModal from './TargetModal';

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
  const [showTargetModal, setShowTargetModal] = useState(false);

  const { hand, players, playerId, turnId, inProgress, turnState } = gameState;
  const isMyTurn = playerId === turnId;

  // Not started
  if (!inProgress) {
    return (
      <div className="flex justify-center">
        <button
          onClick={() => onAction({ action: 'START_GAME' })}
          className="rounded-full border border-green-400/30 bg-gradient-to-r from-green-600 to-emerald-700 px-4 py-1.5 text-sm font-bold text-white shadow-lg transition-all hover:from-green-500 hover:to-emerald-600"
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
      <div className="flex justify-center text-center text-[10px] font-semibold tracking-wide break-words text-purple-800 uppercase">
        {waitingFor ? `Waiting for ${waitingFor.name}…` : ''}
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
  const needsTargetMove = needsTarget || comboNeedsTarget;
  const needs3CardPick = validCombo && selectedIndices.length === 3;

  // Title/prompt shown in the target-selection popup.
  const modalTitle = singleAction
    ? CardDisplayName[hand[selectedIndices[0]]]
    : `${selectedIndices.length}-Card Combo`;
  const modalPrompt = needs3CardPick
    ? 'Name a card to steal from a player'
    : comboNeedsTarget
      ? 'Steal a random card from a player'
      : hand[selectedIndices[0]] === 'FAVOR'
        ? 'Choose who must give you a card'
        : 'Choose a player to target';

  // Targeted moves open a popup to pick the target (and card for 3-card combos),
  // committing only after the selection. Everything else commits immediately.
  const handlePlay = () => {
    if (needsTargetMove) {
      setShowTargetModal(true);
      return;
    }
    if (singleAction) {
      onAction({ action: 'PLAY_CARD', useCardIndex: selectedIndices[0] });
    } else if (validCombo && selectedIndices.length === 5) {
      onAction({ action: 'COMBO', comboIndices: selectedIndices });
    }
  };

  const commitTargetedMove = (
    targetedPlayer: number,
    requestedCard?: string,
  ) => {
    if (singleAction) {
      onAction({
        action: 'PLAY_CARD',
        useCardIndex: selectedIndices[0],
        targetedPlayer,
      });
    } else if (selectedIndices.length === 3) {
      onAction({
        action: 'COMBO',
        comboIndices: selectedIndices,
        targetedPlayer,
        requestedCard,
      });
    } else if (selectedIndices.length === 2) {
      onAction({
        action: 'COMBO',
        comboIndices: selectedIndices,
        targetedPlayer,
      });
    }
    setShowTargetModal(false);
  };

  return (
    <div className="flex flex-col items-center gap-1.5">
      <div className="flex flex-wrap justify-center gap-2">
        {/* Draw button */}
        <button
          onClick={() => onAction({ action: 'DRAW_CARD' })}
          className="animate-glow-pulse rounded-full border border-blue-400/40 bg-gradient-to-r from-blue-600 to-blue-800 px-3 py-1.5 text-xs font-bold text-white shadow-lg transition-all hover:from-blue-500 hover:to-blue-700"
        >
          Draw Card
        </button>

        {/* Play / Combo button */}
        {(singleAction || validCombo) && (
          <button
            onClick={handlePlay}
            className="rounded-full border border-violet-400/40 bg-gradient-to-r from-violet-600 to-purple-800 px-3 py-1.5 text-xs font-bold text-white shadow-lg transition-all hover:from-violet-500 hover:to-purple-700"
          >
            {singleAction
              ? `Play ${CardDisplayName[hand[selectedIndices[0]]]}`
              : `Play ${selectedIndices.length}-Card Combo`}
          </button>
        )}

        {/* Invalid selection hint */}
        {selectedIndices.length > 0 && !singleAction && !validCombo && (
          <span className="max-w-[8rem] self-center text-center text-[10px] font-semibold text-purple-700">
            Select 2-3 matching cards, or 5 unique cards for a combo
          </span>
        )}
      </div>

      {/* Target-selection popup for targeted moves */}
      {showTargetModal && (
        <TargetModal
          title={modalTitle}
          prompt={modalPrompt}
          players={players}
          currentPlayerId={playerId}
          needsCard={needs3CardPick}
          onCommit={commitTargetedMove}
          onCancel={() => setShowTargetModal(false)}
        />
      )}
    </div>
  );
}
