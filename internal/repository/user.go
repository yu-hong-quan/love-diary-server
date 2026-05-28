package repository

import (
	"context"
	"errors"

	"love-diary-go/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepo 用户表 users（登录校验与资料）。
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo 创建用户仓储。
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

const userSelectCols = `id, account, COALESCE(nickname,''), COALESCE(username,''), password, COALESCE(avatar,'')`

func scanUser(row pgx.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Account, &u.Nickname, &u.Username, &u.Password, &u.Avatar)
	if err != nil {
		return nil, err
	}
	if u.Account == "" {
		u.Account = u.Username
	}
	if u.Nickname == "" {
		u.Nickname = u.Account
	}
	u.Username = u.Account
	return &u, nil
}

// FindByAccount 按登录账号查询；不存在返回 nil, nil。
func (r *UserRepo) FindByAccount(ctx context.Context, account string) (*models.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+userSelectCols+` FROM users WHERE account = $1`, account)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

// FindByAccountLegacy 兼容尚未执行迁移、仅有 username 的旧表。
func (r *UserRepo) FindByAccountLegacy(ctx context.Context, account string) (*models.User, error) {
	user, err := r.FindByAccount(ctx, account)
	if err != nil || user != nil {
		return user, err
	}
	row := r.pool.QueryRow(ctx,
		`SELECT id, COALESCE(username,''), COALESCE(username,''), COALESCE(username,''), password, COALESCE(avatar,'')
		 FROM users WHERE username = $1`, account)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.Account = account
	if u.Nickname == "" {
		u.Nickname = account
	}
	return u, nil
}

// UpdateProfile 更新昵称与头像。
func (r *UserRepo) UpdateProfile(ctx context.Context, account, nickname, avatar string) (*models.User, error) {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET nickname = $2, avatar = $3 WHERE account = $1`,
		account, nickname, avatar)
	if err != nil {
		return nil, err
	}
	return r.FindByAccount(ctx, account)
}

// ToProfile 转为对外资料结构。
func ToProfile(u *models.User) models.UserProfile {
	if u == nil {
		return models.UserProfile{}
	}
	return models.UserProfile{
		ID:       u.ID,
		Account:  u.Account,
		Nickname: u.Nickname,
		Avatar:   u.Avatar,
	}
}
