package main

import (
	"cmp"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"slices"
	"time"
)

const (
	ExtraDefuses = 2
)

var multipliers = map[Card]int{
	Cat1:           4,
	Cat2:           4,
	Cat3:           4,
	Cat4:           4,
	FeralCat:       4,
	Skip:           2,
	SeeTheFuture:   2,
	AlterTheFuture: 2,
	Attack:         2,
	TargetedAttack: 2,
	Shuffle:        2,
	DrawFromBottom: 2,
	Favor:          2,
}

type Card int

const (
	Defuse Card = iota
	ExplodingKitten
	Skip
	Attack
	TargetedAttack
	Cat1
	Cat2
	Cat3
	Cat4
	FeralCat
	SeeTheFuture
	AlterTheFuture
	Shuffle
	DrawFromBottom
	Favor
)

func (c Card) String() string {
	switch c {
	case Defuse:
		return "DEFUSE"
	case ExplodingKitten:
		return "EXPLODING_KITTEN"
	case Skip:
		return "SKIP"
	case Attack:
		return "ATTACK"
	case TargetedAttack:
		return "TARGETED_ATTACK"
	case Cat1:
		return "CAT1"
	case Cat2:
		return "CAT2"
	case Cat3:
		return "CAT3"
	case Cat4:
		return "CAT4"
	case FeralCat:
		return "FERAL_CAT"
	case SeeTheFuture:
		return "SEE_THE_FUTURE"
	case AlterTheFuture:
		return "ALTER_THE_FUTURE"
	case Shuffle:
		return "SHUFFLE"
	case DrawFromBottom:
		return "DRAW_FROM_BOTTOM"
	case Favor:
		return "FAVOR"
	default:
		return "UNKNOWN"
	}
}

func ParseCard(s string) (Card, error) {
	if card, ok := cardNames[s]; ok {
		return card, nil
	}
	return 0, errors.New("invalid card: " + s)
}

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
	Err            string   `json:"err,omitempty"`
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
	GiveFavor
	Combo
	TakeFromDiscard
)

var actionTypeNames = map[string]ActionType{
	"START_GAME":   StartGame,
	"PLAY_CARD":    PlayCard,
	"DRAW_CARD":    DrawCard,
	"PLACE_KITTEN": PlaceKitten,
	"DISCONNECT":   Disconnect,
	"ALTER_FUTURE": AlterFuture,
	"GIVE_FAVOR":   GiveFavor,
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

	targetedPlayer int // relevant for favor, targetedAttack, 2 and 3 card combos

	ActionQueue chan PlayerAction
	JoinQueue   chan JoinRequest
}

// Global state
var (
	rng       = rand.New(rand.NewSource(time.Now().UnixNano()))
	cardNames map[string]Card
)

func init() {
	cardNames = make(map[string]Card)
	for i := Card(0); i <= Favor; i++ {
		s := i.String()
		if s != "UNKNOWN" {
			cardNames[s] = i
		}
	}
}

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

