package main

import (
	"errors"
	"fmt"
	"slices"
)

type TurnState int

const (
	Normal TurnState = iota
	NotStarted
	GameOver
	AwaitingKittenPlacement
	SeeingTheFuture
	AlteringTheFuture
	AwaitingFavor
	AwaitingDiscardTake
)

func (t TurnState) String() string {
	switch t {
	case NotStarted:
		return "NOT_STARTED"
	case Normal:
		return "NORMAL"
	case GameOver:
		return "GAME_OVER"
	case AwaitingKittenPlacement:
		return "AWAITING_KITTEN_PLACEMENT"
	case SeeingTheFuture:
		return "SEEING_THE_FUTURE"
	case AlteringTheFuture:
		return "ALTERING_THE_FUTURE"
	case AwaitingFavor:
		return "AWAITING_FAVOR"
	case AwaitingDiscardTake:
		return "AWAITING_DISCARD_TAKE"
	default:
		return "UNKNOWN"
	}
}

type Player struct {
	Hand     []Card
	Id       int
	UserId   string // stable cross-session identity (e.g. Discord user id); used to reconnect returning players
	Name     string
	IsAlive  bool
	IsOnline bool

	Send chan GameState
}

type PlayerGameState struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	CardCount int    `json:"cardCount"`
	IsAlive   bool   `json:"isAlive"`
	IsOnline  bool   `json:"isOnline"`
}

type GameState struct {
	PlayerId    int               `json:"playerId"`
	TurnId      int               `json:"turnId"`
	DeckSize    int               `json:"deckSize"`
	Players     []PlayerGameState `json:"players"`
	TurnState   string            `json:"turnState"`
	Hand        []string          `json:"hand"`
	InProgress  bool              `json:"inProgress"`
	UnderAttack bool              `json:"underAttack"`
	TurnsToTake int               `json:"turnsToTake"`

	Future         []string `json:"future,omitempty"`         // for see/alter the future
	DiscardOptions []string `json:"discardOptions,omitempty"` // discard pile for 5 unique
	TargetedPlayer int      `json:"targetedPlayer"`           // for actions that require another player's response
	LastAction     string   `json:"lastAction,omitempty"`
	Log            []string `json:"log,omitempty"`
	Err            string   `json:"err,omitempty"`
}

// LastAction holds the description of the most recent game event.
// Private overrides Public for specific players (e.g., to reveal what card was stolen from them).
type LastAction struct {
	Public  string
	Private map[int]string // playerIdx -> personalized message
}

type JoinRequest struct {
	Name   string
	UserId string
	Send   chan GameState
	Result chan JoinResponse
}

type JoinResponse struct {
	success  bool
	error    string
	playerId int
}

type ActionType int

const (
	StartGame ActionType = iota
	PlayCard
	DrawCard
	PlaceKitten
	Disconnect
	AlterFuture
	GiveFavor
	Combo
	TakeFromDiscard
)

var actionTypeNames = map[string]ActionType{
	"START_GAME":        StartGame,
	"PLAY_CARD":         PlayCard,
	"DRAW_CARD":         DrawCard,
	"PLACE_KITTEN":      PlaceKitten,
	"DISCONNECT":        Disconnect,
	"ALTER_FUTURE":      AlterFuture,
	"GIVE_FAVOR":        GiveFavor,
	"COMBO":             Combo,
	"TAKE_FROM_DISCARD": TakeFromDiscard,
}

type PlayerAction struct {
	playerId   int
	actionType ActionType

	// optional fields
	placeKittenIndex int   // for placing kittens
	useCardIndex     int   // card that you place
	alterFutureOrder []int // new order of first 3 cards (e.g., [2, 1, 0] to reverse)
	targetedPlayer   int   // player being targeted
	comboIndices     []int // list of card indices used for combo
	requestedCard    Card  // card requested from 3 combo
}

type Lobby struct {
	deck               []Card
	players            []*Player
	currentPlayerIndex int
	turnState          TurnState
	livingPlayers      int
	turnsToTake        int
	underAttack        bool
	discardPile        []Card
	lastAction         LastAction
	actionLog          []LastAction

	targetedPlayer int // relevant for favor, targetedAttack, 2 and 3 card combos

	ActionQueue chan PlayerAction
	JoinQueue   chan JoinRequest
}

func NewLobby() *Lobby {
	return &Lobby{
		players:     make([]*Player, 0),
		ActionQueue: make(chan PlayerAction),
		JoinQueue:   make(chan JoinRequest),
		turnState:   NotStarted,
	}
}

