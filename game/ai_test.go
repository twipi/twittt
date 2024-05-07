package game

import (
	"fmt"
	"testing"
)

func TestAI(t *testing.T) {
	for x := range uint8(3) {
		for y := range uint8(3) {
			t.Run(fmt.Sprintf("start(%d,%d)", x, y), func(t *testing.T) {
				g := NewGame()

				g.MakeMove(BoardPosition{x, y})
				t.Log(g)

				ps := []*AI{NewAI(g, Player1), NewAI(g, Player2)}
				for ps[g.Turns%2].MakeMove() {
					t.Log(g)
				}

				if winner, _ := g.GameState(); winner != NoPlayer {
					t.Errorf("game should always end in a draw, got winner %v", winner)
				}
			})
		}
	}
}
