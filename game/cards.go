package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"
)

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
	Cat5
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
	case Cat5:
		return "CAT5"
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

type TieredCardCount struct {
	Small  int // 2-3 players
	Medium int // 4-6 players
	Large  int //7-10 players
}

var ActionCardTotals = map[Card]TieredCardCount{
	// The 5 standard cat cards (Tacocat, Hairy Potato, Cattermelon, Rainbow, Beard)
	Cat1: {Small: 3, Medium: 4, Large: 7},
	Cat2: {Small: 3, Medium: 4, Large: 7},
	Cat3: {Small: 3, Medium: 4, Large: 7},
	Cat4: {Small: 3, Medium: 4, Large: 7},
	Cat5: {Small: 3, Medium: 4, Large: 7},

	// Special Action Cards
	FeralCat:       {Small: 2, Medium: 4, Large: 6},
	Skip:           {Small: 4, Medium: 6, Large: 10},
	SeeTheFuture:   {Small: 3, Medium: 3, Large: 6},
	AlterTheFuture: {Small: 2, Medium: 4, Large: 6},
	Attack:         {Small: 2, Medium: 3, Large: 5},
	TargetedAttack: {Small: 2, Medium: 3, Large: 5},
	Shuffle:        {Small: 2, Medium: 4, Large: 6},
	DrawFromBottom: {Small: 3, Medium: 4, Large: 7},
	Favor:          {Small: 2, Medium: 4, Large: 6},
	// Nope:           {Small: 4, Medium: 6, Large: 10},
}

var DefuseTotals = TieredCardCount{Small: 3, Medium: 7, Large: 10}

// returns a map of cards based on number of players
func GetDeckConfig(numPlayers int) map[Card]int {
	deck := make(map[Card]int)

	for card, tiers := range ActionCardTotals {
		var count int

		switch {
		case numPlayers <= 3:
			count = tiers.Small
		case numPlayers <= 6:
			count = tiers.Medium
		default: // 7-10 players
			count = tiers.Large
		}

		deck[card] = count
	}

	return deck
}

func GetExtraDefuses(numPlayers int) int {
	switch {
	case numPlayers <= 3:
		return DefuseTotals.Small - numPlayers
	case numPlayers <= 6:
		return DefuseTotals.Medium - numPlayers
	default: // 7-10 players
		return DefuseTotals.Medium - numPlayers
	}
}

func ParseCard(s string) (Card, error) {
	if card, ok := cardNames[s]; ok {
		return card, nil
	}
	return 0, errors.New("invalid card: " + s)
}

func (c Card) isCat() bool {
	return c == Cat1 || c == Cat2 || c == Cat3 || c == Cat4 || c == Cat5 || c == FeralCat
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

// --- Deck Operations ---

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
	drawn := lobby.deck[0]
	lobby.deck = lobby.deck[1:]
	return drawn
}

func (lobby *Lobby) removeBottomCard() Card {
	if len(lobby.deck) == 0 {
		fmt.Println("The deck is empty!")
		os.Exit(1)
	}
	lastIdx := len(lobby.deck) - 1
	drawn := lobby.deck[lastIdx]
	lobby.deck = lobby.deck[:lastIdx]
	return drawn
}

// --- Card Helpers ---

func cardSliceToStrings(cards []Card) []string {
	res := make([]string, len(cards))
	for i, card := range cards {
		res[i] = card.String()
	}
	return res
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
