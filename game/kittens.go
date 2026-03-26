package main

import (
	"fmt"
	"math/rand"
	"os"
	"slices"
	"time"
)

const (
	ExtraDefuses  = 2
	CatMultiplier = 4
	SkipMultipler = 2
)

type Card int

const (
	Defuse Card = iota
	ExplodingKitten
	Skip
	Cat
)

func (c Card) String() string {
	switch c {
	case Defuse:
		return "DEFUSE"
	case ExplodingKitten:
		return "EXPLODING_KITTEN"
	case Skip:
		return "SKIP"
	case Cat:
		return "CAT"
	default:
		return "UNKNOWN"
	}
}

type TurnState int

const (
	Normal TurnState = iota
	GameOver
	AwaitingKittenPlacement
	// TODO: add things like awaiting nope, awaiting alter the future, awaiting favor, awaiting 5 unique, etc
)

func (t TurnState) String() string {
	switch t {
	case Normal:
		return "NORMAL"
	case GameOver:
		return "GAME_OVER"
	case AwaitingKittenPlacement:
		return "AWAITING_KITTEN_PLACEMENT"
	default:
		return "UNKNOWN"
	}
}

type Player struct {
	Hand     []Card
	IsAlive  bool
	IsOnline bool

	Send chan GameState
}

type PlayerGameState struct {
	Cards    int  `json:"cards"`
	IsAlive  bool `json:"isAlive"`
	IsOnline bool `json:"isOnline"`
}

type GameState struct {
	PlayerId       int
	RemainingCards int
	Players        []PlayerGameState
	TurnState      string

	Err string
}

type JoinRequest struct {
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

	Unknown
)

var actionTypeNames = map[string]ActionType{
	"START_GAME":   StartGame,
	"PLAY_CARD":    PlayCard,
	"DRAW_CARD":    DrawCard,
	"PLACE_KITTEN": PlaceKitten,
	"DISCONNECT":   Disconnect,
}

type PlayerAction struct {
	playerId   int
	actionType ActionType

	index int // optional; used for placing kittens
}

type Lobby struct {
	deck               []Card
	players            []*Player
	currentPlayerIndex int
	inProgress         bool
	turnState          TurnState
	livingPlayers      int

	ActionQueue chan PlayerAction
	JoinQueue   chan JoinRequest
}

