package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
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
	Cat5:           4,
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
