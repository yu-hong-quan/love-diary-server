package repository

import (
	"context"
	"errors"
	"time"

	"love-diary-go/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CommentRepo 日记评论仓储。
type CommentRepo struct {
	pool       *pgxpool.Pool
	table      string
	diaryTable string
	diaryRepo  *DiaryRepo
}

// NewTravelCommentRepo 旅行日记评论表 travel_diary_comments。
func NewTravelCommentRepo(pool *pgxpool.Pool, diaryRepo *DiaryRepo) *CommentRepo {
	return &CommentRepo{
		pool:       pool,
		table:      "travel_diary_comments",
		diaryTable: "travel_diaries",
		diaryRepo:  diaryRepo,
	}
}

// NewDailyCommentRepo 日常日记评论表 daily_diary_comments。
func NewDailyCommentRepo(pool *pgxpool.Pool, diaryRepo *DiaryRepo) *CommentRepo {
	return &CommentRepo{
		pool:       pool,
		table:      "daily_diary_comments",
		diaryTable: "daily_diaries",
		diaryRepo:  diaryRepo,
	}
}

// ListByDiaryID 查询某篇日记的全部评论（按时间正序）。
func (r *CommentRepo) ListByDiaryID(ctx context.Context, diaryID int) ([]models.Comment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.diary_id, c.content, c.username,
		        COALESCE(NULLIF(u.nickname, ''), u.account, c.username),
		        COALESCE(u.avatar,''), c.created_at
		 FROM `+r.table+` c
		 LEFT JOIN users u ON u.account = c.username
		 WHERE c.diary_id = $1
		 ORDER BY c.created_at ASC, c.id ASC`, diaryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Comment
	for rows.Next() {
		var c models.Comment
		var createdAt time.Time
		if err := rows.Scan(&c.ID, &c.DiaryID, &c.Content, &c.Username, &c.Name, &c.Avatar, &createdAt); err != nil {
			return nil, err
		}
		if c.Name == "" {
			c.Name = c.Username
		}
		c.CreatedAt = createdAt
		list = append(list, c)
	}
	if list == nil {
		list = []models.Comment{}
	}
	return list, nil
}

// NextID 评论业务 id：MAX(id)+1。
func (r *CommentRepo) NextID(ctx context.Context) (int, error) {
	var id int
	err := r.pool.QueryRow(ctx, "SELECT COALESCE(MAX(id), 0) + 1 FROM "+r.table).Scan(&id)
	return id, err
}

// Create 发表评论并同步日记评论计数。
func (r *CommentRepo) Create(ctx context.Context, diaryID int, username, content string) (*models.Comment, error) {
	diary, err := r.diaryRepo.GetByID(ctx, diaryID)
	if err != nil {
		return nil, err
	}
	if diary == nil {
		return nil, nil
	}

	id, err := r.NextID(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()

	_, err = r.pool.Exec(ctx,
		`INSERT INTO `+r.table+` (id, diary_id, username, content, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		id, diaryID, username, content, now)
	if err != nil {
		return nil, err
	}
	if err := r.diaryRepo.IncrementComments(ctx, diaryID); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

// GetByID 按评论 id 查询。
func (r *CommentRepo) GetByID(ctx context.Context, id int) (*models.Comment, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT c.id, c.diary_id, c.content, c.username,
		        COALESCE(NULLIF(u.nickname, ''), u.account, c.username),
		        COALESCE(u.avatar,''), c.created_at
		 FROM `+r.table+` c
		 LEFT JOIN users u ON u.account = c.username
		 WHERE c.id = $1`, id)
	var c models.Comment
	var createdAt time.Time
	err := row.Scan(&c.ID, &c.DiaryID, &c.Content, &c.Username, &c.Name, &c.Avatar, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if c.Name == "" {
		c.Name = c.Username
	}
	c.CreatedAt = createdAt
	return &c, nil
}

// Delete 删除评论（仅当 diary_id 与 username 匹配时），并同步评论计数。
func (r *CommentRepo) Delete(ctx context.Context, diaryID, commentID int, username string) (bool, error) {
	existing, err := r.GetByID(ctx, commentID)
	if err != nil {
		return false, err
	}
	if existing == nil || existing.DiaryID != diaryID || existing.Username != username {
		return false, nil
	}

	tag, err := r.pool.Exec(ctx,
		`DELETE FROM `+r.table+` WHERE id=$1 AND diary_id=$2 AND username=$3`,
		commentID, diaryID, username)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if err := r.diaryRepo.DecrementComments(ctx, diaryID); err != nil {
		return false, err
	}
	return true, nil
}
