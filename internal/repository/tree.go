package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"love-diary-go/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TreeRepo 爱情树表 trees（通常仅一条记录）。
type TreeRepo struct {
	pool *pgxpool.Pool
}

// NewTreeRepo 创建爱情树仓储。
func NewTreeRepo(pool *pgxpool.Pool) *TreeRepo {
	return &TreeRepo{pool: pool}
}

// GetFirst 取 id 最小的一棵树（单用户场景只有一棵）。
func (r *TreeRepo) GetFirst(ctx context.Context) (*models.Tree, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, start_date, COALESCE(last_watered,''), water_count, logs, created_at, updated_at
		 FROM trees ORDER BY id ASC LIMIT 1`)
	t, err := scanTreeRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

// CreateDefault 首次访问时创建默认树：startDate = lastWatered = 当天，waterCount = 0。
func (r *TreeRepo) CreateDefault(ctx context.Context, startDate string) (*models.Tree, error) {
	var id int
	if err := r.pool.QueryRow(ctx, "SELECT COALESCE(MAX(id), 0) + 1 FROM trees").Scan(&id); err != nil {
		return nil, err
	}
	logs := []models.TreeLog{}
	logsJSON, _ := json.Marshal(logs)
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO trees (id, start_date, last_watered, water_count, logs, created_at, updated_at)
		 VALUES ($1,$2,$2,$3,$4::jsonb,$5,$5)`,
		id, startDate, 0, logsJSON, now)
	if err != nil {
		return nil, err
	}
	return r.GetFirst(ctx)
}

// Water 更新最近浇水日与累计浇水次数。
func (r *TreeRepo) Water(ctx context.Context, id int, lastWatered string, waterCount int) (*models.Tree, error) {
	_, err := r.pool.Exec(ctx,
		`UPDATE trees SET last_watered=$2, water_count=$3, updated_at=$4 WHERE id=$1`,
		id, lastWatered, waterCount, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

// GetByID 按 id 查询。
func (r *TreeRepo) GetByID(ctx context.Context, id int) (*models.Tree, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, start_date, COALESCE(last_watered,''), water_count, logs, created_at, updated_at
		 FROM trees WHERE id = $1`, id)
	return scanTreeRow(row)
}

func scanTreeRow(row pgx.Row) (*models.Tree, error) {
	var t models.Tree
	var logsJSON []byte
	var createdAt, updatedAt time.Time
	err := row.Scan(&t.ID, &t.StartDate, &t.LastWatered, &t.WaterCount, &logsJSON, &createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(logsJSON, &t.Logs)
	if t.Logs == nil {
		t.Logs = []models.TreeLog{}
	}
	t.CreatedAt = createdAt
	t.UpdatedAt = updatedAt
	return &t, nil
}
