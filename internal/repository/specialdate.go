package repository

import (
	"context"
	"errors"
	"time"

	"love-diary-go/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SpecialDateRepo 特殊日期表 special_dates。
type SpecialDateRepo struct {
	pool *pgxpool.Pool
}

// NewSpecialDateRepo 创建特殊日期仓储。
func NewSpecialDateRepo(pool *pgxpool.Pool) *SpecialDateRepo {
	return &SpecialDateRepo{pool: pool}
}

// List 分页列表。
func (r *SpecialDateRepo) List(ctx context.Context, page, limit int) (*models.PaginatedResult[models.SpecialDate], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM special_dates").Scan(&total); err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, title, date, COALESCE(description,''), is_anniversary, COALESCE(type,''), created_at, updated_at
		 FROM special_dates ORDER BY id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []models.SpecialDate
	for rows.Next() {
		s, err := scanSpecialDate(rows)
		if err != nil {
			return nil, err
		}
		data = append(data, s)
	}
	if data == nil {
		data = []models.SpecialDate{}
	}
	return &models.PaginatedResult[models.SpecialDate]{Data: data, Total: total, Page: page, Limit: limit}, nil
}

// GetByID 按 id 查询。
func (r *SpecialDateRepo) GetByID(ctx context.Context, id int) (*models.SpecialDate, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, title, date, COALESCE(description,''), is_anniversary, COALESCE(type,''), created_at, updated_at
		 FROM special_dates WHERE id = $1`, id)
	return scanSpecialDateRow(row)
}

// NextID 下一个业务 id。
func (r *SpecialDateRepo) NextID(ctx context.Context) (int, error) {
	var id int
	err := r.pool.QueryRow(ctx, "SELECT COALESCE(MAX(id), 0) + 1 FROM special_dates").Scan(&id)
	return id, err
}

// Create 新建特殊日期。
func (r *SpecialDateRepo) Create(ctx context.Context, in models.SpecialDateInput) (*models.SpecialDate, error) {
	id, err := r.NextID(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	_, err = r.pool.Exec(ctx,
		`INSERT INTO special_dates (id, title, date, description, is_anniversary, type, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$7)`,
		id, in.Title, in.Date, in.Description, in.IsAnniversary, in.Type, now)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

// Update 更新；空字段尽量保留原值。
func (r *SpecialDateRepo) Update(ctx context.Context, id int, in models.SpecialDateInput) (*models.SpecialDate, error) {
	existing, err := r.GetByID(ctx, id)
	if err != nil || existing == nil {
		return nil, err
	}
	title := in.Title
	if title == "" {
		title = existing.Title
	}
	date := in.Date
	if date == "" {
		date = existing.Date
	}
	desc := in.Description
	if in.Title == "" && in.Date == "" && in.Type == "" && !in.IsAnniversary {
		desc = existing.Description
	} else if desc == "" && in.Description == "" {
		desc = existing.Description
	}
	typ := in.Type
	if typ == "" {
		typ = existing.Type
	}
	isAnn := in.IsAnniversary

	_, err = r.pool.Exec(ctx,
		`UPDATE special_dates SET title=$2, date=$3, description=$4, is_anniversary=$5, type=$6, updated_at=$7 WHERE id=$1`,
		id, title, date, desc, isAnn, typ, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

// Delete 删除记录。
func (r *SpecialDateRepo) Delete(ctx context.Context, id int) (bool, error) {
	tag, err := r.pool.Exec(ctx, "DELETE FROM special_dates WHERE id=$1", id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func scanSpecialDate(rows pgx.Rows) (models.SpecialDate, error) {
	var s models.SpecialDate
	var createdAt, updatedAt time.Time
	err := rows.Scan(&s.ID, &s.Title, &s.Date, &s.Description, &s.IsAnniversary, &s.Type, &createdAt, &updatedAt)
	s.CreatedAt = createdAt
	s.UpdatedAt = updatedAt
	return s, err
}

func scanSpecialDateRow(row pgx.Row) (*models.SpecialDate, error) {
	var s models.SpecialDate
	var createdAt, updatedAt time.Time
	err := row.Scan(&s.ID, &s.Title, &s.Date, &s.Description, &s.IsAnniversary, &s.Type, &createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.CreatedAt = createdAt
	s.UpdatedAt = updatedAt
	return &s, nil
}
