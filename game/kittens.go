package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

// 1. Define the Card enum
type Card int

const (
	Defuse Card = iota
	ExplodingKitten
	Skip
	Cat
)

const (
	ExtraDefuses  = 2
	CatMultiplier = 4
	SkipMultipler = 2
)

func (c Card) String() string {
	switch c {
	case Defuse:
		return "🛡️ Defuse"
	case ExplodingKitten:
		return "💥 EXPLODING KITTEN"
	case Skip:
		return "⏭️  SKIP"
	case Cat:
		return "🐈 CAT"
	default:
		return "UNKNOWN"
	}
}

// 2. Define the Player
type Player struct {
	Hand     []Card
	IsAlive  bool
	IsOnline bool

	Send chan GameState
}

type PlayerGameState struct {
	cards    int
	IsAlive  bool
	IsOnline bool
}

type GameState struct {
	playerId       int
	remainingCards int
	players        []PlayerGameState
	turnState      TurnState

	err string
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

type ActionType = int

const (
	PlayCard ActionType = iota
	DrawCard
	PlaceKitten
	Quit
)

type PlayerAction struct {
	playerId   int
	actionType ActionType

	index int // optional; used for placing kittens
}

type TurnState = int

const (
	Normal TurnState = iota
	AwaitingKittenPlacement
	// TODO: add things like awaiting nope, awaiting alter the future, awaiting favor, awaiting 5 unique, etc
)

type Lobby struct {
	deck               []Card
	players            []*Player
	currentPlayerIndex int
	inProgress         bool
	turnState          TurnState

	ActionQueue chan PlayerAction
	JoinQueue   chan JoinRequest
}

// Global state
var (
	scanner = bufio.NewScanner(os.Stdin)
	rng     = rand.New(rand.NewSource(time.Now().UnixNano()))
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

func (lobby *Lobby) getActivePlayerCount() int {
	count := 0
	for _, p := range lobby.players {
		if p.IsAlive {
			count++
		}
	}
	return count
}

// --- Setup & Game Loop ---

func (lobby *Lobby) setupDeck(numPlayers int) {
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

func (lobby *Lobby) playTurn(action PlayerAction) {
	player := lobby.players[action.playerId]
	isPlayerTurn := action.playerId == lobby.currentPlayerIndex
	switch action.actionType {
	case Quit:
		if player.IsOnline {
			player.IsAlive = false
			player.IsOnline = false
			close(player.Send)
			// TODO: make sure this player gets out so lobby doesn't freeze
		}
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
				lobby.turnState = PlaceKitten

				fmt.Printf("Enter the new kitten position (0-%d)", len(lobby.deck))
				scanner.Scan()
				newKittenPosition, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
				if err != nil || newKittenPosition < 0 || newKittenPosition > len(lobby.deck) {
					fmt.Println("Invalid input. Putting kitten on top")
					newKittenPosition = 0
				}
				lobby.deck = slices.Insert(lobby.deck, newKittenPosition, ExplodingKitten)
			} else {
				player.IsAlive = false
			}
		}
	}
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
		playerId:       playerIdx,
		remainingCards: len(lobby.deck),
		turnState:      lobby.turnState,
	}
	for _, player := range lobby.players {
		res.players = append(res.players, PlayerGameState{
			cards:    len(player.Hand),
			IsAlive:  player.IsAlive,
			IsOnline: player.IsOnline,
		})
	}
	return res
}

func (lobby *Lobby) sendError(playerIdx int, err string) {
	res := lobby.getGameState(playerIdx)
	res.err = err
	lobby.players[playerIdx].Send <- res
}

func (lobby *Lobby) broadcastGameState() {
	for idx, player := range lobby.players {
		player.Send <- lobby.getGameState(idx)
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

		case actionReq := <-lobby.ActionQueue:
			fmt.Print(actionReq) // TODO: handle action request
		}
	}
}
