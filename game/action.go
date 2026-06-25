package main

import (
	"cmp"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"slices"
	"strings"
	"time"
)

const nopeDelay = 5000 * time.Millisecond

// playerName returns the display name for a player id, falling back to a
// generic label if the id is somehow out of range.
func (lobby *Lobby) playerName(id int) string {
	if id < 0 || id >= len(lobby.players) {
		return fmt.Sprintf("Player %d", id)
	}
	return lobby.players[id].Name
}

// handles action received by player and executes it if not nopeable
func (lobby *Lobby) receivePlayerAction(action PlayerAction) error {
	playerId := action.playerId
	player := lobby.players[playerId]
	name := player.Name
	isPlayerTurn := action.playerId == lobby.currentPlayerIndex
	switch action.actionType {
	case StartGame:
		if lobby.inProgress() {
			return errors.New("Cannot start lobby - game already in progress")
		}
		if err := lobby.startGame(); err != nil {
			return err
		}
		lobby.recordAction(LastAction{Public: "Game started!"})

	case Disconnect:
		// don't disconnect new channel when client reconnects
		if action.conn != nil && player.Send != action.conn {
			return nil
		}
		if !player.IsOnline {
			fmt.Printf("Illegal state? Player id %d is offline but disconnected", playerId)
		}
		lobby.disconnectPlayer(playerId)

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
		lobby.recordAction(LastAction{Public: fmt.Sprintf("%s placed the Exploding Kitten back in the deck", name)})

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

		switch playedCard {
		case Skip, SeeTheFuture, AlterTheFuture, Attack, Shuffle, DrawFromBottom:
			lobby.recordAction(LastAction{Public: fmt.Sprintf("%s wants to %s", name, playedCard.CardName())})
		case TargetedAttack:
			lobby.recordAction(LastAction{Public: fmt.Sprintf("%s wants to target %s", name, lobby.playerName(action.targetedPlayer))})
		case Favor:
			lobby.recordAction(LastAction{Public: fmt.Sprintf("%s wants to ask %s for a favor", name, lobby.playerName(action.targetedPlayer))})
		default:
			return errors.New("Cannot play that card")
		}
		lobby.discardCard(player, action.useCardIndex)
		lobby.turnState = AcceptingNopes
		lobby.startNopeTimer()
		lobby.pendingAction = &PendingNopeableAction{
			playerId:       playerId,
			actionType:     PlayCard,
			playedCard:     playedCard,
			targetedPlayer: action.targetedPlayer,
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
		lobby.recordAction(LastAction{Public: fmt.Sprintf("%s altered the future", name)})

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
		lobby.recordAction(LastAction{
			Public:  fmt.Sprintf("%s gave a card to %s", name, lobby.playerName(lobby.currentPlayerIndex)),
			Private: map[int]string{lobby.currentPlayerIndex: fmt.Sprintf("%s gave you %s", name, transferredCard)},
		})

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
		joinedComboCards := strings.Join(cardSliceToStrings(comboCards), "+")
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
			lobby.pendingAction = &PendingNopeableAction{
				playerId:       playerId,
				actionType:     Combo,
				comboSize:      comboSize,
				targetedPlayer: action.targetedPlayer,
				requestedCard:  action.requestedCard,
			}
			lobby.turnState = AcceptingNopes
			lobby.startNopeTimer()
			lobby.recordAction(LastAction{Public: fmt.Sprintf("%s wants to play a %d-combo (%s) on %s", name, comboSize, joinedComboCards, lobby.playerName(action.targetedPlayer))})

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
			lobby.recordAction(LastAction{Public: fmt.Sprintf("%s started a 5-combo (%s) and can pick a card from the discard pile", name, joinedComboCards)})
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
			lobby.recordAction(LastAction{Public: fmt.Sprintf("%s took %s from the discard pile", name, taken)})
		} else {
			return errors.New("Card is not in discard pile!")
		}
		lobby.turnState = Normal
		lobby.decreaseTurns()

	case PlayNope:
		if !player.IsAlive {
			return errors.New("Must be alive to nope")
		}
		if lobby.pendingAction == nil {
			return errors.New("No action to nope")
		}
		nopeIdx := slices.Index(player.Hand, Nope)
		if nopeIdx == -1 {
			return errors.New("You do not have a nope")
		}
		if lobby.pendingAction.isNoped && action.wantNoped {
			return errors.New("Card has already been noped")
		}
		if !lobby.pendingAction.isNoped && !action.wantNoped {
			return errors.New("Card has already been yuped")
		}
		var nopedYupedStr string
		if action.wantNoped {
			nopedYupedStr = "noped"
		} else {
			nopedYupedStr = "yuped"
		}
		var originalMoveName string
		if lobby.pendingAction.actionType == PlayCard {
			originalMoveName = lobby.pendingAction.playedCard.CardName()
		} else {
			originalMoveName = fmt.Sprintf("%d-combo", lobby.pendingAction.comboSize)
		}
		originalPlayerName := lobby.playerName(lobby.pendingAction.playerId)
		lobby.recordAction(LastAction{Public: fmt.Sprintf("%s %s %s's %s", name, nopedYupedStr, originalPlayerName, originalMoveName)})
		lobby.discardCard(player, nopeIdx)
		lobby.pendingAction.isNoped = !lobby.pendingAction.isNoped
		lobby.startNopeTimer()
	}
	return nil
}

