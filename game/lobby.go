package main

import (
	"errors"
	"fmt"
	"slices"
	"time"
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
	AcceptingNopes
)

const nonKittenCardsPerPlayer = 7

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
	case AcceptingNopes:
		return "ACCEPTING_NOPES"
	default:
		return "UNKNOWN"
	}
}

type Player struct {
	Hand          []Card
	Id            int
	DiscordUserId string // stable cross-session identity used to reconnect returning players
	Name          string
	Avatar        string // avatar image URL; empty when the player has none (client falls back to an emoji)
	IsAlive       bool
	IsOnline      bool

	Send chan GameState
}

type PlayerGameState struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
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
	IsSpectator bool              `json:"isSpectator"` // true for watch-only clients that joined mid-game

	Future         []string `json:"future,omitempty"`         // for see/alter the future
	DiscardOptions []string `json:"discardOptions,omitempty"` // discard pile for 5 unique
	TargetedPlayer int      `json:"targetedPlayer"`           // for actions that require another player's response
	IsNoped        bool     `json:"isNoped,omitempty"`        // indicates whether pending action is noped
	NopeDeadline   int64    `json:"nopeDeadline,omitempty"`   // unix ms when the nope window closes
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
	Avatar string
	Send   chan GameState
	Result chan JoinResponse
}

type JoinResponse struct {
	success     bool
	error       string
	playerId    int
	isSpectator bool // true when the join was accepted as a watch-only spectator
}

// Spectator is a watch-only client that joined after the game began. Spectators
// hold no seat at the table and only ever receive the fully public game state.
// Name/DiscordUserId/Avatar are retained so a spectator can be promoted to a
// real Player on lobby restart.
type Spectator struct {
	Id            int
	Name          string
	DiscordUserId string
	Avatar        string
	Send          chan GameState
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
	PlayNope
	RandomizeOrder
	RestartLobby
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
	"PLAY_NOPE":         PlayNope,
	"RANDOMIZE_ORDER":   RandomizeOrder,
	"RESTART_LOBBY":     RestartLobby,
}

type PlayerAction struct {
	playerId   int
	actionType ActionType

	// optional fields
	placeKittenIndex int            // for placing kittens
	useCardIndex     int            // card that you place
	alterFutureOrder []int          // new order of first 3 cards (e.g., [2, 1, 0] to reverse)
	targetedPlayer   int            // player being targeted
	comboIndices     []int          // list of card indices used for combo
	requestedCard    Card           // card requested from 3 combo
	wantNoped        bool           // when placing a nope, true means player wants to nope, false means yup
	conn             chan GameState // only used for disconnect actions to ensure right channel is closed
}

type PendingNopeableAction struct {
	playerId   int
	actionType ActionType

	// optional fields
	playedCard     Card // for play card actions
	comboSize      int  // for combo actions
	targetedPlayer int  // player being targeted
	requestedCard  Card // card requested from 3 combo
	isNoped        bool
}

type Lobby struct {
	deck            []Card
	playersMap      map[int]*Player
	playersList     []*Player
	spectators      map[int]*Spectator
	nextId          int // single counter so player and spectator ids never collide
	currentPlayerId int
	turnState       TurnState
	livingPlayers   int
	turnsToTake     int
	underAttack     bool
	discardPile     []Card
	lastAction      LastAction
	actionLog       []LastAction

	targetedPlayer int // relevant for favor, targetedAttack, 2 and 3 card combos

	pendingAction *PendingNopeableAction // relevant when card is nopeable
	nopeTimer     *time.Timer
	nopeDeadline  time.Time // when the current nope window closes

	ActionQueue chan PlayerAction
	JoinQueue   chan JoinRequest
}

func NewLobby() *Lobby {
	return &Lobby{
		playersList: make([]*Player, 0),
		playersMap:  make(map[int]*Player),
		spectators:  make(map[int]*Spectator),
		ActionQueue: make(chan PlayerAction),
		JoinQueue:   make(chan JoinRequest),
		turnState:   NotStarted,
		nextId:      0,
	}
}

