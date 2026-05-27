package repository

import (
	"context"
	"errors"

	"love-diary-go/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepo 用户表 users（登录校验）。
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo 创建用户仓储。
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// FindByUsername 按登录名查询；不存在返回 nil, nil。
func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, username, password, COALESCE(avatar,'') FROM users WHERE username = $1`, username)
	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Password, &u.Avatar)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
