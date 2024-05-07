// Package game implements a game of Tic-Tac-Toe.
package game

import (
	"fmt"
	"strings"
)

// Player represents a player.
type Player uint8

const (
	NoPlayer Player = iota
	Player1
	Player2
)

// String returns the string representation of the player.
func (p Player) String() string {
	switch p {
	case Player1:
		return "X"
	case Player2:
		return "O"
	default:
		return " "
	}
}

// Opponent returns the opponent of the player.
func (p Player) Opponent() Player {
	switch p {
	case Player1:
		return Player2
	case Player2:
		return Player1
	default:
		return NoPlayer
	}
}

// BoardPosition represents a position on the board.
type BoardPosition struct {
	X, Y uint8
}

// BoardPositionAt returns a board position at the given coordinates.
// If the coordinates are invalid, returns an error.
func BoardPositionAt(x, y int) (BoardPosition, error) {
	if x < 0 || x > 2 || y < 0 || y > 2 {
		return BoardPosition{}, fmt.Errorf("invalid position: (%d, %d)", x, y)
	}
	return BoardPosition{X: uint8(x), Y: uint8(y)}, nil
}

// BoardPositionFromIndex returns a board position from the given index.
func BoardPositionFromIndex(i int) (BoardPosition, error) {
	if i < 0 || i > 8 {
		return BoardPosition{}, fmt.Errorf("invalid index: %d", i)
	}
	return BoardPosition{X: uint8(i / 3), Y: uint8(i % 3)}, nil
}

// IsValid returns true if the position is valid.
func (p BoardPosition) IsValid() bool {
	return p.X < 3 && p.Y < 3
}

// Board represents a Tic-Tac-Toe board.
type Board [3][3]Player

func (b Board) String() string {
	var s strings.Builder
	s.WriteByte('[')
	for r := range 3 {
		if r > 0 {
			s.WriteByte(' ')
		}

		s.WriteByte('[')
		for c := range 2 {
			s.WriteString(b[r][c].String())
			s.WriteByte(' ')
		}
		s.WriteString(b[r][2].String())
		s.WriteString("]")

		if r < 2 {
			s.WriteString("\n")
		} else {
			s.WriteByte(']')
		}
	}
	return s.String()
}

// At returns the player at the given position.
func (b Board) At(pos BoardPosition) Player {
	if !pos.IsValid() {
		return NoPlayer
	}
	return b[pos.X][pos.Y]
}

// PlacePiece places a player at the given position.
// If the position is invalid, returns false.
func (b *Board) PlacePiece(p Player, pos BoardPosition) bool {
	if !pos.IsValid() || b.At(pos) != NoPlayer {
		return false
	}
	b[pos.X][pos.Y] = p
	return true
}

// HasEnded is a convenience method around [GameState] that returns true if the
// game has ended.
func (b Board) HasEnded() bool {
	_, ended := b.GameState()
	return ended
}

// GameState returns the state of the game.
// If the game is over, returns the winner and true or NoPlayer and true if it's
// a draw.
// Otherwise, returns NoPlayer and false.
func (b Board) GameState() (winner Player, ended bool) {
	// Check rows and columns.
	for i := range 3 {
		if b[i][0] == b[i][1] && b[i][1] == b[i][2] && b[i][0] != NoPlayer {
			return b[i][0], true
		}
		if b[0][i] == b[1][i] && b[1][i] == b[2][i] && b[0][i] != NoPlayer {
			return b[0][i], true
		}
	}
	// Check diagonals.
	if b[0][0] == b[1][1] && b[1][1] == b[2][2] && b[0][0] != NoPlayer {
		return b[0][0], true
	}
	if b[0][2] == b[1][1] && b[1][1] == b[2][0] && b[0][2] != NoPlayer {
		return b[0][2], true
	}
	for i := range 3 {
		for j := range 3 {
			if b[i][j] == NoPlayer {
				return NoPlayer, false
			}
		}
	}
	return NoPlayer, true
}

// Game represents a game of Tic-Tac-Toe.
type Game struct {
	Board
	Turns int
}

// NewGame creates a new game of Tic-Tac-Toe.
func NewGame() *Game {
	return &Game{
		Board: Board{},
	}
}

func (g *Game) String() string {
	return fmt.Sprintf("turn %d:\n%s", g.Turns, g.Board)
}

// Turn returns the current player.
func (g *Game) Turn() Player {
	if g.Turns%2 == 0 {
		return Player1
	}
	return Player2
}

// MakeMove makes a move for the current player at the given position.
func (g *Game) MakeMove(pos BoardPosition) bool {
	if g.Board.PlacePiece(g.Turn(), pos) {
		g.Turns++
		return true
	}
	return false
}

// Clone creates a deep copy of the game.
func (g *Game) Clone() *Game {
	g2 := *g
	return &g2
}
