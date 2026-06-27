package main

import (
	"errors"
	"fmt"
	"slices"
)

const maxLogEntries = 200

// recordAction sets the most-recent action (for the banner) and appends it to
// the append-only game log, trimming the log to the last maxLogEntries entries.
func (lobby *Lobby) recordAction(a LastAction) {
	lobby.lastAction = a
	lobby.actionLog = append(lobby.actionLog, a)
	if len(lobby.actionLog) > maxLogEntries {
		lobby.actionLog = lobby.actionLog[len(lobby.actionLog)-maxLogEntries:]
	}
}

func (lobby *Lobby) resolveDrawnCard(player *Player, drawn Card, actionDesc string) {
	if drawn == ExplodingKitten {
		if defuseIndex := slices.Index(player.Hand, Defuse); defuseIndex != -1 {
			lobby.discardCard(player, defuseIndex)
			lobby.turnState = AwaitingKittenPlacement
			lobby.recordAction(LastAction{Public: fmt.Sprintf("%s %s and had to defuse an exploding kitten!", player.Name, actionDesc)})
		} else {
			lobby.recordAction(LastAction{Public: fmt.Sprintf("%s %s and exploded!", player.Name, actionDesc)})
			lobby.eliminatePlayer(player.Id)
		}
	} else {
		player.Hand = append(player.Hand, drawn)
		lobby.decreaseTurns()
		lobby.recordAction(LastAction{Public: fmt.Sprintf("%s %s", player.Name, actionDesc)})
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
	pos := 0
	for i, p := range lobby.playersList {
		if p.Id == lobby.currentPlayerIndex {
			pos = i
			break
		}
	}
	n := len(lobby.playersList)
	for offset := 1; offset <= n; offset++ {
		next := lobby.playersList[(pos+offset)%n]
		if next.IsAlive {
			lobby.setPlayerTurn(attack, next.Id)
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
	player, ok := lobby.playersMap[playerId]
	if !ok {
		return errors.New("Player does not exist")
	} else if !player.IsAlive {
		return errors.New("Player must be alive")
	}
	return nil
}

// --- State Broadcasting ---

// resolveAction picks the personalized message for a player, falling back to
// the public one.
func resolveAction(a LastAction, playerIdx int) string {
	if msg, ok := a.Private[playerIdx]; ok {
		return msg
	}
	return a.Public
}

func (lobby *Lobby) lastActionFor(playerIdx int) string {
	return resolveAction(lobby.lastAction, playerIdx)
}

func (lobby *Lobby) getGameState(playerIdx int) GameState {
	player := lobby.playersMap[playerIdx]
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
	log := make([]string, len(lobby.actionLog))
	for i, a := range lobby.actionLog {
		log[i] = resolveAction(a, playerIdx)
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
		IsNoped:        lobby.pendingAction != nil && lobby.pendingAction.isNoped,
		Log:            log,
	}
	if lobby.turnState == AcceptingNopes {
		res.NopeDeadline = lobby.nopeDeadline.UnixMilli()
	}
	for _, player := range lobby.playersList {
		res.Players = append(res.Players, PlayerGameState{
			Id:        player.Id,
			Name:      player.Name,
			Avatar:    player.Avatar,
			CardCount: len(player.Hand),
			IsAlive:   player.IsAlive,
			IsOnline:  player.IsOnline,
		})
	}
	return res
}

// tries sending to online player, without blocking. if channel is full
// (meaning client hasn't received many messages) then it must be disconnected.
func (lobby *Lobby) sendTo(playerIdx int, state GameState) {
	player := lobby.playersMap[playerIdx]
	if !player.IsOnline {
		return
	}
	select {
	case player.Send <- state:
	default:
		lobby.disconnectPlayer(playerIdx)
	}
}

func (lobby *Lobby) sendError(playerIdx int, err string) {
	player := lobby.playersMap[playerIdx]
	if !player.IsOnline {
		return
	}
	res := lobby.getGameState(playerIdx)
	res.Err = err
	lobby.sendTo(playerIdx, res)
}

func (lobby *Lobby) broadcastGameState() {
	for _, player := range lobby.playersMap {
		if player.IsOnline {
			lobby.sendTo(player.Id, lobby.getGameState(player.Id))
		}
	}
}
