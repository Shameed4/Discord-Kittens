package main

import (
	"errors"
	"fmt"
	"slices"
)

func (lobby *Lobby) resolveDrawnCard(player *Player, drawn Card, actionDesc string) {
	if drawn == ExplodingKitten {
		if defuseIndex := slices.Index(player.Hand, Defuse); defuseIndex != -1 {
			player.Hand = slices.Delete(player.Hand, defuseIndex, defuseIndex+1)
			lobby.turnState = AwaitingKittenPlacement
			lobby.lastAction = LastAction{Public: fmt.Sprintf("Player %d %s and defused it!", player.Id, actionDesc)}
		} else {
			player.IsAlive = false
			lobby.lastAction = LastAction{Public: fmt.Sprintf("Player %d %s and exploded!", player.Id, actionDesc)}
		}
	} else {
		player.Hand = append(player.Hand, drawn)
		lobby.setNextPlayerTurn(false)
		lobby.lastAction = LastAction{Public: fmt.Sprintf("Player %d %s", player.Id, actionDesc)}
	}
}

// --- Turn Management ---

func (lobby *Lobby) decreaseTurns() {
	lobby.turnsToTake -= 1
	if lobby.turnsToTake == 0 {
		lobby.setNextPlayerTurn(false)
	}
}

func (lobby *Lobby) setNextPlayerTurn(attack bool) {
	if lobby.livingPlayers == 0 {
		return // Safety valve to prevent infinite loops
	}
	idx := lobby.currentPlayerIndex
	for {
		idx = (idx + 1) % len(lobby.players)
		if lobby.players[idx].IsAlive {
			lobby.setPlayerTurn(attack, idx)
			return
		}
	}
}

func (lobby *Lobby) setPlayerTurn(attack bool, playerIdx int) {
	lobby.currentPlayerIndex = playerIdx

	if !attack {
		lobby.turnsToTake = 1
		lobby.underAttack = false
		return
	}

	if lobby.underAttack {
		lobby.turnsToTake += 2
	} else {
		lobby.turnsToTake = 2
	}
	lobby.underAttack = true
}

// --- Assertions ---

func (lobby *Lobby) assertTurnAndState(validStates []TurnState, isPlayerTurn bool, action string) error {
	if !slices.Contains(validStates, lobby.turnState) {
		return errors.New("Cannot " + action + " - Invalid state")
	}
	if !isPlayerTurn {
		return errors.New("Cannot " + action + " - Not your turn")
	}
	return nil
}

func (lobby *Lobby) assertPlayerExistsAndAlive(playerId int) error {
	if playerId < 0 || playerId >= len(lobby.players) {
		return errors.New("Player does not exist")
	} else if !lobby.players[playerId].IsAlive {
		return errors.New("Player must be alive")
	}
	return nil
}

// --- State Broadcasting ---

func (lobby *Lobby) lastActionFor(playerIdx int) string {
	if msg, ok := lobby.lastAction.Private[playerIdx]; ok {
		return msg
	}
	return lobby.lastAction.Public
}

func (lobby *Lobby) getGameState(playerIdx int) GameState {
	player := lobby.players[playerIdx]
	hand := cardSliceToStrings(player.Hand)
	var future []string = nil
	var isPlayerTurn = lobby.currentPlayerIndex == playerIdx
	if (lobby.turnState == SeeingTheFuture || lobby.turnState == AlteringTheFuture) && isPlayerTurn {
		count := min(3, len(lobby.deck))
		future = cardSliceToStrings(lobby.deck[:count])
	}
	var discardOptions []string
	if lobby.turnState == AwaitingDiscardTake && isPlayerTurn {
		discardMap := make(map[Card]bool)
		discardOptions = []string{}
		for _, c := range lobby.discardPile {
			if !discardMap[c] {
				discardMap[c] = true
				discardOptions = append(discardOptions, c.String())
			}
		}
	}
	res := GameState{
		PlayerId:    playerIdx,
		TurnId:      lobby.currentPlayerIndex,
		DeckSize:    len(lobby.deck),
		TurnState:   lobby.turnState.String(),
		Hand:        hand,
		InProgress:  lobby.inProgress(),
		UnderAttack: lobby.underAttack,
		TurnsToTake: lobby.turnsToTake,

		Future:         future,
		TargetedPlayer: lobby.targetedPlayer,
		DiscardOptions: discardOptions,
		LastAction:     lobby.lastActionFor(playerIdx),
	}
	for _, player := range lobby.players {
		res.Players = append(res.Players, PlayerGameState{
			Id:        player.Id,
			CardCount: len(player.Hand),
			IsAlive:   player.IsAlive,
			IsOnline:  player.IsOnline,
		})
	}
	return res
}

func (lobby *Lobby) sendError(playerIdx int, err string) {
	player := lobby.players[playerIdx]
	if !player.IsOnline {
		return
	}
	res := lobby.getGameState(playerIdx)
	res.Err = err
	player.Send <- res
}

func (lobby *Lobby) broadcastGameState() {
	for playerIdx, player := range lobby.players {
		if player.IsOnline {
			player.Send <- lobby.getGameState(playerIdx)
		}
	}
}
