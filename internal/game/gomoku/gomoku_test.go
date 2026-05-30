package gomoku_test

import (
	"testing"

	"love-diary-go/internal/game/gomoku"
)

func TestWinHorizontal(t *testing.T) {
	b := gomoku.NewBoard(15)
	for c := 0; c < 4; c++ {
		b[7][c] = 1
	}
	b[7][4] = 1
	if !gomoku.Win(b, 7, 4, 1, 15) {
		t.Fatal("expected win")
	}
}

func TestBoardFull(t *testing.T) {
	b := gomoku.NewBoard(3)
	n := 0
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			n++
			b[r][c] = (n % 2) + 1
		}
	}
	if !gomoku.BoardFull(b) {
		t.Fatal("expected full board")
	}
}
