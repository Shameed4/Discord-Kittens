package main

import (
	"log"
	"math/rand"
	"net/url"
	"time"

	"github.com/gorilla/websocket"

	"game/wire"
)

type bot struct {
	target    string
	lobby     string
	userId    string
	wantSeats int // start the game once this many players are seated
	think     time.Duration
	verbose   bool

	seq int
}

func (b *bot) run() {
	q := url.Values{}
	q.Set("lobby", b.lobby)
	q.Set("username", b.userId)
	q.Set("userId", b.userId)
	q.Set("avatar", "")
	q.Set("create", "1")
	wsURL := b.target + "/api/ws?" + q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Printf("%s: dial failed: %v", b.userId, err)
		return
	}
	defer conn.Close()

	for {
		var st wire.GameState
		if err := conn.ReadJSON(&st); err != nil {
			log.Printf("%s: read ended: %v", b.userId, err)
			return
		}

		act := b.decide(&st)
		if act == nil {
			continue
		}

		if b.think > 0 {
			time.Sleep(b.think)
		}
		b.seq++
		act.SeqNumber = b.seq
		if err := conn.WriteJSON(act); err != nil {
			log.Printf("%s: write failed: %v", b.userId, err)
			return
		}
		if b.verbose {
			log.Printf("%s (lobby %s, seat %d): %s seq=%d", b.userId, b.lobby, st.PlayerId, act.ActionStr, b.seq)
		}
	}
}

// makes a move. return nil to avoid making a move.
func (b *bot) decide(st *wire.GameState) *wire.ActionRequest {
	myTurn := st.TurnId == st.PlayerId

	switch st.TurnState {
	case wire.TurnNotStarted:
		// starts game once required players have joined
		if st.PlayerId == 0 && len(st.Players) >= max(2, b.wantSeats) {
			return &wire.ActionRequest{ActionStr: wire.ActionStartGame}
		}

	case wire.TurnNormal:
		if myTurn {
			return &wire.ActionRequest{ActionStr: wire.ActionDrawCard}
		}

	case wire.TurnAwaitingKittenPlacement:
		if myTurn {
			return &wire.ActionRequest{
				ActionStr:        wire.ActionPlaceKitten,
				PlaceKittenIndex: rand.Intn(st.DeckSize + 1),
			}
		}
	}

	return nil
}