// Global state
var (
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// --- Helper Functions ---
func NewLobby() *Lobby {
	return &Lobby{
		players:     make([]*Player, 0),
		ActionQueue: make(chan PlayerAction),
		JoinQueue:   make(chan JoinRequest),
	}
}

func (lobby *Lobby) shuffleDeck() {
	rng.Shuffle(len(lobby.deck), func(i, j int) {
		lobby.deck[i], lobby.deck[j] = lobby.deck[j], lobby.deck[i]
	})
}

func (lobby *Lobby) drawCard() Card {
	if len(lobby.deck) == 0 {
		fmt.Println("The deck is empty! (This shouldn't happen with correct kitten math)")
		os.Exit(1)
	}
	// Pop the top card
	drawn := lobby.deck[0]
	lobby.deck = lobby.deck[1:]
	return drawn
}

// --- Setup & Game Loop ---

func (lobby *Lobby) startGame() {
	numPlayers := lobby.livingPlayers
	lobby.inProgress = true
	// Create a pool of safe cards (Lots of Cats, some Skips)
	var safeDeck []Card
	for i := 0; i < numPlayers*CatMultiplier; i++ {
		safeDeck = append(safeDeck, Cat)
	}
	for i := 0; i < numPlayers*SkipMultipler; i++ {
		safeDeck = append(safeDeck, Skip)
	}

	// Put safe cards in the main deck and shuffle
	lobby.deck = safeDeck
	lobby.shuffleDeck()

	// Deal 1 diffuse + 4 starting cards to each player
	for _, p := range lobby.players {
		p.Hand = append(p.Hand, Defuse)
		for i := 0; i < 4; i++ {
			p.Hand = append(p.Hand, lobby.drawCard())
		}
	}

	// Insert (Players - 1) Exploding Kittens into the remaining deck
	for i := 0; i < numPlayers-1; i++ {
		lobby.deck = append(lobby.deck, ExplodingKitten)
	}

	for i := 0; i < ExtraDefuses; i++ {
		lobby.deck = append(lobby.deck, Defuse)
	}

	// Final shuffle
	lobby.shuffleDeck()
	fmt.Printf("\n--- Game Setup Complete! Deck has %d cards. ---\n", len(lobby.deck))
}

func (lobby *Lobby) takePlayerAction(action PlayerAction) {
	playerId := action.playerId
	player := lobby.players[playerId]
	isPlayerTurn := action.playerId == lobby.currentPlayerIndex
	switch action.actionType {
	case StartGame:
		if lobby.inProgress {
			lobby.sendError(playerId, "Cannot start lobby - Lobby already in progress")
			return
		}
		if lobby.livingPlayers < 2 {
			lobby.sendError(playerId, "Cannot start lobby - Not enough players")
			return
		}
		lobby.startGame()

	case Disconnect:
		if !player.IsOnline {
			fmt.Printf("Illegal state? Player id %d is offline but disconnected", playerId)
		}
		lobby.disconnectPlayer(playerId)

	case DrawCard:
		if !isPlayerTurn {
			lobby.sendError(action.playerId, "Not your turn")
			return
		}
		if lobby.turnState != Normal {
			lobby.sendError(action.playerId, "Cannot pick up cards right now")
			return
		}

		drawn := lobby.drawCard()

		if drawn == ExplodingKitten {
			if defuseIndex := slices.Index(player.Hand, Defuse); defuseIndex != -1 {
				player.Hand = append(player.Hand[:defuseIndex], player.Hand[defuseIndex+1:]...)
				lobby.turnState = AwaitingKittenPlacement
			} else {
				player.IsAlive = false
			}
		}

	case PlaceKitten:
		if !isPlayerTurn {
			lobby.sendError(playerId, "Not your turn")
			return
		}
		newKittenPosition := action.index
		if newKittenPosition < 0 || newKittenPosition > len(lobby.deck) {
			lobby.sendError(playerId, "Invalid kitten position")
			return
		}
		lobby.deck = slices.Insert(lobby.deck, newKittenPosition, ExplodingKitten)
		lobby.setNextPlayerTurn()

	case PlayCard:
		if !isPlayerTurn {
			lobby.sendError(playerId, "Not your turn")
			return
		}
		if action.index < 0 || action.index >= len(player.Hand) {
			lobby.sendError(playerId, "No card found at that index")
			return
		}
		playedCard := player.Hand[action.index]
		switch playedCard {
		case Skip:
			lobby.setNextPlayerTurn() // TODO: make this decrease attacks by 1 instead
		default:
			lobby.sendError(playerId, "Cannot play that card")
		}
	}

	lobby.broadcastGameState()
}

func (lobby *Lobby) eliminatePlayer(playerId int) {
	lobby.livingPlayers--
	lobby.players[playerId].IsAlive = false
	if lobby.currentPlayerIndex == playerId {
		lobby.setNextPlayerTurn()
	}
	if lobby.livingPlayers == 1 {
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
	lobby.eliminatePlayer(playerId)
}

func (lobby *Lobby) setNextPlayerTurn() {
	idx := lobby.currentPlayerIndex
	for {
		idx = (idx + 1) % len(lobby.players)
		if lobby.players[idx].IsAlive {
			lobby.currentPlayerIndex = idx
			return
		}
	}
}

func (lobby *Lobby) getGameState(playerIdx int) GameState {
	res := GameState{
		PlayerId:       playerIdx,
		RemainingCards: len(lobby.deck),
		TurnState:      lobby.turnState.String(),
	}
	for _, player := range lobby.players {
		res.Players = append(res.Players, PlayerGameState{
			Cards:    len(player.Hand),
			IsAlive:  player.IsAlive,
			IsOnline: player.IsOnline,
		})
	}
	return res
}

func (lobby *Lobby) sendError(playerIdx int, err string) {
	res := lobby.getGameState(playerIdx)
	res.Err = err
	lobby.players[playerIdx].Send <- res
}

func (lobby *Lobby) broadcastGameState() {
	for playerIdx, player := range lobby.players {
		player.Send <- lobby.getGameState(playerIdx)
	}
}

func (lobby *Lobby) run() {
	for {
		select {
		case joinReq := <-lobby.JoinQueue:
			if lobby.inProgress {
				joinReq.Result <- JoinResponse{
					success: false,
					error:   "Game in progress",
				}
			}
			newPlayer := &Player{
				Send:     joinReq.Send,
				IsOnline: true,
				IsAlive:  true,
				Hand:     make([]Card, 0),
			}
			joinReq.Result <- JoinResponse{
				success:  true,
				playerId: len(lobby.players), // TODO: make this resistant to players exiting
			}
			lobby.players = append(lobby.players, newPlayer)
			lobby.livingPlayers += 1
			lobby.broadcastGameState()

		case actionReq := <-lobby.ActionQueue:
			lobby.takePlayerAction(actionReq)
		}
	}
}
