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
	Name    string
	Hand    []Card
	IsAlive bool
}

// Global state
var (
	deck    []Card
	players []*Player
	scanner = bufio.NewScanner(os.Stdin)
	rng     = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// --- Helper Functions ---

func shuffleDeck() {
	rng.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
}

func drawCard() Card {
	if len(deck) == 0 {
		fmt.Println("The deck is empty! (This shouldn't happen with correct kitten math)")
		os.Exit(1)
	}
	// Pop the top card
	drawn := deck[0]
	deck = deck[1:]
	return drawn
}

func getActivePlayerCount() int {
	count := 0
	for _, p := range players {
		if p.IsAlive {
			count++
		}
	}
	return count
}

// --- Setup & Game Loop ---

func setupGame() {
	fmt.Print("Enter the number of players: ")
	scanner.Scan()
	numPlayers, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || numPlayers < 2 {
		fmt.Println("Invalid input. Defaulting to 2 players.")
		numPlayers = 2
	}

	// Create players
	for i := 1; i <= numPlayers; i++ {
		players = append(players, &Player{
			Name:    fmt.Sprintf("Player %d", i),
			Hand:    []Card{},
			IsAlive: true,
		})
	}

	// Create a pool of safe cards (Lots of Cats, some Skips)
	var safeDeck []Card
	for i := 0; i < numPlayers*CatMultiplier; i++ {
		safeDeck = append(safeDeck, Cat)
	}
	for i := 0; i < numPlayers*SkipMultipler; i++ {
		safeDeck = append(safeDeck, Skip)
	}

	// Put safe cards in the main deck and shuffle
	deck = safeDeck
	shuffleDeck()

	// Deal 1 diffuse + 4 starting cards to each player
	for _, p := range players {
		p.Hand = append(p.Hand, Defuse)
		for i := 0; i < 4; i++ {
			p.Hand = append(p.Hand, drawCard())
		}
	}

	// Insert (Players - 1) Exploding Kittens into the remaining deck
	for i := 0; i < numPlayers-1; i++ {
		deck = append(deck, ExplodingKitten)
	}

	for i := 0; i < ExtraDefuses; i++ {
		deck = append(deck, Defuse)
	}

	// Final shuffle
	shuffleDeck()
	fmt.Printf("\n--- Game Setup Complete! Deck has %d cards. ---\n", len(deck))
}

func playTurn(p *Player) {
	fmt.Printf("\n==============================\n")
	fmt.Printf("It is %s's turn!\n", p.Name)

	for {
		fmt.Println("\nYour Hand:")
		if len(p.Hand) == 0 {
			fmt.Println("  (Empty)")
		} else {
			for i, card := range p.Hand {
				fmt.Printf("  [%d] %s\n", i, card)
			}
		}
		fmt.Println("  [D] Draw a card (Ends Turn)")

		fmt.Print("\nChoose a card to play (number) or type 'D' to draw: ")
		scanner.Scan()
		input := strings.ToUpper(strings.TrimSpace(scanner.Text()))

		// Handle Drawing
		if input == "D" {
			drawn := drawCard()
			fmt.Printf("\n>>> You drew: %s <<<\n", drawn)

			if drawn == ExplodingKitten {
				if defuseIndex := slices.Index(p.Hand, Defuse); defuseIndex != -1 {
					p.Hand = append(p.Hand[:defuseIndex], p.Hand[defuseIndex+1:]...)
					fmt.Printf("✅ %s used a defuse card! Crisis averted!\n", p.Name)

					fmt.Printf("Enter the new kitten position (0-%d)", len(deck))
					scanner.Scan()
					newKittenPosition, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
					if err != nil || newKittenPosition < 0 || newKittenPosition > len(deck) {
						fmt.Println("Invalid input. Putting kitten on top")
						newKittenPosition = 0
					}
					deck = slices.Insert(deck, newKittenPosition, ExplodingKitten)
				} else {
					p.IsAlive = false
					fmt.Printf("💀 %s has no defuse card and is eliminated!\n", p.Name)
				}
			} else {
				p.Hand = append(p.Hand, drawn)
				fmt.Println("Card added to your hand.")
			}
			break // Turn ends after drawing
		}

		// Handle Playing a Card
		idx, err := strconv.Atoi(input)
		if err != nil || idx < 0 || idx >= len(p.Hand) {
			fmt.Println("❌ Invalid choice. Please enter a valid number or 'D'.")
			continue
		}

		playedCard := p.Hand[idx]

		success := true
		endTurn := false

		switch playedCard {
		case Skip:
			fmt.Println("⏭️  You skipped your draw phase! Turn ends.")
			endTurn = true
		case Cat:
			fmt.Println("🚫  You cannot play a cat card!")
			success = false
		case Defuse:
			fmt.Println("🚫  You cannot play a defuse card!")
			success = false
		}

		if success {
			p.Hand = slices.Delete(p.Hand, idx, idx+1)
		}

		if endTurn {
			break
		}
	}
}

func main() {
	fmt.Println("🧨 Welcome to CLI Exploding Kittens (Sudden Death Edition) 🧨")
	setupGame()

	currentPlayerIndex := 0

	// Game loop continues until only 1 player is left
	for getActivePlayerCount() > 1 {
		p := players[currentPlayerIndex]

		// Only alive players get to take a turn
		if p.IsAlive {
			playTurn(p)
			time.Sleep(1 * time.Second) // Small pause for readability in the terminal
		}

		// Move to the next player, wrapping around the slice using modulo
		currentPlayerIndex = (currentPlayerIndex + 1) % len(players)
	}

	// Find and announce the winner
	for _, p := range players {
		if p.IsAlive {
			fmt.Printf("\n🎉🎉🎉 %s SURVIVED AND WINS THE GAME! 🎉🎉🎉\n", p.Name)
			break
		}
	}
}
