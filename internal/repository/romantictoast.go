package repository

import (
	"context"
	"errors"

	"love-diary-go/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RomanticToastRepo 首页浪漫 toast 文案表 romantic_toasts。
type RomanticToastRepo struct {
	pool *pgxpool.Pool
}

// NewRomanticToastRepo 创建浪漫文案仓储。
func NewRomanticToastRepo(pool *pgxpool.Pool) *RomanticToastRepo {
	return &RomanticToastRepo{pool: pool}
}

// Random 随机返回一条文案。
func (r *RomanticToastRepo) Random(ctx context.Context) (*models.RomanticToast, error) {
	var t models.RomanticToast
	err := r.pool.QueryRow(ctx,
		`SELECT id, content FROM romantic_toasts ORDER BY RANDOM() LIMIT 1`,
	).Scan(&t.ID, &t.Content)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}
