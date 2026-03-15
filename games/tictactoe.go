package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Global 3x3 board
var board [3][3]string

// Initialize the board with empty spaces
func initBoard() {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			board[i][j] = " "
		}
	}
}

// Print the board to the terminal
func printBoard() {
	fmt.Println("\n  0 1 2")
	for i := 0; i < 3; i++ {
		fmt.Printf("%d ", i)
		for j := 0; j < 3; j++ {
			fmt.Printf("%s", board[i][j])
			if j < 2 {
				fmt.Print("|")
			}
		}
		fmt.Println()
		if i < 2 {
			fmt.Println("  -+-+-")
		}
	}
	fmt.Println()
}

// Check if a specific player has won
func checkWin(player string) bool {
	// Check rows and columns
	for i := 0; i < 3; i++ {
		if board[i][0] == player && board[i][1] == player && board[i][2] == player {
			return true
		}
		if board[0][i] == player && board[1][i] == player && board[2][i] == player {
			return true
		}
	}
	// Check diagonals
	if board[0][0] == player && board[1][1] == player && board[2][2] == player {
		return true
	}
	if board[0][2] == player && board[1][1] == player && board[2][0] == player {
		return true
	}
	return false
}

// Check if the board is completely full
func checkDraw() bool {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if board[i][j] == " " {
				return false
			}
		}
	}
	return true
}

func main() {
	initBoard()
	currentPlayer := "X"
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Welcome to Tic-Tac-Toe in Go!")

	for {
		printBoard()
		fmt.Printf("Player %s, enter row and column (e.g., '0 2'): ", currentPlayer)

		scanner.Scan()
		input := scanner.Text()
		parts := strings.Fields(input)

		// Validate that exactly two inputs were provided
		if len(parts) != 2 {
			fmt.Println("-> Invalid input. Please enter row and column separated by a space.")
			continue
		}

		// Convert inputs to integers
		row, err1 := strconv.Atoi(parts[0])
		col, err2 := strconv.Atoi(parts[1])

		// Validate coordinates
		if err1 != nil || err2 != nil || row < 0 || row > 2 || col < 0 || col > 2 {
			fmt.Println("-> Invalid input. Row and column must be numbers between 0 and 2.")
			continue
		}

		// Check if the cell is available
		if board[row][col] != " " {
			fmt.Println("-> That space is already taken. Try again.")
			continue
		}

		// Place the marker
		board[row][col] = currentPlayer

		// Check for a win
		if checkWin(currentPlayer) {
			printBoard()
			fmt.Printf("🎉 Player %s wins! 🎉\n", currentPlayer)
			break
		}

		// Check for a draw
		if checkDraw() {
			printBoard()
			fmt.Println("It's a draw!")
			break
		}

		// Switch player turn
		if currentPlayer == "X" {
			currentPlayer = "O"
		} else {
			currentPlayer = "X"
		}
	}
}