// startNopeTimer (re)starts the nope window, keeping the timer and the
// broadcastable deadline in sync. Safe to call whether or not a timer exists.
func (lobby *Lobby) startNopeTimer() {
	lobby.nopeDeadline = time.Now().Add(nopeDelay)
	if lobby.nopeTimer == nil {
		lobby.nopeTimer = time.NewTimer(nopeDelay)
	} else {
		lobby.nopeTimer.Reset(nopeDelay)
	}
}

func (lobby *Lobby) handleNopeTimerComplete() {
	if lobby.pendingAction == nil {
		log.Println("Nope timer completed but no pending action")
		return
	}
	if lobby.pendingAction.isNoped {
		lobby.denyPlayerAction()
	} else {
		lobby.confirmPlayerAction()
	}
	lobby.pendingAction = nil
	lobby.nopeTimer = nil
	lobby.broadcastGameState()
}

// when a nopeable action was not noped
func (lobby *Lobby) confirmPlayerAction() {
	action := lobby.pendingAction
	playerId := action.playerId
	player := lobby.players[playerId]
	name := lobby.playerName(playerId)
	switch action.actionType {
	case PlayCard:
		switch action.playedCard {
		case Skip:
			lobby.turnState = Normal
			lobby.decreaseTurns()
			lobby.recordAction(LastAction{Public: fmt.Sprintf("%s successfully skipped", name)})
		case SeeTheFuture:
			lobby.turnState = SeeingTheFuture
			lobby.recordAction(LastAction{Public: fmt.Sprintf("%s is seeing the future...", name)})
		case AlterTheFuture:
			lobby.turnState = AlteringTheFuture
			lobby.recordAction(LastAction{Public: fmt.Sprintf("%s is altering the future...", name)})
		case Attack:
			lobby.turnState = Normal
			lobby.setNextPlayerTurn(true)
			lobby.recordAction(LastAction{Public: fmt.Sprintf("%s attacked!", name)})
		case TargetedAttack:
			lobby.turnState = Normal
			lobby.setPlayerTurn(true, action.targetedPlayer)
			lobby.recordAction(LastAction{Public: fmt.Sprintf("%s targeted %s!", name, lobby.playerName(action.targetedPlayer))})
		case Shuffle:
			lobby.turnState = Normal
			lobby.shuffleDeck()
			lobby.recordAction(LastAction{Public: fmt.Sprintf("%s shuffled the deck", name)})
		case DrawFromBottom:
			lobby.turnState = Normal
			drawn := lobby.removeBottomCard()
			lobby.resolveDrawnCard(player, drawn, "drew from the bottom")
		case Favor:
			targetedPlayer := lobby.players[action.targetedPlayer]
			if len(targetedPlayer.Hand) > 0 {
				lobby.targetedPlayer = action.targetedPlayer
				lobby.turnState = AwaitingFavor
				lobby.recordAction(LastAction{Public: fmt.Sprintf("%s is asking %s for a favor", name, lobby.playerName(action.targetedPlayer))})
			} else {
				lobby.turnState = Normal
				lobby.recordAction(LastAction{Public: fmt.Sprintf("%s ran out of cards and could not give %s a favor", name, lobby.playerName(action.targetedPlayer))})
			}
		default:
			log.Printf("Unknown card %s was received when confirming player action", action.playedCard.String())
		}
	case Combo:
		lobby.turnState = Normal
		comboSize := action.comboSize
		deleteIndex := -1
		targetedPlayer := lobby.players[action.targetedPlayer]
		if len(targetedPlayer.Hand) > 0 {
			switch comboSize {
			case 2:
				deleteIndex = rand.Intn(len(targetedPlayer.Hand))
			case 3:
				deleteIndex = slices.Index(targetedPlayer.Hand, action.requestedCard)
			}
		}
		if deleteIndex != -1 {
			stolen := targetedPlayer.Hand[deleteIndex]
			player.Hand = append(player.Hand, stolen)
			targetedPlayer.Hand = slices.Delete(targetedPlayer.Hand, deleteIndex, deleteIndex+1)

			targetName := lobby.playerName(action.targetedPlayer)
			var publicAction string
			if comboSize == 2 {
				publicAction = fmt.Sprintf("%s successfully stole from %s using a 2-combo", name, targetName)
			} else {
				publicAction = fmt.Sprintf("%s successfully stole %s from %s using a 3-combo", name, stolen, targetName)
			}

			lobby.recordAction(LastAction{
				Public: publicAction,
				Private: map[int]string{
					playerId:              fmt.Sprintf("You stole %s from %s using a %d-combo", stolen, targetName, comboSize),
					action.targetedPlayer: fmt.Sprintf("%s stole your %s using a %d-combo", name, stolen, comboSize),
				},
			})
		} else {
			lobby.recordAction(LastAction{Public: fmt.Sprintf("%s played a %d-combo on %s but got nothing", name, comboSize, lobby.playerName(action.targetedPlayer))})
		}
	default:
		log.Printf("Unknown action type %d confirmed ", action.actionType)
	}
}

// when a nopeable action was noped
func (lobby *Lobby) denyPlayerAction() {
	action := lobby.pendingAction
	playerId := action.playerId
	name := lobby.playerName(playerId)
	lobby.turnState = Normal
	switch action.actionType {
	case PlayCard:
		lobby.recordAction(LastAction{Public: fmt.Sprintf("%s's %s failed due to being noped", name, action.playedCard.CardName())})
	case Combo:
		lobby.recordAction(LastAction{Public: fmt.Sprintf("%s's %d-combo failed due to being noped", name, action.comboSize)})
	default:
		log.Printf("Unknown action type %d denied from nope ", action.actionType)
	}
}