func (lobby *Lobby) removeBottomCard() Card {
	if len(lobby.deck) == 0 {
		fmt.Println("The deck is empty!")
		os.Exit(1)
	}
	// Pop the bottom card
	lastIdx := len(lobby.deck) - 1
	drawn := lobby.deck[lastIdx]
	lobby.deck = lobby.deck[:lastIdx]
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

	// Create a pool of safe cards
	var safeDeck []Card
	for cardType, multiplier := range multipliers {
		count := numPlayers * multiplier
		for i := 0; i < count; i++ {
			safeDeck = append(safeDeck, cardType)
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
		lobby.resolveDrawnCard(player, drawn)

	case PlaceKitten:
		if err := lobby.assertTurnAndState([]TurnState{AwaitingKittenPlacement}, isPlayerTurn, "place kitten"); err != nil {
			return err
		}
		newKittenPosition := action.placeKittenIndex
		if newKittenPosition < 0 || newKittenPosition > len(lobby.deck) {
			return errors.New("Invalid kitten position")
		}
		lobby.deck = slices.Insert(lobby.deck, newKittenPosition, ExplodingKitten)
		lobby.turnState = Normal
		lobby.setNextPlayerTurn(false)

	case PlayCard:
		if err := lobby.assertTurnAndState([]TurnState{Normal, SeeingTheFuture}, isPlayerTurn, "play card"); err != nil {
			return err
		}
		if action.useCardIndex < 0 || action.useCardIndex >= len(player.Hand) {
			return errors.New("Cannot play card - No card found at that index")
		}

		playedCard := player.Hand[action.useCardIndex]

		// validate before we take the card away
		if playedCard == TargetedAttack || playedCard == Favor {
			if err := lobby.assertPlayerExistsAndAlive(action.targetedPlayer); err != nil {
				return err
			}
		}
		if playedCard == Favor && len(lobby.players[action.targetedPlayer].Hand) == 0 {
			return errors.New("Cannot ask a favor from a player without cards!")
		}

		lobby.discardCard(player, action.useCardIndex)

		switch playedCard {
		case Skip:
			lobby.turnState = Normal
			lobby.decreaseTurns()
		case SeeTheFuture:
			lobby.turnState = SeeingTheFuture
		case AlterTheFuture:
			lobby.turnState = AlteringTheFuture
		case Attack:
			lobby.turnState = Normal
			lobby.setNextPlayerTurn(true)
		case TargetedAttack:
			lobby.turnState = Normal
			lobby.setPlayerTurn(true, action.targetedPlayer)
		case Shuffle:
			lobby.turnState = Normal
			lobby.shuffleDeck()
		case DrawFromBottom:
			lobby.turnState = Normal
			drawn := lobby.removeBottomCard()
			lobby.resolveDrawnCard(player, drawn)
		case Favor:
			lobby.turnState = AwaitingFavor
			lobby.targetedPlayer = action.targetedPlayer
		default:
			return errors.New("Cannot play that card")
		}

	case AlterFuture:
		if err := lobby.assertTurnAndState([]TurnState{AlteringTheFuture}, isPlayerTurn, "alter future"); err != nil {
			return err
		}
		alterSize := min(3, len(lobby.deck))
		if len(action.alterFutureOrder) != alterSize {
			return fmt.Errorf("Expected indices to have length %d, got %d", alterSize, len(action.alterFutureOrder))
		}
		for i := range action.alterFutureOrder {
			if !slices.Contains(action.alterFutureOrder, i) {
				return fmt.Errorf("Invalid indices list - missing value %d", i)
			}
		}
		buffer := make([]Card, alterSize)
		for newIndex, oldIndex := range action.alterFutureOrder {
			buffer[newIndex] = lobby.deck[oldIndex]
		}
		copy(lobby.deck, buffer)
		lobby.turnState = Normal

	case GiveFavor:
		if lobby.turnState != AwaitingFavor || lobby.targetedPlayer != playerId {
			return errors.New("Must be the target of a favor request")
		}
		if action.useCardIndex < 0 || action.useCardIndex >= len(player.Hand) {
			return errors.New("Given card is out of bounds")
		}

		lobby.turnState = Normal
		transferredCard := player.Hand[action.useCardIndex]
		requester := lobby.players[lobby.currentPlayerIndex]
		player.Hand = slices.Delete(player.Hand, action.useCardIndex, action.useCardIndex+1)
		requester.Hand = append(requester.Hand, transferredCard)

	case Combo:
		if err := lobby.assertTurnAndState([]TurnState{Normal, SeeingTheFuture}, isPlayerTurn, "combo"); err != nil {
			return err
		}
		if err := assertUniqueAndInBounds(action.comboIndices, len(player.Hand)); err != nil {
			return err
		}
		lobby.turnState = Normal
		comboSize := len(action.comboIndices)
		comboCards := make([]Card, len(action.comboIndices))
		for i, cardIdx := range action.comboIndices {
			comboCards[i] = player.Hand[cardIdx]
		}
		switch comboSize {
		case 2, 3:
			if err := lobby.assertPlayerExistsAndAlive(action.targetedPlayer); err != nil {
				return err
			}
			targetedPlayer := lobby.players[action.targetedPlayer]
			if len(targetedPlayer.Hand) == 0 {
				return errors.New("Cannot target a player without cards in their hand")
			}
			if err := assertValidMatchingCombo(comboCards); err != nil {
				return err
			}
			deleteIndex := -1
			switch comboSize {
			case 2:
				deleteIndex = rand.Intn(len(targetedPlayer.Hand))
			case 3:
				deleteIndex = slices.Index(targetedPlayer.Hand, action.requestedCard)
			}
			if deleteIndex != -1 {
				player.Hand = append(player.Hand, targetedPlayer.Hand[deleteIndex])
				targetedPlayer.Hand = slices.Delete(targetedPlayer.Hand, deleteIndex, deleteIndex+1)
			}
		case 5:
			if len(lobby.discardPile) == 0 {
				return errors.New("No cards to take from the discard pile!")
			}
			uniqueCards := make(map[Card]bool)
			for _, c := range comboCards {
				uniqueCards[c] = true
			}
			if len(uniqueCards) != 5 {
				return errors.New("All 5 cards must be unique")
			}
			lobby.turnState = AwaitingDiscardTake
		default:
			return errors.New("Combos must contain 2, 3, or 5 cards")
		}
		// at this point we know it was successful, remove the cards
		slices.SortFunc(action.comboIndices, func(a, b int) int {
			return cmp.Compare(b, a)
		})
		for _, cardIdx := range action.comboIndices {
			lobby.discardCard(player, cardIdx)
		}

	case TakeFromDiscard:
		if err := lobby.assertTurnAndState([]TurnState{AwaitingDiscardTake}, isPlayerTurn, "take from discard"); err != nil {
			return err
		}
		if takeIdx := slices.Index(lobby.discardPile, action.requestedCard); takeIdx != -1 {
			player.Hand = append(player.Hand, lobby.discardPile[takeIdx])
			lobby.discardPile = slices.Delete(lobby.discardPile, takeIdx, takeIdx+1)
		} else {
			return errors.New("Card is not in discard pile!")
		}
		lobby.turnState = Normal
	}
	return nil
}

func (lobby *Lobby) resolveDrawnCard(player *Player, drawn Card) {
	if drawn == ExplodingKitten {
		if defuseIndex := slices.Index(player.Hand, Defuse); defuseIndex != -1 {
			player.Hand = slices.Delete(player.Hand, defuseIndex, defuseIndex+1)
			lobby.turnState = AwaitingKittenPlacement
		} else {
			player.IsAlive = false
		}
	} else {
		player.Hand = append(player.Hand, drawn)
		lobby.decreaseTurns()
	}
}

func (lobby *Lobby) decreaseTurns() {
	lobby.turnsToTake -= 1
	if lobby.turnsToTake == 0 {
		lobby.setNextPlayerTurn(false)
	}
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

func (lobby *Lobby) assertPlayerExistsAndAlive(playerId int) error {
	if playerId < 0 || playerId >= len(lobby.players) {
		return errors.New("Player does not exist")
	} else if !lobby.players[playerId].IsAlive {
		return errors.New("Player must be alive")
	}
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
	lobby.eliminatePlayer(playerId)
}

func (lobby *Lobby) setNextPlayerTurn(attack bool) {
	if lobby.livingPlayers == 0 {
		return // Safety valve to prevent infinite loops
	}
	idx := lobby.currentPlayerIndex
	for {
		idx = (idx + 1) % len(lobby.players)
		if lobby.players[idx].IsAlive {
			lobby.setPlayerTurn(attack, idx)
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
			// Check if we have already processed this specific card value
			if !discardMap[c] {
				discardMap[c] = true
				discardOptions = append(discardOptions, c.String())
			}
		}
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

func (lobby *Lobby) discardCard(player *Player, idx int) {
	lobby.discardPile = append(lobby.discardPile, player.Hand[idx])
	player.Hand = slices.Delete(player.Hand, idx, idx+1)
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

func (c Card) isCat() bool {
	return c == Cat1 || c == Cat2 || c == Cat3 || c == Cat4 || c == FeralCat
}

// returns true if either all the cards are the same, or it contains only 1 cat type + feral cats
func assertValidMatchingCombo(cards []Card) error {
	if len(cards) <= 1 {
		return nil
	}

	counts := make(map[Card]bool)
	for _, c := range cards {
		counts[c] = true
	}

	if len(counts) == 1 {
		return nil
	}

	if len(counts) == 2 {
		hasFeral := false
		allAreCats := true

		for c := range counts {
			if c == FeralCat {
				hasFeral = true
			}
			if !c.isCat() {
				allAreCats = false
			}
		}

		if hasFeral && allAreCats {
			return nil
		} else {
			return errors.New("Not a matching combo")
		}
	}

	return errors.New("Not a matching combo")
}

// check if each element in the array is between 0 and n and that they're all unique
func assertUniqueAndInBounds(indices []int, n int) error {
	counts := make(map[int]bool)
	for _, el := range indices {
		counts[el] = true
		if el < 0 || el >= n {
			return errors.New("All indices must be in bounds")
		}
	}
	if len(counts) != len(indices) {
		return errors.New("All elements must be unique")
	}
	return nil
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
