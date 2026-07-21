// holds the types that pass through websocket so bot doesn't drift from game types
package wire

// information shown about each player (excludes hand)
type PlayerGameState struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	CardCount int    `json:"cardCount"`
	IsAlive   bool   `json:"isAlive"`
	IsOnline  bool   `json:"isOnline"`
}

// personalized snapshot the server pushes to one client.
type GameState struct {
	PlayerId    int               `json:"playerId"`
	LastAcked   int               `json:"lastAcked"`
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

// ActionRequest is a single move sent client -> server.
type ActionRequest struct {
	ActionStr string `json:"action"`
	SeqNumber int    `json:"seqNumber"`

	// optional fields
	PlaceKittenIndex int    `json:"placeKittenIndex"` // for placing kittens
	UseCardIndex     int    `json:"useCardIndex"`     // card that you place
	AlterFutureOrder []int  `json:"alterFutureOrder"` // new order of first 3 cards (e.g., [2, 1, 0] to reverse)
	TargetedPlayer   int    `json:"targetedPlayer"`   // player being targeted
	ComboIndices     []int  `json:"comboIndices"`     // list of cards used for combo
	RequestedCardStr string `json:"requestedCard"`    // card requested for combo
	WantNoped        bool   `json:"wantNoped"`        // for PLAY_NOPE: true = nope, false = yup
}

// action strings passed from client to server
const (
	ActionStartGame       = "START_GAME"
	ActionPlayCard        = "PLAY_CARD"
	ActionDrawCard        = "DRAW_CARD"
	ActionPlaceKitten     = "PLACE_KITTEN"
	ActionDisconnect      = "DISCONNECT"
	ActionAlterFuture     = "ALTER_FUTURE"
	ActionGiveFavor       = "GIVE_FAVOR"
	ActionCombo           = "COMBO"
	ActionTakeFromDiscard = "TAKE_FROM_DISCARD"
	ActionPlayNope        = "PLAY_NOPE"
	ActionRandomizeOrder  = "RANDOMIZE_ORDER"
	ActionRestartLobby    = "RESTART_LOBBY"
)

// card strings, mirroring Card.String() in cards.go
const (
	CardDefuse             = "DEFUSE"
	CardExplodingKitten    = "EXPLODING_KITTEN"
	CardSkip               = "SKIP"
	CardAttack             = "ATTACK"
	CardTargetedAttack     = "TARGETED_ATTACK"
	CardTacocat            = "TACOCAT"
	CardHairyPotatoCat     = "HAIRY_POTATO_CAT"
	CardCattermelon        = "CATTERMELON"
	CardRainbowRalphingCat = "RAINBOW_RALPHING_CAT"
	CardRageCat            = "RAGE_CAT"
	CardFeralCat           = "FERAL_CAT"
	CardSeeTheFuture       = "SEE_THE_FUTURE"
	CardAlterTheFuture     = "ALTER_THE_FUTURE"
	CardShuffle            = "SHUFFLE"
	CardDrawFromBottom     = "DRAW_FROM_BOTTOM"
	CardFavor              = "FAVOR"
	CardNope               = "NOPE"
)

// game states passed from server to client
const (
	TurnNotStarted              = "NOT_STARTED"
	TurnNormal                  = "NORMAL"
	TurnGameOver                = "GAME_OVER"
	TurnAwaitingKittenPlacement = "AWAITING_KITTEN_PLACEMENT"
	TurnSeeingTheFuture         = "SEEING_THE_FUTURE"
	TurnAlteringTheFuture       = "ALTERING_THE_FUTURE"
	TurnAwaitingFavor           = "AWAITING_FAVOR"
	TurnAwaitingDiscardTake     = "AWAITING_DISCARD_TAKE"
	TurnAcceptingNopes          = "ACCEPTING_NOPES"
)
