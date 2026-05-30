package gomoku

const DefaultBoardSize = 15

// Win checks whether placing stone at (row,col) for player wins.
func Win(board [][]int, row, col, player, size int) bool {
	if player == 0 {
		return false
	}
	dirs := [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	for _, d := range dirs {
		count := 1
		count += countDir(board, row, col, d[0], d[1], player, size)
		count += countDir(board, row, col, -d[0], -d[1], player, size)
		if count >= 5 {
			return true
		}
	}
	return false
}

func countDir(board [][]int, row, col, dr, dc, player, size int) int {
	n := 0
	r, c := row+dr, col+dc
	for r >= 0 && r < size && c >= 0 && c < size && board[r][c] == player {
		n++
		r += dr
		c += dc
	}
	return n
}

// BoardFull returns true when no empty cell remains.
func BoardFull(board [][]int) bool {
	for _, row := range board {
		for _, cell := range row {
			if cell == 0 {
				return false
			}
		}
	}
	return true
}

// NewBoard creates an empty square board.
func NewBoard(size int) [][]int {
	b := make([][]int, size)
	for i := range b {
		b[i] = make([]int, size)
	}
	return b
}

// ApplyMove returns a copy with the move applied (board unchanged on invalid).
func ApplyMove(board [][]int, row, col, player int) [][]int {
	size := len(board)
	out := NewBoard(size)
	for r := 0; r < size; r++ {
		copy(out[r], board[r])
	}
	if row < 0 || row >= size || col < 0 || col >= size || out[row][col] != 0 {
		return board
	}
	out[row][col] = player
	return out
}
