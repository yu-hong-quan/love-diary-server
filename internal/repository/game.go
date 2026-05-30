package repository

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"love-diary-go/internal/game/gomoku"
	"love-diary-go/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GameRepo 游戏模块数据访问。
type GameRepo struct {
	pool *pgxpool.Pool
}

// NewGameRepo 创建游戏仓储。
func NewGameRepo(pool *pgxpool.Pool) *GameRepo {
	return &GameRepo{pool: pool}
}

// ListGameTypes 返回支持的游戏类型。
func (r *GameRepo) ListGameTypes() []models.GameTypeInfo {
	return []models.GameTypeInfo{
		{Type: models.GameTypeGomoku, Name: "五子棋", Description: "双人对弈，先连成五子者胜"},
		{Type: models.GameTypeDice, Name: "丢骰子比大小", Description: "同屏轮流掷骰，点数大者赢得本局"},
	}
}

// ListSessions 分页查询对局记录。
func (r *GameRepo) ListSessions(ctx context.Context, gameType, status string, page, limit int) (*models.PaginatedResult[models.GameSession], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	args := []interface{}{}
	where := "WHERE 1=1"
	if gameType != "" {
		args = append(args, gameType)
		where += fmt.Sprintf(" AND game_type = $%d", len(args))
	}
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM game_sessions "+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx,
		fmt.Sprintf(`SELECT s.id, s.game_type, s.status, s.player1_name, s.player2_name, s.current_turn, s.winner,
		        s.board_size, s.player1_score, s.player2_score, s.target_wins, COALESCE(s.dice_phase,''),
		        s.player1_last_roll, s.player2_last_roll, s.created_at, s.updated_at, s.finished_at,
		        (SELECT COUNT(*)::int FROM gomoku_moves m WHERE m.session_id = s.id),
		        (SELECT COUNT(*)::int FROM dice_rounds d WHERE d.session_id = s.id)
		 FROM game_sessions s %s ORDER BY s.id DESC LIMIT $%d OFFSET $%d`, where, limitIdx, offsetIdx),
		args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data, err := scanSessionRows(rows)
	if err != nil {
		return nil, err
	}
	if data == nil {
		data = []models.GameSession{}
	}
	return &models.PaginatedResult[models.GameSession]{Data: data, Total: total, Page: page, Limit: limit}, nil
}

// ListFinishedSessions 分页查询已结束的对局（含和局）。
func (r *GameRepo) ListFinishedSessions(ctx context.Context, gameType string, page, limit int) (*models.PaginatedResult[models.GameSession], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	args := []interface{}{gameType}
	where := "WHERE game_type = $1 AND status IN ('finished', 'draw')"

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM game_sessions "+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx,
		fmt.Sprintf(`SELECT s.id, s.game_type, s.status, s.player1_name, s.player2_name, s.current_turn, s.winner,
		        s.board_size, s.player1_score, s.player2_score, s.target_wins, COALESCE(s.dice_phase,''),
		        s.player1_last_roll, s.player2_last_roll, s.created_at, s.updated_at, s.finished_at,
		        (SELECT COUNT(*)::int FROM gomoku_moves m WHERE m.session_id = s.id),
		        (SELECT COUNT(*)::int FROM dice_rounds d WHERE d.session_id = s.id)
		 FROM game_sessions s %s ORDER BY s.id DESC LIMIT $2 OFFSET $3`, where),
		args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data, err := scanSessionRows(rows)
	if err != nil {
		return nil, err
	}
	if data == nil {
		data = []models.GameSession{}
	}
	return &models.PaginatedResult[models.GameSession]{Data: data, Total: total, Page: page, Limit: limit}, nil
}

