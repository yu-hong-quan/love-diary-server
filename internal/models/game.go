package models

import "time"

const (
	GameTypeGomoku = "gomoku"
	GameTypeDice   = "dice"

	GameStatusPlaying  = "playing"
	GameStatusFinished = "finished"
	GameStatusDraw     = "draw"

	DicePhaseRollP1 = "roll_p1"
	DicePhaseRollP2 = "roll_p2"

	WinnerNone = 0
	WinnerP1   = 1
	WinnerP2   = 2
	WinnerDraw = 3
)

// GameTypeInfo 游戏类型元数据。
type GameTypeInfo struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GameSession 对局会话（五子棋 / 骰子共用）。
type GameSession struct {
	ID              int        `json:"id"`
	GameType        string     `json:"gameType"`
	Status          string     `json:"status"`
	Player1Name     string     `json:"player1Name"`
	Player2Name     string     `json:"player2Name"`
	CurrentTurn     int        `json:"currentTurn"`
	Winner          int        `json:"winner"`
	BoardSize       int        `json:"boardSize,omitempty"`
	Player1Score    int        `json:"player1Score,omitempty"`
	Player2Score    int        `json:"player2Score,omitempty"`
	TargetWins      int        `json:"targetWins,omitempty"`
	DicePhase       string     `json:"dicePhase,omitempty"`
	Player1LastRoll *int       `json:"player1LastRoll,omitempty"`
	Player2LastRoll *int       `json:"player2LastRoll,omitempty"`
	MoveCount       int        `json:"moveCount,omitempty"`
	RoundCount      int        `json:"roundCount,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	FinishedAt      *time.Time `json:"finishedAt,omitempty"`
}

// GomokuMove 五子棋一步棋。
type GomokuMove struct {
	ID        int       `json:"id"`
	SessionID int       `json:"sessionId"`
	Player    int       `json:"player"`
	Row       int       `json:"row"`
	Col       int       `json:"col"`
	MoveNo    int       `json:"moveNo"`
	CreatedAt time.Time `json:"createdAt"`
}

// DiceRound 骰子一局比大小记录。
type DiceRound struct {
	ID          int       `json:"id"`
	SessionID   int       `json:"sessionId"`
	RoundNo     int       `json:"roundNo"`
	Player1Roll int       `json:"player1Roll"`
	Player2Roll int       `json:"player2Roll"`
	Winner      int       `json:"winner"`
	CreatedAt   time.Time `json:"createdAt"`
}

// GomokuSessionDetail 五子棋对局详情。
type GomokuSessionDetail struct {
	GameSession
	Board [][]int      `json:"board"`
	Moves []GomokuMove `json:"moves"`
}

// DiceSessionDetail 骰子对局详情。
type DiceSessionDetail struct {
	GameSession
	Rounds []DiceRound `json:"rounds"`
}

// CreateGomokuSessionInput 创建五子棋。
type CreateGomokuSessionInput struct {
	Player1Name string `json:"player1Name"`
	Player2Name string `json:"player2Name"`
	BoardSize   int    `json:"boardSize"`
}

// GomokuMoveInput 落子。
type GomokuMoveInput struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// CreateDiceSessionInput 创建骰子对局。
type CreateDiceSessionInput struct {
	Player1Name string `json:"player1Name"`
	Player2Name string `json:"player2Name"`
	TargetWins  int    `json:"targetWins"`
}

// DiceRollInput 掷骰（同屏轮流操作）。
type DiceRollInput struct {
	Player int `json:"player"` // 1 或 2
}

// DiceRollResult 掷骰响应。
type DiceRollResult struct {
	GameSession
	Roll      int        `json:"roll"`
	Round     *DiceRound `json:"round,omitempty"`
	RoundDone bool       `json:"roundDone"`
}