func (lobby *Lobby) startGame() error {
	numPlayers := 0
	for _, player := range lobby.players {
		if player.IsOnline {
			numPlayers += 1
		}
	}
	if numPlayers < 2 {
		return errors.New("Cannot start lobby - Not enough players")
	} else if numPlayers > 10 {
		return errors.New("Cannot start lobby - Too many players")
	}
	lobby.livingPlayers = numPlayers
	lobby.turnState = Normal
	lobby.actionLog = nil

	// Create a pool of safe cards
	var safeDeck []Card
	var config = GetDeckConfig(numPlayers)
	for card, count := range config {
		for i := 0; i < count; i++ {
			safeDeck = append(safeDeck, card)
		}
	}

	// Put safe cards in the main deck and shuffle
	lobby.deck = safeDeck
	lobby.shuffleDeck()

	// Deal 1 diffuse + 4 starting cards to each player
	for _, p := range lobby.players {
		if !p.IsOnline {
			continue
		}
		p.Hand = append(p.Hand, Defuse)
		for range 4 {
			p.Hand = append(p.Hand, lobby.removeTopCard())
		}
	}

	// Insert (Players - 1) Exploding Kittens into the remaining deck
	for i := 0; i < numPlayers-1; i++ {
		lobby.deck = append(lobby.deck, ExplodingKitten)
	}

	for i := 0; i < GetExtraDefuses(numPlayers); i++ {
		lobby.deck = append(lobby.deck, Defuse)
	}

	// Final shuffle
	lobby.shuffleDeck()
	lobby.turnsToTake = 1
	fmt.Printf("\n--- Game Setup Complete! Deck has %d cards. ---\n", len(lobby.deck))
	return nil
}

func (lobby *Lobby) eliminatePlayer(playerId int) {
	lobby.livingPlayers--
	lobby.players[playerId].IsAlive = false
	if lobby.currentPlayerIndex == playerId {
		lobby.setNextPlayerTurn(false)
		lobby.turnState = Normal
	}
	if lobby.livingPlayers == 1 && lobby.inProgress() {
		lobby.turnState = GameOver
	}
}

func (lobby *Lobby) disconnectPlayer(playerId int) {
	player := lobby.players[playerId]
	if !player.IsOnline {
		return
	}
	player.IsOnline = false
	close(player.Send)

	for _, player := range lobby.players {
		if player.IsOnline {
			return
		}
	}
	lobby.turnState = GameOver
}

func (lobby *Lobby) discardCard(player *Player, idx int) {
	lobby.discardPile = append(lobby.discardPile, player.Hand[idx])
	player.Hand = slices.Delete(player.Hand, idx, idx+1)
}

func (lobby *Lobby) inProgress() bool {
	return lobby.turnState != NotStarted && lobby.turnState != GameOver
}

func (lobby *Lobby) handleJoin(joinReq JoinRequest) {
	// Reconnect a returning player by their stable id. This works even mid-game
	// (and after they quit) since the player already has a seat at the table.
	if joinReq.UserId != "" {
		for _, p := range lobby.players {
			if p.UserId != joinReq.UserId {
				continue
			}
			// Drop a still-live duplicate connection so its writer goroutine
			// exits; in the normal quit-then-rejoin case Send is already closed.
			if p.IsOnline {
				close(p.Send)
			}
			p.IsOnline = true
			p.Send = joinReq.Send
			if joinReq.Name != "" {
				p.Name = joinReq.Name
			}
			joinReq.Result <- JoinResponse{
				success:  true,
				playerId: p.Id,
			}
			lobby.broadcastGameState()
			return
		}
	}

	// Only returning players may join once the game has started.
	if lobby.inProgress() {
		joinReq.Result <- JoinResponse{
			success: false,
			error:   "Game in progress",
		}
		return
	}

	newId := len(lobby.players)
	name := joinReq.Name
	if name == "" {
		name = fmt.Sprintf("Player %d", newId)
	}
	newPlayer := &Player{
		Id:       newId,
		UserId:   joinReq.UserId,
		Name:     name,
		Send:     joinReq.Send,
		IsOnline: true,
		IsAlive:  true,
		Hand:     make([]Card, 0),
	}
	joinReq.Result <- JoinResponse{
		success:  true,
		playerId: newId,
	}
	lobby.players = append(lobby.players, newPlayer)
	lobby.broadcastGameState()
}

func (lobby *Lobby) run() {
	for {
		select {
		case joinReq := <-lobby.JoinQueue:
			lobby.handleJoin(joinReq)

		case actionReq := <-lobby.ActionQueue:
			if err := lobby.takePlayerAction(actionReq); err != nil {
				lobby.sendError(actionReq.playerId, err.Error())
			} else {
				lobby.broadcastGameState()
			}
		}
	}
}