// CreateGomokuSession 新建五子棋对局。
func (r *GameRepo) CreateGomokuSession(ctx context.Context, in models.CreateGomokuSessionInput) (*models.GomokuSessionDetail, error) {
	size := in.BoardSize
	if size < 9 || size > 19 {
		size = gomoku.DefaultBoardSize
	}
	now := time.Now().UTC()
	var id int
	err := r.pool.QueryRow(ctx,
		`INSERT INTO game_sessions
		 (game_type, status, player1_name, player2_name, current_turn, board_size, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,1,$5,$6,$6) RETURNING id`,
		models.GameTypeGomoku, models.GameStatusPlaying, in.Player1Name, in.Player2Name, size, now,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetGomokuSession(ctx, id)
}

// GetGomokuSession 五子棋详情（含棋盘与棋谱）。
func (r *GameRepo) GetGomokuSession(ctx context.Context, id int) (*models.GomokuSessionDetail, error) {
	s, err := r.getSessionByID(ctx, id)
	if err != nil || s == nil {
		return nil, err
	}
	if s.GameType != models.GameTypeGomoku {
		return nil, fmt.Errorf("not a gomoku session")
	}
	moves, err := r.listGomokuMoves(ctx, id)
	if err != nil {
		return nil, err
	}
	board := gomoku.NewBoard(s.BoardSize)
	for _, m := range moves {
		board[m.Row][m.Col] = m.Player
	}
	return &models.GomokuSessionDetail{GameSession: *s, Board: board, Moves: moves}, nil
}

// PlaceGomokuMove 落子并判定胜负。
func (r *GameRepo) PlaceGomokuMove(ctx context.Context, sessionID int, row, col int) (*models.GomokuSessionDetail, error) {
	detail, err := r.GetGomokuSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, nil
	}
	if detail.Status != models.GameStatusPlaying {
		return nil, fmt.Errorf("game already finished")
	}
	if row < 0 || row >= detail.BoardSize || col < 0 || col >= detail.BoardSize {
		return nil, fmt.Errorf("position out of board")
	}
	if detail.Board[row][col] != 0 {
		return nil, fmt.Errorf("cell occupied")
	}

	player := detail.CurrentTurn
	moveNo := len(detail.Moves) + 1
	now := time.Now().UTC()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO gomoku_moves (session_id, player, row, col, move_no, created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		sessionID, player, row, col, moveNo, now)
	if err != nil {
		return nil, err
	}

	detail.Board[row][col] = player
	status := models.GameStatusPlaying
	winner := models.WinnerNone
	nextTurn := 3 - player
	var finishedAt *time.Time

	if gomoku.Win(detail.Board, row, col, player, detail.BoardSize) {
		status = models.GameStatusFinished
		winner = player
		nextTurn = player
		t := now
		finishedAt = &t
	} else if gomoku.BoardFull(detail.Board) {
		status = models.GameStatusDraw
		winner = models.WinnerDraw
		t := now
		finishedAt = &t
	}

	_, err = tx.Exec(ctx,
		`UPDATE game_sessions SET status=$2, current_turn=$3, winner=$4, updated_at=$5, finished_at=$6 WHERE id=$1`,
		sessionID, status, nextTurn, winner, now, finishedAt)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetGomokuSession(ctx, sessionID)
}

