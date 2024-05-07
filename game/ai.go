package game

import (
	"math"
)

// AI represents an AI player.
// The AI is implemented using the minimax algorithm.
type AI struct {
	game   *Game
	player Player
}

// NewAI creates a new AI player.
func NewAI(g *Game, p Player) *AI {
	return &AI{game: g, player: p}
}

// NextMove returns the next move that the AI should make.
// If the game is over or the AI can't make a move, return false.
func (a *AI) NextMove() (BoardPosition, bool) {
	if a.game.Turn() != a.player {
		return BoardPosition{}, false
	}
	_, pos, ended := minimax(a.game, a.player)
	return pos, ended
}

// MakeMove makes the next move for the AI.
// Returns true if the move was made successfully.
func (a *AI) MakeMove() bool {
	pos, ended := a.NextMove()
	if ended {
		return false
	}
	return a.game.MakeMove(pos)
}

func possibleMoves(g *Game) func(func(BoardPosition) bool) {
	return func(yield func(BoardPosition) bool) {
		for i := range uint8(3) {
			for j := range uint8(3) {
				if g.Board[i][j] == NoPlayer {
					yield(BoardPosition{i, j})
				}
			}
		}
	}
}

func minimax(g *Game, self Player) (value int, move BoardPosition, ended bool) {
	if winner, ended := g.GameState(); ended {
		if winner == self {
			return 1, BoardPosition{}, true
		}
		if winner == self.Opponent() {
			return -1, BoardPosition{}, true
		}
		return 0, BoardPosition{}, true
	}

	var bestValue int
	var bestMove BoardPosition

	if self == g.Turn() {
		bestValue = math.MinInt

		moves := possibleMoves(g)
		moves(func(pos BoardPosition) bool {
			g2 := g.Clone()
			g2.MakeMove(pos)

			value, _, _ := minimax(g2, self)
			if value > bestValue {
				bestValue = value
				bestMove = pos
			}

			return true
		})
	} else {
		bestValue = math.MaxInt

		moves := possibleMoves(g)
		moves(func(pos BoardPosition) bool {
			g2 := g.Clone()
			g2.MakeMove(pos)

			value, _, _ := minimax(g2, self)
			if value < bestValue {
				bestValue = value
				bestMove = pos
			}

			return true
		})
	}

	return bestValue, bestMove, false
}
