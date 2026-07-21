package main

import (
	"context"
	"log"
	"math/rand"
	"net/url"
	"slices"
	"time"

	"github.com/gorilla/websocket"

	"game/wire"
)

type bot struct {
	target     string
	lobby      string
	userId     string
	wantSeats  int // start the game once this many players are seated
	think      time.Duration
	playChance float64 // odds of playing a card instead of just drawing, on your turn
	nopeChance float64 // odds of noping a pending action when holding a Nope
	restart    bool    // restart the lobby after game over to sustain load
	verbose    bool

	seq int
}

// backoff bounds when reconecting
const (
	minBackoff = 250 * time.Millisecond
	maxBackoff = 5 * time.Second
)

func (b *bot) run(ctx context.Context) {
	q := url.Values{}
	q.Set("lobby", b.lobby)
	q.Set("username", b.userId)
	q.Set("userId", b.userId)
	q.Set("avatar", "")
	q.Set("create", "1")
	wsURL := b.target + "/api/ws?" + q.Encode()

	backoff := minBackoff
	for ctx.Err() == nil {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			log.Printf("%s: dial failed: %v (retry in %s)", b.userId, err, backoff)
			if sleepCtx(ctx, jitter(backoff)) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		// close the socket when the run is cancelled; this unblocks the pending
		// ReadJSON in session so the bot exits promptly instead of hanging on it.
		stopWatch := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				conn.Close()
			case <-stopWatch:
			}
		}()

		// a session that reads at least once counts as a real connection, so
		// reset backoff; a session that dies immediately keeps escalating it.
		real := b.session(conn)
		close(stopWatch)

		if ctx.Err() != nil {
			return
		}
		if real {
			backoff = minBackoff
		}
		log.Printf("%s: disconnected, reconnecting in %s", b.userId, backoff)
		if sleepCtx(ctx, jitter(backoff)) {
			return
		}
		backoff = min(backoff*2, maxBackoff)
	}
}

// sleepCtx waits for d, or returns early if ctx is cancelled first. Reports
// true when it was cut short by cancellation, so callers can bail out.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return true
	case <-t.C:
		return false
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

		if st.Err != "" && b.verbose {
			// a legal-move bug or a race — surface it, don't silently ignore
			log.Printf("%s: server rejected action: %s", b.userId, st.Err)
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

// makes a move. return nil to avoid making a move. Every action it emits is a
// legal move for the given state, so the server never has to reject it.
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
			return b.playTurn(st)
		}

	case wire.TurnSeeingTheFuture:
		if myTurn {
			return &wire.ActionRequest{ActionStr: wire.ActionDrawCard}
		}

	case wire.TurnAlteringTheFuture:
		if myTurn {
			return &wire.ActionRequest{
				ActionStr:        wire.ActionAlterFuture,
				AlterFutureOrder: rand.Perm(min(3, st.DeckSize)),
			}
		}

	case wire.TurnAwaitingKittenPlacement:
		if myTurn {
			return &wire.ActionRequest{
				ActionStr:        wire.ActionPlaceKitten,
				PlaceKittenIndex: rand.Intn(st.DeckSize + 1),
			}
		}

	case wire.TurnAwaitingFavor:
		if st.TargetedPlayer == st.PlayerId && len(st.Hand) > 0 {
			return &wire.ActionRequest{
				ActionStr:    wire.ActionGiveFavor,
				UseCardIndex: rand.Intn(len(st.Hand)),
			}
		}

	case wire.TurnAwaitingDiscardTake:
		if myTurn && len(st.DiscardOptions) > 0 {
			return &wire.ActionRequest{
				ActionStr:        wire.ActionTakeFromDiscard,
				RequestedCardStr: st.DiscardOptions[rand.Intn(len(st.DiscardOptions))],
			}
		}

	case wire.TurnAcceptingNopes:
		return b.maybeNope(st)

	case wire.TurnGameOver:
		if b.restart && st.PlayerId == 0 {
			return &wire.ActionRequest{ActionStr: wire.ActionRestartLobby}
		}
	}

	return nil
}

