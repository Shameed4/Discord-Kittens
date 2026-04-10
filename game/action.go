package main

import (
	"cmp"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"strings"
)

func (lobby *Lobby) takePlayerAction(action PlayerAction) error {
	playerId := action.playerId
	player := lobby.players[playerId]
	isPlayerTurn := action.playerId == lobby.currentPlayerIndex
	switch action.actionType {
	case StartGame:
		if lobby.inProgress() {
			return errors.New("Cannot start lobby - game already in progress")
		}
		if err := lobby.startGame(); err != nil {
			return err
		}
		lobby.lastAction = LastAction{Public: "Game started!"}

	case Disconnect:
		if !player.IsOnline {
			fmt.Printf("Illegal state? Player id %d is offline but disconnected", playerId)
		}
		lobby.disconnectPlayer(playerId)
		lobby.lastAction = LastAction{Public: fmt.Sprintf("Player %d disconnected", playerId)}

	case DrawCard:
		if err := lobby.assertTurnAndState([]TurnState{Normal, SeeingTheFuture}, isPlayerTurn, "draw card"); err != nil {
			return err
		}

		lobby.turnState = Normal // clear effects like seeing the future
		drawn := lobby.removeTopCard()
		lobby.resolveDrawnCard(player, drawn, "drew a card")

	case PlaceKitten:
		if err := lobby.assertTurnAndState([]TurnState{AwaitingKittenPlacement}, isPlayerTurn, "place kitten"); err != nil {
			return err
		}
		newKittenPosition := action.placeKittenIndex
		if newKittenPosition < 0 || newKittenPosition > len(lobby.deck) {
			return errors.New("Invalid kitten position")
		}
		lobby.deck = slices.Insert(lobby.deck, newKittenPosition, ExplodingKitten)
		lobby.turnState = Normal
		lobby.setNextPlayerTurn(false)
		lobby.lastAction = LastAction{Public: fmt.Sprintf("Player %d placed the Exploding Kitten back in the deck", playerId)}

	case PlayCard:
		if err := lobby.assertTurnAndState([]TurnState{Normal, SeeingTheFuture}, isPlayerTurn, "play card"); err != nil {
			return err
		}
		if action.useCardIndex < 0 || action.useCardIndex >= len(player.Hand) {
			return errors.New("Cannot play card - No card found at that index")
		}

		playedCard := player.Hand[action.useCardIndex]

		// validate before we take the card away
		if playedCard == TargetedAttack || playedCard == Favor {
			if err := lobby.assertPlayerExistsAndAlive(action.targetedPlayer); err != nil {
				return err
			}
		}
		if playedCard == Favor && len(lobby.players[action.targetedPlayer].Hand) == 0 {
			return errors.New("Cannot ask a favor from a player without cards!")
		}

		lobby.discardCard(player, action.useCardIndex)

		switch playedCard {
		case Skip:
			lobby.turnState = Normal
			lobby.decreaseTurns()
			lobby.lastAction = LastAction{Public: fmt.Sprintf("Player %d played Skip", playerId)}
		case SeeTheFuture:
			lobby.turnState = SeeingTheFuture
			lobby.lastAction = LastAction{Public: fmt.Sprintf("Player %d is seeing the future...", playerId)}
		case AlterTheFuture:
			lobby.turnState = AlteringTheFuture
			lobby.lastAction = LastAction{Public: fmt.Sprintf("Player %d is altering the future...", playerId)}
		case Attack:
			lobby.turnState = Normal
			lobby.setNextPlayerTurn(true)
			lobby.lastAction = LastAction{Public: fmt.Sprintf("Player %d attacked!", playerId)}
		case TargetedAttack:
			lobby.turnState = Normal
			lobby.setPlayerTurn(true, action.targetedPlayer)
			lobby.lastAction = LastAction{Public: fmt.Sprintf("Player %d targeted Player %d!", playerId, action.targetedPlayer)}
		case Shuffle:
			lobby.turnState = Normal
			lobby.shuffleDeck()
			lobby.lastAction = LastAction{Public: fmt.Sprintf("Player %d shuffled the deck", playerId)}
		case DrawFromBottom:
			lobby.turnState = Normal
			drawn := lobby.removeBottomCard()
			lobby.resolveDrawnCard(player, drawn, "drew from the bottom")
		case Favor:
			lobby.turnState = AwaitingFavor
			lobby.targetedPlayer = action.targetedPlayer
			lobby.lastAction = LastAction{Public: fmt.Sprintf("Player %d asked Player %d for a favor", playerId, action.targetedPlayer)}
		default:
			return errors.New("Cannot play that card")
		}

	case AlterFuture:
		if err := lobby.assertTurnAndState([]TurnState{AlteringTheFuture}, isPlayerTurn, "alter future"); err != nil {
			return err
		}
		alterSize := min(3, len(lobby.deck))
		if len(action.alterFutureOrder) != alterSize {
			return fmt.Errorf("Expected indices to have length %d, got %d", alterSize, len(action.alterFutureOrder))
		}
		for i := range action.alterFutureOrder {
			if !slices.Contains(action.alterFutureOrder, i) {
				return fmt.Errorf("Invalid indices list - missing value %d", i)
			}
		}
		buffer := make([]Card, alterSize)
		for newIndex, oldIndex := range action.alterFutureOrder {
			buffer[newIndex] = lobby.deck[oldIndex]
		}
		copy(lobby.deck, buffer)
		lobby.turnState = Normal
		lobby.lastAction = LastAction{Public: fmt.Sprintf("Player %d altered the future", playerId)}

	case GiveFavor:
		if lobby.turnState != AwaitingFavor || lobby.targetedPlayer != playerId {
			return errors.New("Must be the target of a favor request")
		}
		if action.useCardIndex < 0 || action.useCardIndex >= len(player.Hand) {
			return errors.New("Given card is out of bounds")
		}

		lobby.turnState = Normal
		transferredCard := player.Hand[action.useCardIndex]
		requester := lobby.players[lobby.currentPlayerIndex]
		player.Hand = slices.Delete(player.Hand, action.useCardIndex, action.useCardIndex+1)
		requester.Hand = append(requester.Hand, transferredCard)
		lobby.lastAction = LastAction{
			Public:  fmt.Sprintf("Player %d gave a card to Player %d", playerId, lobby.currentPlayerIndex),
			Private: map[int]string{lobby.currentPlayerIndex: fmt.Sprintf("Player %d gave you %s", playerId, transferredCard)},
		}

	case Combo:
		if err := lobby.assertTurnAndState([]TurnState{Normal, SeeingTheFuture}, isPlayerTurn, "combo"); err != nil {
			return err
		}
		if err := assertUniqueAndInBounds(action.comboIndices, len(player.Hand)); err != nil {
			return err
		}
		lobby.turnState = Normal
		comboSize := len(action.comboIndices)
		comboCards := make([]Card, len(action.comboIndices))
		for i, cardIdx := range action.comboIndices {
			comboCards[i] = player.Hand[cardIdx]
		}
		switch comboSize {
		case 2, 3:
			if err := lobby.assertPlayerExistsAndAlive(action.targetedPlayer); err != nil {
				return err
			}
			targetedPlayer := lobby.players[action.targetedPlayer]
			if len(targetedPlayer.Hand) == 0 {
				return errors.New("Cannot target a player without cards in their hand")
			}
			if err := assertValidMatchingCombo(comboCards); err != nil {
				return err
			}
			deleteIndex := -1
			switch comboSize {
			case 2:
				deleteIndex = rand.Intn(len(targetedPlayer.Hand))
			case 3:
				deleteIndex = slices.Index(targetedPlayer.Hand, action.requestedCard)
			}
			joinedComboCards := strings.Join(cardSliceToStrings(comboCards), "+")
			if deleteIndex != -1 {
				stolen := targetedPlayer.Hand[deleteIndex]
				player.Hand = append(player.Hand, stolen)
				targetedPlayer.Hand = slices.Delete(targetedPlayer.Hand, deleteIndex, deleteIndex+1)

				var publicAction string
				if comboSize == 2 {
					publicAction = fmt.Sprintf("Player %d successfully stole from Player %d using a 2-combo (%s)", playerId, action.targetedPlayer, joinedComboCards)
				} else {
					publicAction = fmt.Sprintf("Player %d successfully stole %s from Player %d using a 3-combo (%s)", playerId, stolen.String(), action.targetedPlayer, joinedComboCards)
				}

				lobby.lastAction = LastAction{
					Public: publicAction,
					Private: map[int]string{
						playerId:              fmt.Sprintf("You stole %s from Player %d using a %d-combo (%s)", stolen, action.targetedPlayer, comboSize, joinedComboCards),
						action.targetedPlayer: fmt.Sprintf("Player %d stole your %s using a %d-combo (%s)", playerId, stolen.String(), comboSize, joinedComboCards),
					},
				}
			} else {
				lobby.lastAction = LastAction{Public: fmt.Sprintf("Player %d played a %d-combo (%s) on Player %d but got nothing", playerId, len(comboCards), strings.Join(cardSliceToStrings(comboCards), "+"), action.targetedPlayer)}
			}
		case 5:
			if len(lobby.discardPile) == 0 {
				return errors.New("No cards to take from the discard pile!")
			}
			uniqueCards := make(map[Card]bool)
			for _, c := range comboCards {
				uniqueCards[c] = true
			}
			if len(uniqueCards) != 5 {
				return errors.New("All 5 cards must be unique")
			}
			lobby.turnState = AwaitingDiscardTake
			lobby.lastAction = LastAction{Public: fmt.Sprintf("Player %d played %s", playerId, strings.Join(cardSliceToStrings(comboCards), "+"))}
		default:
			return errors.New("Combos must contain 2, 3, or 5 cards")
		}
		// at this point we know it was successful, remove the cards
		slices.SortFunc(action.comboIndices, func(a, b int) int {
			return cmp.Compare(b, a)
		})
		for _, cardIdx := range action.comboIndices {
			lobby.discardCard(player, cardIdx)
		}

	case TakeFromDiscard:
		if err := lobby.assertTurnAndState([]TurnState{AwaitingDiscardTake}, isPlayerTurn, "take from discard"); err != nil {
			return err
		}
		if takeIdx := slices.Index(lobby.discardPile, action.requestedCard); takeIdx != -1 {
			taken := lobby.discardPile[takeIdx]
			player.Hand = append(player.Hand, taken)
			lobby.discardPile = slices.Delete(lobby.discardPile, takeIdx, takeIdx+1)
			lobby.lastAction = LastAction{Public: fmt.Sprintf("Player %d took %s from the discard pile", playerId, taken)}
		} else {
			return errors.New("Card is not in discard pile!")
		}
		lobby.turnState = Normal
	}
	return nil
}
