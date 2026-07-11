package main

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

// constructs a lobby in the middle of a nope (hardest serialization/deserialization case)
// intentionally avoids player ids being the same as their indices to catch bugs early
func buildMidNopeLobby() *Lobby {
	lobby := &Lobby{
		name:            "round-trip",
		deck:            []Card{Skip, Attack, ExplodingKitten, Defuse},
		discardPile:     []Card{Favor, Nope},
		nextId:          10,
		currentPlayerId: 9,
		turnState:       AcceptingNopes,
		livingPlayers:   2,
		turnsToTake:     1,
		underAttack:     true,
		targetedPlayer:  5,
		nopeDeadline:    time.Now().Add(5 * time.Second),
		playersMap:      map[int]*Player{},
		spectators:      map[int]*Spectator{},
		lastAction:      CompletedAction{Public: "Player 9 played Favor", Private: map[int]string{5: "Player 9 stole your Defuse"}},
		actionLog: []CompletedAction{
			{Public: "Player 5 drew a card"},
			{Public: "Player 9 played Favor", Private: map[int]string{5: "Player 9 stole your Defuse"}},
		},
		pendingAction: &PendingNopeableAction{
			playerId:       9,
			actionType:     PlayCard,
			playedCard:     Favor,
			targetedPlayer: 5,
			isNoped:        true,
		},
	}
	p5 := &Player{Id: 5, DiscordUserId: "u5", Name: "Alice", Hand: []Card{Defuse, Skip}, IsAlive: true, IsOnline: true}
	p9 := &Player{Id: 9, DiscordUserId: "u9", Name: "Bob", Hand: []Card{Attack, Nope}, IsAlive: true, IsOnline: true}
	lobby.playersList = []*Player{p5, p9}
	lobby.playersMap[5] = p5
	lobby.playersMap[9] = p9
	return lobby
}

// test if serializing, deserializing, and serializing results in the same lobby
func TestSnapshotRoundTrip(t *testing.T) {
	orig := buildMidNopeLobby()

	b1, err := json.Marshal(orig.SerializeLobby())
	if err != nil {
		t.Fatalf("marshal 1: %v", err)
	}

	var snap LobbySnapshot
	if err := json.Unmarshal(b1, &snap); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	restored := DeserializeLobby(&snap)

	b2, err := json.Marshal(restored.SerializeLobby())
	if err != nil {
		t.Fatalf("marshal 2: %v", err)
	}

	if !bytes.Equal(b1, b2) {
		t.Fatalf("round-trip not stable:\n first: %s\nsecond: %s", b1, b2)
	}
}

// makes sure that necessary fields are actually being serialized
func TestRestorePreservesGameState(t *testing.T) {
	b, _ := json.Marshal(buildMidNopeLobby().SerializeLobby())
	var snap LobbySnapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	restored := DeserializeLobby(&snap)

	for _, id := range []int{5, 9} {
		p, ok := restored.playersMap[id]
		if !ok {
			t.Fatalf("playersMap missing id %d (keyed by slice index?)", id)
		}
		if p.Id != id {
			t.Fatalf("playersMap[%d] resolved to player id %d", id, p.Id)
		}
	}

	for _, p := range restored.playersList {
		if p.IsOnline {
			t.Fatalf("player %d restored IsOnline=true, want false", p.Id)
		}
	}

	if !restored.playersMap[5].IsAlive {
		t.Fatalf("IsAlive not preserved")
	}

	if restored.pendingAction == nil {
		t.Fatalf("pendingAction lost on restore")
	}
	if restored.pendingAction.playerId != 9 || restored.pendingAction.playedCard != Favor || !restored.pendingAction.isNoped {
		t.Fatalf("pendingAction fields not preserved: %+v", restored.pendingAction)
	}

	if restored.nopeTimer == nil {
		t.Fatalf("nopeTimer not re-armed; nope window would never fire")
	}

	if restored.lastAction.Public == "" || restored.lastAction.Private[5] == "" {
		t.Fatalf("lastAction not preserved: %+v", restored.lastAction)
	}
}