// playTurn decides a Normal-state turn: sometimes play a card, otherwise draw.
func (b *bot) playTurn(st *wire.GameState) *wire.ActionRequest {
	if rand.Float64() < b.playChance {
		if act := b.tryPlay(st); act != nil {
			return act
		}
	}
	return &wire.ActionRequest{ActionStr: wire.ActionDrawCard}
}

// tryPlay enumerates the legal card plays for the current hand and picks one at
// random, or returns nil if the hand affords none. (5-combos are omitted: they
// need a non-empty discard pile, whose size the Normal-state snapshot doesn't
// expose, so we can't guarantee legality.)
func (b *bot) tryPlay(st *wire.GameState) *wire.ActionRequest {
	var cands []*wire.ActionRequest

	targets := aliveOthers(st, false)
	withCards := aliveOthers(st, true)

	for i, c := range st.Hand {
		switch c {
		case wire.CardSkip, wire.CardAttack, wire.CardSeeTheFuture,
			wire.CardAlterTheFuture, wire.CardShuffle, wire.CardDrawFromBottom:
			cands = append(cands, &wire.ActionRequest{ActionStr: wire.ActionPlayCard, UseCardIndex: i})
		case wire.CardTargetedAttack:
			if len(targets) > 0 {
				cands = append(cands, &wire.ActionRequest{
					ActionStr: wire.ActionPlayCard, UseCardIndex: i, TargetedPlayer: pick(targets),
				})
			}
		case wire.CardFavor:
			if len(withCards) > 0 {
				cands = append(cands, &wire.ActionRequest{
					ActionStr: wire.ActionPlayCard, UseCardIndex: i, TargetedPlayer: pick(withCards),
				})
			}
		}
	}

	cands = append(cands, comboCandidates(st, withCards)...)

	if len(cands) == 0 {
		return nil
	}
	return cands[rand.Intn(len(cands))]
}

// comboCandidates builds legal 2- and 3-combos: any card held 2+ (or 3+) times
// is a matching combo, and it needs a live target holding cards to steal from.
func comboCandidates(st *wire.GameState, withCards []int) []*wire.ActionRequest {
	if len(withCards) == 0 {
		return nil
	}

	byCard := map[string][]int{}
	for i, c := range st.Hand {
		byCard[c] = append(byCard[c], i)
	}

	var cands []*wire.ActionRequest
	for _, idxs := range byCard {
		if len(idxs) >= 2 {
			cands = append(cands, &wire.ActionRequest{
				ActionStr:      wire.ActionCombo,
				ComboIndices:   []int{idxs[0], idxs[1]},
				TargetedPlayer: pick(withCards),
			})
		}
		if len(idxs) >= 3 {
			cands = append(cands, &wire.ActionRequest{
				ActionStr:        wire.ActionCombo,
				ComboIndices:     []int{idxs[0], idxs[1], idxs[2]},
				TargetedPlayer:   pick(withCards),
				RequestedCardStr: wire.CardDefuse,
			})
		}
	}
	return cands
}

// maybeNope flips a pending action's nope state, at nopeChance odds, if we are
// alive and holding a Nope.
func (b *bot) maybeNope(st *wire.GameState) *wire.ActionRequest {
	if rand.Float64() >= b.nopeChance || !meAlive(st) || !slices.Contains(st.Hand, wire.CardNope) {
		return nil
	}
	return &wire.ActionRequest{ActionStr: wire.ActionPlayNope, WantNoped: !st.IsNoped}
}

// aliveOthers returns the ids of living players other than us; withCards limits
// it to those still holding at least one card.
func aliveOthers(st *wire.GameState, withCards bool) []int {
	var ids []int
	for _, p := range st.Players {
		if p.Id == st.PlayerId || !p.IsAlive {
			continue
		}
		if withCards && p.CardCount == 0 {
			continue
		}
		ids = append(ids, p.Id)
	}
	return ids
}

func meAlive(st *wire.GameState) bool {
	for _, p := range st.Players {
		if p.Id == st.PlayerId {
			return p.IsAlive
		}
	}
	return false
}

func pick(ids []int) int {
	return ids[rand.Intn(len(ids))]
}
