// Package database 负责 PostgreSQL 连接池的创建与探活。
package database

import (
	"context"
	"fmt"

	"love-diary-go/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect 根据配置建立连接池，并在返回前 Ping 一次确保可用。
func Connect(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
