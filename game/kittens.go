package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"slices"
	"time"
)

const (
	ExtraDefuses             = 2
	CatMultiplier            = 4
	SkipMultiplier           = 2
	SeeTheFutureMultiplier   = 2
	AlterTheFutureMultiplier = 2
)

type Card int

const (
	Defuse Card = iota
	ExplodingKitten
	Skip
	Cat
	SeeTheFuture
	AlterTheFuture
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
	case SeeTheFuture:
		return "SEE_THE_FUTURE"
	case AlterTheFuture:
		return "ALTER_THE_FUTURE"
	default:
		return "UNKNOWN"
	}
}

type TurnState int

const (
	Normal TurnState = iota
	NotStarted
	GameOver
	AwaitingKittenPlacement
	SeeingTheFuture
	AlteringTheFuture
	// TODO: add things like awaiting nope, awaiting favor, awaiting 5 unique, etc
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
	default:
		return "UNKNOWN"
	}
}

type Player struct {
	Hand     []Card
	Id       int
	IsAlive  bool
	IsOnline bool

	Send chan GameState
}

type PlayerGameState struct {
	Id        int  `json:"id"`
	CardCount int  `json:"cardCount"`
	IsAlive   bool `json:"isAlive"`
	IsOnline  bool `json:"isOnline"`
}

type GameState struct {
	PlayerId   int               `json:"playerId"`
	TurnId     int               `json:"turnId"`
	DeckSize   int               `json:"deckSize"`
	Players    []PlayerGameState `json:"players"`
	TurnState  string            `json:"turnState"`
	Hand       []string          `json:"hand"`
	InProgress bool              `json:"inProgress"`

	Future []string `json:"future,omitempty"` // for see/alter the future
	Err    string   `json:"err,omitempty"`
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
	AlterFuture
)

var actionTypeNames = map[string]ActionType{
	"START_GAME":   StartGame,
	"PLAY_CARD":    PlayCard,
	"DRAW_CARD":    DrawCard,
	"PLACE_KITTEN": PlaceKitten,
	"DISCONNECT":   Disconnect,
	"ALTER_FUTURE": AlterFuture,
}

type PlayerAction struct {
	playerId   int
	actionType ActionType

	// optional fields
	index   int   // for placing kittens
	indices []int // for altering the future
}

type Lobby struct {
	deck               []Card
	players            []*Player
	currentPlayerIndex int
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
		turnState:   NotStarted,
	}
}

func (lobby *Lobby) shuffleDeck() {
	rng.Shuffle(len(lobby.deck), func(i, j int) {
		lobby.deck[i], lobby.deck[j] = lobby.deck[j], lobby.deck[i]
	})
}

func (lobby *Lobby) removeTopCard() Card {
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

func (lobby *Lobby) startGame() error {
	numPlayers := 0
	for _, player := range lobby.players {
		if player.IsOnline {
			numPlayers += 1
		}
	}
	if numPlayers < 2 {
		return errors.New("Cannot start lobby - Not enough players")
	}
	lobby.livingPlayers = numPlayers
	lobby.turnState = Normal

	// Create a pool of safe cards (Lots of Cats, some Skips)
	var safeDeck []Card
	for i := 0; i < numPlayers*CatMultiplier; i++ {
		safeDeck = append(safeDeck, Cat)
	}
	for i := 0; i < numPlayers*SkipMultiplier; i++ {
		safeDeck = append(safeDeck, Skip)
	}
	for i := 0; i < numPlayers*SeeTheFutureMultiplier; i++ {
		safeDeck = append(safeDeck, SeeTheFuture)
	}
	for i := 0; i < numPlayers*AlterTheFutureMultiplier; i++ {
		safeDeck = append(safeDeck, AlterTheFuture)
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

	for i := 0; i < ExtraDefuses; i++ {
		lobby.deck = append(lobby.deck, Defuse)
	}

	// Final shuffle
	lobby.shuffleDeck()
	fmt.Printf("\n--- Game Setup Complete! Deck has %d cards. ---\n", len(lobby.deck))
	return nil
}

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

	case Disconnect:
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

		if drawn == ExplodingKitten {
			if defuseIndex := slices.Index(player.Hand, Defuse); defuseIndex != -1 {
				player.Hand = slices.Delete(player.Hand, defuseIndex, defuseIndex+1)
				lobby.turnState = AwaitingKittenPlacement
			} else {
				player.IsAlive = false
			}
		} else {
			player.Hand = append(player.Hand, drawn)
			lobby.setNextPlayerTurn()
		}

	case PlaceKitten:
		if err := lobby.assertTurnAndState([]TurnState{AwaitingKittenPlacement}, isPlayerTurn, "place kitten"); err != nil {
			return err
		}
		newKittenPosition := action.index
		if newKittenPosition < 0 || newKittenPosition > len(lobby.deck) {
			return errors.New("Invalid kitten position")
		}
		lobby.deck = slices.Insert(lobby.deck, newKittenPosition, ExplodingKitten)
		lobby.turnState = Normal
		lobby.setNextPlayerTurn()

	case PlayCard:
		if err := lobby.assertTurnAndState([]TurnState{Normal, SeeingTheFuture}, isPlayerTurn, "play card"); err != nil {
			return err
		}
		if action.index < 0 || action.index >= len(player.Hand) {
			return errors.New("Cannot play card - No card found at that index")
		}

		lobby.turnState = Normal // clear effects like seeing the future
		playedCard := player.Hand[action.index]
		player.Hand = slices.Delete(player.Hand, action.index, action.index+1)

		switch playedCard {
		case Skip:
			lobby.setNextPlayerTurn() // TODO: make this decrease attacks by 1 instead
		case SeeTheFuture:
			lobby.turnState = SeeingTheFuture
		case AlterTheFuture:
			lobby.turnState = AlteringTheFuture
		default:
			return errors.New("Cannot play that card")
		}

	case AlterFuture:
		if err := lobby.assertTurnAndState([]TurnState{AlteringTheFuture}, isPlayerTurn, "alter future"); err != nil {
			return err
		}
		alterSize := min(3, len(lobby.deck))
		if len(action.indices) != alterSize {
			return fmt.Errorf("Expected indices to have length %d, got %d", alterSize, len(action.indices))
		}
		for i := range action.indices {
			if !slices.Contains(action.indices, i) {
				return fmt.Errorf("Invalid indices list - missing value %d", i)
			}
		}
		buffer := make([]Card, alterSize)
		for newIndex, oldIndex := range action.indices {
			buffer[newIndex] = lobby.deck[oldIndex]
		}
		copy(lobby.deck, buffer)
		lobby.turnState = Normal
	}

	return nil
}

