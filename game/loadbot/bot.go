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

// backoff bounds when reconecting
const (
	minBackoff = 250 * time.Millisecond
	maxBackoff = 5 * time.Second
)

func (b *bot) run() {
	q := url.Values{}
	q.Set("lobby", b.lobby)
	q.Set("username", b.userId)
	q.Set("userId", b.userId)
	q.Set("avatar", "")
	q.Set("create", "1")
	wsURL := b.target + "/api/ws?" + q.Encode()

	backoff := minBackoff
	for {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			log.Printf("%s: dial failed: %v (retry in %s)", b.userId, err, backoff)
			time.Sleep(jitter(backoff))
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		// a session that reads at least once counts as a real connection, so
		// reset backoff; a session that dies immediately keeps escalating it.
		if b.session(conn) {
			backoff = minBackoff
		}
		log.Printf("%s: disconnected, reconnecting in %s", b.userId, backoff)
		time.Sleep(jitter(backoff))
		backoff = min(backoff*2, maxBackoff)
	}
}

// session drives one connection until it errors. returns true if the
// connection ever delivered a state (i.e. it was a real, working session).
func (b *bot) session(conn *websocket.Conn) bool {
	defer conn.Close()

	gotState := false
	for {
		var st wire.GameState
		if err := conn.ReadJSON(&st); err != nil {
			if b.verbose {
				log.Printf("%s: read ended: %v", b.userId, err)
			}
			return gotState
		}
		gotState = true

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
			if b.verbose {
				log.Printf("%s: write failed: %v", b.userId, err)
			}
			return gotState
		}
		if b.verbose {
			log.Printf("%s (lobby %s, seat %d): %s seq=%d", b.userId, b.lobby, st.PlayerId, act.ActionStr, b.seq)
		}
	}
}

// jitter spreads reconnect storms so a killed node's bots don't redial in
// lockstep. returns d scaled by a random factor in [0.5, 1.0).
func jitter(d time.Duration) time.Duration {
	return d/2 + time.Duration(rand.Int63n(int64(d/2)))
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
