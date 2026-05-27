package repository

import (
	"context"
	"errors"
	"time"

	"love-diary-go/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WhisperRepo 悄悄话表 whispers。
type WhisperRepo struct {
	pool *pgxpool.Pool
}

// NewWhisperRepo 创建悄悄话仓储。
func NewWhisperRepo(pool *pgxpool.Pool) *WhisperRepo {
	return &WhisperRepo{pool: pool}
}

// List 分页列表，id 倒序。
func (r *WhisperRepo) List(ctx context.Context, page, limit int) (*models.PaginatedResult[models.Whisper], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM whispers").Scan(&total); err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, content, date, likes, created_at, updated_at FROM whispers ORDER BY id DESC LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []models.Whisper
	for rows.Next() {
		w, err := scanWhisper(rows)
		if err != nil {
			return nil, err
		}
		data = append(data, w)
	}
	if data == nil {
		data = []models.Whisper{}
	}
	return &models.PaginatedResult[models.Whisper]{Data: data, Total: total, Page: page, Limit: limit}, nil
}

// GetByID 按 id 查询。
func (r *WhisperRepo) GetByID(ctx context.Context, id int) (*models.Whisper, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, content, date, likes, created_at, updated_at FROM whispers WHERE id = $1`, id)
	return scanWhisperRow(row)
}

// NextID 下一个业务 id。
func (r *WhisperRepo) NextID(ctx context.Context) (int, error) {
	var id int
	err := r.pool.QueryRow(ctx, "SELECT COALESCE(MAX(id), 0) + 1 FROM whispers").Scan(&id)
	return id, err
}

// Create 新建悄悄话。
func (r *WhisperRepo) Create(ctx context.Context, in models.WhisperInput) (*models.Whisper, error) {
	id, err := r.NextID(ctx)
	if err != nil {
		return nil, err
	}
	likes := in.Likes
	now := time.Now().UTC()
	_, err = r.pool.Exec(ctx,
		`INSERT INTO whispers (id, content, date, likes, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$5)`,
		id, in.Content, in.Date, likes, now)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

// Update 更新；空字段保留原值。
func (r *WhisperRepo) Update(ctx context.Context, id int, in models.WhisperInput) (*models.Whisper, error) {
	existing, err := r.GetByID(ctx, id)
	if err != nil || existing == nil {
		return nil, err
	}
	content := in.Content
	if content == "" {
		content = existing.Content
	}
	date := in.Date
	if date == "" {
		date = existing.Date
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE whispers SET content=$2, date=$3, updated_at=$4 WHERE id=$1`,
		id, content, date, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

// IncrementLikes 点赞 +1。
func (r *WhisperRepo) IncrementLikes(ctx context.Context, id int) (*models.Whisper, error) {
	w, err := r.GetByID(ctx, id)
	if err != nil || w == nil {
		return nil, err
	}
	_, err = r.pool.Exec(ctx, `UPDATE whispers SET likes=$2, updated_at=$3 WHERE id=$1`, id, w.Likes+1, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

// Delete 删除记录。
func (r *WhisperRepo) Delete(ctx context.Context, id int) (bool, error) {
	tag, err := r.pool.Exec(ctx, "DELETE FROM whispers WHERE id=$1", id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func scanWhisper(rows pgx.Rows) (models.Whisper, error) {
	var w models.Whisper
	var createdAt, updatedAt time.Time
	err := rows.Scan(&w.ID, &w.Content, &w.Date, &w.Likes, &createdAt, &updatedAt)
	w.CreatedAt = createdAt
	w.UpdatedAt = updatedAt
	return w, err
}

func scanWhisperRow(row pgx.Row) (*models.Whisper, error) {
	var w models.Whisper
	var createdAt, updatedAt time.Time
	err := row.Scan(&w.ID, &w.Content, &w.Date, &w.Likes, &createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	w.CreatedAt = createdAt
	w.UpdatedAt = updatedAt
	return &w, nil
}