func (lobby *Lobby) assertTurnAndState(validStates []TurnState, isPlayerTurn bool, action string) error {
	if !slices.Contains(validStates, lobby.turnState) {
		return errors.New("Cannot " + action + " - Invalid state")
	}
	if !isPlayerTurn {
		return errors.New("Cannot " + action + " - Not your turn")
	}
	return nil
}

func (lobby *Lobby) eliminatePlayer(playerId int) {
	lobby.livingPlayers--
	lobby.players[playerId].IsAlive = false
	if lobby.currentPlayerIndex == playerId {
		lobby.setNextPlayerTurn()
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
	lobby.eliminatePlayer(playerId)
}

func (lobby *Lobby) setNextPlayerTurn() {
	if lobby.livingPlayers == 0 {
		return // Safety valve to prevent infinite loops
	}
	idx := lobby.currentPlayerIndex
	for {
		idx = (idx + 1) % len(lobby.players)
		if lobby.players[idx].IsAlive {
			lobby.currentPlayerIndex = idx
			return
		}
	}
}

func cardSliceToStrings(cards []Card) []string {
	res := make([]string, len(cards))
	for i, card := range cards {
		res[i] = card.String()
	}
	return res
}

func (lobby *Lobby) getGameState(playerIdx int) GameState {
	player := lobby.players[playerIdx]
	hand := cardSliceToStrings(player.Hand)
	var future []string = nil
	if (lobby.turnState == SeeingTheFuture || lobby.turnState == AlteringTheFuture) && lobby.currentPlayerIndex == playerIdx {
		count := min(3, len(player.Hand))
		future = cardSliceToStrings(lobby.deck[:count])
	}
	res := GameState{
		PlayerId:   playerIdx,
		TurnId:     lobby.currentPlayerIndex,
		DeckSize:   len(lobby.deck),
		TurnState:  lobby.turnState.String(),
		Hand:       hand,
		InProgress: lobby.inProgress(),

		Future: future,
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

func (lobby *Lobby) inProgress() bool {
	return lobby.turnState != NotStarted && lobby.turnState != GameOver
}

func (lobby *Lobby) run() {
	for {
		select {
		case joinReq := <-lobby.JoinQueue:
			if lobby.inProgress() {
				joinReq.Result <- JoinResponse{
					success: false,
					error:   "Game in progress",
				}
			}
			newId := len(lobby.players)
			newPlayer := &Player{
				Id:       newId,
				Send:     joinReq.Send,
				IsOnline: true,
				IsAlive:  true,
				Hand:     make([]Card, 0),
			}
			joinReq.Result <- JoinResponse{
				success:  true,
				playerId: newId, // TODO: make this resistant to players exiting
			}
			lobby.players = append(lobby.players, newPlayer)
			lobby.broadcastGameState()

		case actionReq := <-lobby.ActionQueue:
			if err := lobby.takePlayerAction(actionReq); err != nil {
				lobby.sendError(actionReq.playerId, err.Error())
			} else {
				lobby.broadcastGameState()
			}
		}
	}
}