// CreateDiceSession 新建骰子对局。
func (r *GameRepo) CreateDiceSession(ctx context.Context, in models.CreateDiceSessionInput) (*models.DiceSessionDetail, error) {
	target := in.TargetWins
	if target < 1 {
		target = 1
	}
	if target > 9 {
		target = 9
	}
	now := time.Now().UTC()
	var id int
	err := r.pool.QueryRow(ctx,
		`INSERT INTO game_sessions
		 (game_type, status, player1_name, player2_name, current_turn, target_wins, dice_phase, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,1,$5,$6,$7,$7) RETURNING id`,
		models.GameTypeDice, models.GameStatusPlaying, in.Player1Name, in.Player2Name, target, models.DicePhaseRollP1, now,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetDiceSession(ctx, id)
}

// GetDiceSession 骰子对局详情。
func (r *GameRepo) GetDiceSession(ctx context.Context, id int) (*models.DiceSessionDetail, error) {
	s, err := r.getSessionByID(ctx, id)
	if err != nil || s == nil {
		return nil, err
	}
	if s.GameType != models.GameTypeDice {
		return nil, fmt.Errorf("not a dice session")
	}
	rounds, err := r.listDiceRounds(ctx, id)
	if err != nil {
		return nil, err
	}
	return &models.DiceSessionDetail{GameSession: *s, Rounds: rounds}, nil
}

// RollDice 当前玩家掷骰；P1 掷完等待 P2，P2 掷完结算本局。
func (r *GameRepo) RollDice(ctx context.Context, sessionID, player int) (*models.DiceRollResult, error) {
	detail, err := r.GetDiceSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, nil
	}
	if detail.Status != models.GameStatusPlaying {
		return nil, fmt.Errorf("game already finished")
	}
	if player != 1 && player != 2 {
		return nil, fmt.Errorf("invalid player")
	}

	roll := rand.Intn(6) + 1
	now := time.Now().UTC()
	var round *models.DiceRound
	roundDone := false

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	switch detail.DicePhase {
	case models.DicePhaseRollP1:
		if player != 1 {
			return nil, fmt.Errorf("waiting for player 1")
		}
		_, err = tx.Exec(ctx,
			`UPDATE game_sessions SET player1_last_roll=$2, dice_phase=$3, current_turn=2, updated_at=$4 WHERE id=$1`,
			sessionID, roll, models.DicePhaseRollP2, now)
		if err != nil {
			return nil, err
		}
	case models.DicePhaseRollP2:
		if player != 2 {
			return nil, fmt.Errorf("waiting for player 2")
		}
		p1Roll := 0
		if detail.Player1LastRoll != nil {
			p1Roll = *detail.Player1LastRoll
		}
		if p1Roll < 1 {
			return nil, fmt.Errorf("player 1 has not rolled")
		}

		roundNo := len(detail.Rounds) + 1
		roundWinner := models.WinnerDraw
		if roll > p1Roll {
			roundWinner = models.WinnerP2
		} else if p1Roll > roll {
			roundWinner = models.WinnerP1
		}

		var roundID int
		err = tx.QueryRow(ctx,
			`INSERT INTO dice_rounds (session_id, round_no, player1_roll, player2_roll, winner, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
			sessionID, roundNo, p1Roll, roll, roundWinner, now,
		).Scan(&roundID)
		if err != nil {
			return nil, err
		}
		round = &models.DiceRound{
			ID: roundID, SessionID: sessionID, RoundNo: roundNo,
			Player1Roll: p1Roll, Player2Roll: roll, Winner: roundWinner, CreatedAt: now,
		}
		roundDone = true

		p1Score := detail.Player1Score
		p2Score := detail.Player2Score
		if roundWinner == models.WinnerP1 {
			p1Score++
		} else if roundWinner == models.WinnerP2 {
			p2Score++
		}

		status := models.GameStatusPlaying
		winner := models.WinnerNone
		phase := models.DicePhaseRollP1
		nextTurn := 1
		var finishedAt *time.Time
		if p1Score >= detail.TargetWins || p2Score >= detail.TargetWins {
			status = models.GameStatusFinished
			if p1Score > p2Score {
				winner = models.WinnerP1
			} else if p2Score > p1Score {
				winner = models.WinnerP2
			} else {
				status = models.GameStatusDraw
				winner = models.WinnerDraw
			}
			t := now
			finishedAt = &t
			phase = ""
		}

		_, err = tx.Exec(ctx,
			`UPDATE game_sessions SET player1_score=$2, player2_score=$3, player1_last_roll=NULL, player2_last_roll=$4,
			 dice_phase=$5, current_turn=$6, status=$7, winner=$8, updated_at=$9, finished_at=$10 WHERE id=$1`,
			sessionID, p1Score, p2Score, roll, phase, nextTurn, status, winner, now, finishedAt)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid dice phase")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	updated, err := r.GetDiceSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return &models.DiceRollResult{
		GameSession: updated.GameSession,
		Roll:        roll,
		Round:       round,
		RoundDone:   roundDone,
	}, nil
}

func (r *GameRepo) getSessionByID(ctx context.Context, id int) (*models.GameSession, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT s.id, s.game_type, s.status, s.player1_name, s.player2_name, s.current_turn, s.winner,
		        s.board_size, s.player1_score, s.player2_score, s.target_wins, COALESCE(s.dice_phase,''),
		        s.player1_last_roll, s.player2_last_roll, s.created_at, s.updated_at, s.finished_at,
		        (SELECT COUNT(*)::int FROM gomoku_moves m WHERE m.session_id = s.id),
		        (SELECT COUNT(*)::int FROM dice_rounds d WHERE d.session_id = s.id)
		 FROM game_sessions s WHERE s.id = $1`, id)
	return scanSessionRow(row)
}

func (r *GameRepo) listGomokuMoves(ctx context.Context, sessionID int) ([]models.GomokuMove, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, session_id, player, row, col, move_no, created_at FROM gomoku_moves
		 WHERE session_id = $1 ORDER BY move_no ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.GomokuMove
	for rows.Next() {
		var m models.GomokuMove
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Player, &m.Row, &m.Col, &m.MoveNo, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if out == nil {
		out = []models.GomokuMove{}
	}
	return out, nil
}

func (r *GameRepo) listDiceRounds(ctx context.Context, sessionID int) ([]models.DiceRound, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, session_id, round_no, player1_roll, player2_roll, winner, created_at FROM dice_rounds
		 WHERE session_id = $1 ORDER BY round_no ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.DiceRound
	for rows.Next() {
		var d models.DiceRound
		if err := rows.Scan(&d.ID, &d.SessionID, &d.RoundNo, &d.Player1Roll, &d.Player2Roll, &d.Winner, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if out == nil {
		out = []models.DiceRound{}
	}
	return out, nil
}

func scanSessionRows(rows pgx.Rows) ([]models.GameSession, error) {
	var out []models.GameSession
	for rows.Next() {
		s, err := scanSessionFromRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func scanSessionRow(row pgx.Row) (*models.GameSession, error) {
	s, err := scanSessionFromRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func scanSessionFromRow(row pgx.Row) (models.GameSession, error) {
	var s models.GameSession
	var dicePhase string
	var p1Roll, p2Roll *int
	var finishedAt *time.Time
	err := row.Scan(
		&s.ID, &s.GameType, &s.Status, &s.Player1Name, &s.Player2Name, &s.CurrentTurn, &s.Winner,
		&s.BoardSize, &s.Player1Score, &s.Player2Score, &s.TargetWins, &dicePhase,
		&p1Roll, &p2Roll, &s.CreatedAt, &s.UpdatedAt, &finishedAt,
		&s.MoveCount, &s.RoundCount,
	)
	if err != nil {
		return s, err
	}
	s.DicePhase = dicePhase
	s.Player1LastRoll = p1Roll
	s.Player2LastRoll = p2Roll
	s.FinishedAt = finishedAt
	return s, nil
}