func (lobby *Lobby) startGame() error {
	numPlayers := len(lobby.playersList)
	if numPlayers < 2 {
		return errors.New("Cannot start lobby - Not enough players")
	} else if numPlayers > 10 {
		return errors.New("Cannot start lobby - Too many players")
	}
	lobby.livingPlayers = numPlayers
	lobby.turnState = Normal
	lobby.actionLog = nil
	lobby.currentPlayerId = lobby.playersList[0].Id

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
	for _, p := range lobby.playersList {
		if !p.IsOnline {
			continue
		}
		p.Hand = append(p.Hand, Defuse)
		for range nonKittenCardsPerPlayer {
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
	lobby.playersMap[playerId].IsAlive = false
	if lobby.currentPlayerId == playerId {
		lobby.setNextPlayerTurn(false)
		lobby.turnState = Normal
	}
	if lobby.livingPlayers == 1 && lobby.inProgress() {
		lobby.turnState = GameOver
	}
}

func (lobby *Lobby) disconnectPlayer(playerId int) {
	player := lobby.playersMap[playerId]
	if !player.IsOnline {
		return
	}
	player.IsOnline = false
	close(player.Send)

	// drop players who leave before the game starts
	if lobby.turnState == NotStarted {
		delete(lobby.playersMap, playerId)
		lobby.playersList = slices.DeleteFunc(lobby.playersList, func(p *Player) bool {
			return p.Id == playerId
		})
	}
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
		for _, p := range lobby.playersList {
			if p.DiscordUserId != joinReq.UserId {
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
			if joinReq.Avatar != "" {
				p.Avatar = joinReq.Avatar
			}
			joinReq.Result <- JoinResponse{
				success:  true,
				playerId: p.Id,
			}
			lobby.broadcastGameState()
			return
		}
	}

	// player without an id who joins can replace a disconnected id-less player
	if joinReq.UserId == "" && lobby.inProgress() {
		for _, p := range lobby.playersList {
			if p.DiscordUserId != "" || p.IsOnline {
				continue
			}
			p.IsOnline = true
			p.Send = joinReq.Send
			if joinReq.Name != "" {
				p.Name = joinReq.Name
			}
			if joinReq.Avatar != "" {
				p.Avatar = joinReq.Avatar
			}
			joinReq.Result <- JoinResponse{
				success:  true,
				playerId: p.Id,
			}
			lobby.broadcastGameState()
			return
		}
	}

	// joining an already started lobby turns player to spectator
	if lobby.turnState != NotStarted {
		specId := lobby.nextId
		lobby.nextId++
		lobby.spectators[specId] = &Spectator{
			Id:            specId,
			Name:          joinReq.Name,
			DiscordUserId: joinReq.UserId,
			Avatar:        joinReq.Avatar,
			Send:          joinReq.Send,
		}
		joinReq.Result <- JoinResponse{
			success:     true,
			playerId:    specId,
			isSpectator: true,
		}
		lobby.broadcastGameState()
		return
	}

	newId := lobby.nextId
	lobby.nextId++
	name := joinReq.Name
	if name == "" {
		name = fmt.Sprintf("Player %d", newId)
	}
	newPlayer := &Player{
		Id:            newId,
		DiscordUserId: joinReq.UserId,
		Name:          name,
		Avatar:        joinReq.Avatar,
		Send:          joinReq.Send,
		IsOnline:      true,
		IsAlive:       true,
		Hand:          make([]Card, 0),
	}
	joinReq.Result <- JoinResponse{
		success:  true,
		playerId: newId,
	}
	lobby.playersList = append(lobby.playersList, newPlayer)
	lobby.playersMap[newPlayer.Id] = newPlayer
	lobby.broadcastGameState()
}

func (lobby *Lobby) run() {
	for {
		var nopeC <-chan time.Time
		if lobby.nopeTimer != nil {
			nopeC = lobby.nopeTimer.C
		}
		select {
		case joinReq := <-lobby.JoinQueue:
			lobby.handleJoin(joinReq)

		case actionReq := <-lobby.ActionQueue:
			if err := lobby.receivePlayerAction(actionReq); err != nil {
				lobby.sendError(actionReq.playerId, err.Error())
			} else {
				lobby.broadcastGameState()
			}

		case <-nopeC:
			lobby.handleNopeTimerComplete()
		}
	}
}

// resetToLobby returns the lobby to the pre-game NOT_STARTED state: it clears
// all game state, drops players who went offline mid-game, promotes every
// spectator into a seated player in place (same id + socket), and records who
// triggered the restart so the banner can show it.
func (lobby *Lobby) resetToLobby(restarterId int) {
	if lobby.nopeTimer != nil {
		lobby.nopeTimer.Stop()
		lobby.nopeTimer = nil
	}
	lobby.pendingAction = nil
	lobby.deck = nil
	lobby.discardPile = nil
	lobby.turnsToTake = 0
	lobby.underAttack = false
	lobby.targetedPlayer = 0
	lobby.livingPlayers = 0

	name := lobby.playerName(restarterId)

	// Drop players who left during the game so the NOT_STARTED invariant
	// ("every seated player is online") holds again.
	lobby.playersList = slices.DeleteFunc(lobby.playersList, func(p *Player) bool {
		if !p.IsOnline {
			delete(lobby.playersMap, p.Id)
			return true
		}
		return false
	})

	// Promote spectators to players in place: same id and Send channel, so their
	// existing socket (already in the unified read loop) starts driving actions.
	for id, spec := range lobby.spectators {
		specName := spec.Name
		if specName == "" {
			specName = fmt.Sprintf("Player %d", id)
		}
		player := &Player{
			Id:            id,
			DiscordUserId: spec.DiscordUserId,
			Name:          specName,
			Avatar:        spec.Avatar,
			Send:          spec.Send,
			IsOnline:      true,
			IsAlive:       true,
			Hand:          make([]Card, 0),
		}
		lobby.playersList = append(lobby.playersList, player)
		lobby.playersMap[id] = player
		delete(lobby.spectators, id)
	}

	// Deal everyone back out fresh.
	for _, p := range lobby.playersList {
		p.Hand = make([]Card, 0)
		p.IsAlive = true
	}

	lobby.turnState = NotStarted
	if len(lobby.playersList) > 0 {
		lobby.currentPlayerId = lobby.playersList[0].Id
	}
	lobby.actionLog = nil
	lobby.recordAction(LastAction{Public: fmt.Sprintf("%s restarted the lobby", name)})
}
