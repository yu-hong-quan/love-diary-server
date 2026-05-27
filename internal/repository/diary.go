// Package repository 封装 PostgreSQL 数据访问，表名与迁移 SQL 一致。
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"love-diary-go/internal/models"
	"love-diary-go/internal/storage"
	"love-diary-go/internal/util"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DiaryRepo 旅行/日常日记仓储，通过 table 字段区分 travel_diaries / daily_diaries。
type DiaryRepo struct {
	pool     *pgxpool.Pool
	table    string
	category string
	store    *storage.Store
}

// NewTravelRepo 旅行日记表 travel_diaries。
func NewTravelRepo(pool *pgxpool.Pool, store *storage.Store) *DiaryRepo {
	return &DiaryRepo{pool: pool, table: "travel_diaries", category: storage.TravelDiaries, store: store}
}

// NewDailyRepo 日常日记表 daily_diaries。
func NewDailyRepo(pool *pgxpool.Pool, store *storage.Store) *DiaryRepo {
	return &DiaryRepo{pool: pool, table: "daily_diaries", category: storage.DailyDiaries, store: store}
}

func (r *DiaryRepo) persistImages(ctx context.Context, id int, image string, images []string) ([]string, error) {
	prepared := util.PrepareDiaryImages(image, images)
	if r.store == nil || !storage.HasDataURL(prepared) {
		return prepared, nil
	}
	return r.store.PersistDiaryImages(r.category, id, prepared)
}

// List 分页查询，按 id 倒序（最新在前）。
func (r *DiaryRepo) List(ctx context.Context, page, limit int) (*models.PaginatedResult[models.Diary], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+r.table).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, title, COALESCE(content,''), COALESCE(image,''), images, date, likes, comments, created_at, updated_at
		 FROM `+r.table+` ORDER BY id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []models.Diary
	for rows.Next() {
		d, err := scanDiary(rows)
		if err != nil {
			return nil, err
		}
		data = append(data, d)
	}
	if data == nil {
		data = []models.Diary{}
	}
	return &models.PaginatedResult[models.Diary]{Data: data, Total: total, Page: page, Limit: limit}, nil
}

// GetByID 按业务主键 id 查询（非 SERIAL，与 Mongo 迁移数据一致）。
func (r *DiaryRepo) GetByID(ctx context.Context, id int) (*models.Diary, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, title, COALESCE(content,''), COALESCE(image,''), images, date, likes, comments, created_at, updated_at
		 FROM `+r.table+` WHERE id = $1`, id)
	d, err := scanDiaryRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return d, err
}

// NextID 自增业务 id：MAX(id)+1，与旧 Mongo 整数 id 策略相同。
func (r *DiaryRepo) NextID(ctx context.Context) (int, error) {
	var id int
	err := r.pool.QueryRow(ctx, "SELECT COALESCE(MAX(id), 0) + 1 FROM "+r.table).Scan(&id)
	return id, err
}

// Create 插入新日记；images 存 JSONB，首张图同步写入 image 列以兼容旧数据。
func (r *DiaryRepo) Create(ctx context.Context, in models.DiaryInput) (*models.Diary, error) {
	id, err := r.NextID(ctx)
	if err != nil {
		return nil, err
	}
	images, err := r.persistImages(ctx, id, in.Image, in.Images)
	if err != nil {
		return nil, err
	}
	imgJSON, err := json.Marshal(images)
	if err != nil {
		return nil, err
	}
	var firstImage *string
	if len(images) > 0 {
		firstImage = &images[0]
	}
	likes := in.Likes
	comments := in.Comments
	now := time.Now().UTC()

	_, err = r.pool.Exec(ctx,
		`INSERT INTO `+r.table+` (id, title, content, image, images, date, likes, comments, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5::jsonb,$6,$7,$8,$9,$9)`,
		id, in.Title, in.Content, firstImage, imgJSON, in.Date, likes, comments, now)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

// Update 全量更新指定字段（handler 已合并 patch 与已有记录）。
func (r *DiaryRepo) Update(ctx context.Context, id int, in models.DiaryInput) (*models.Diary, error) {
	existing, err := r.GetByID(ctx, id)
	if err != nil || existing == nil {
		return nil, err
	}

	title := in.Title
	if title == "" {
		title = existing.Title
	}
	content := in.Content
	if in.Title == "" && in.Date == "" && in.Image == "" && in.Images == nil {
		content = existing.Content
	}
	date := in.Date
	if date == "" {
		date = existing.Date
	}

	images := existing.Images
	if in.Images != nil || in.Image != "" {
		var err error
		images, err = r.persistImages(ctx, id, in.Image, in.Images)
		if err != nil {
			return nil, err
		}
	}
	imgJSON, err := json.Marshal(images)
	if err != nil {
		return nil, err
	}
	var firstImage *string
	if len(images) > 0 {
		firstImage = &images[0]
	}

	likes := existing.Likes
	comments := existing.Comments

	_, err = r.pool.Exec(ctx,
		`UPDATE `+r.table+` SET title=$2, content=$3, image=$4, images=$5::jsonb, date=$6, likes=$7, comments=$8, updated_at=$9
		 WHERE id=$1`,
		id, title, content, firstImage, imgJSON, date, likes, comments, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

// UpdateFields 按字段 map 更新（目前仅 likes 点赞使用）。
func (r *DiaryRepo) UpdateFields(ctx context.Context, id int, fields map[string]interface{}) (*models.Diary, error) {
	existing, err := r.GetByID(ctx, id)
	if err != nil || existing == nil {
		return nil, err
	}

	likes := existing.Likes
	if v, ok := fields["likes"].(int); ok {
		likes = v
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE `+r.table+` SET likes=$2, updated_at=$3 WHERE id=$1`,
		id, likes, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

// IncrementLikes 点赞数 +1。
func (r *DiaryRepo) IncrementLikes(ctx context.Context, id int) (*models.Diary, error) {
	existing, err := r.GetByID(ctx, id)
	if err != nil || existing == nil {
		return nil, err
	}
	return r.UpdateFields(ctx, id, map[string]interface{}{"likes": existing.Likes + 1})
}

// Delete 删除记录及对应图片目录，返回是否删到行。
func (r *DiaryRepo) Delete(ctx context.Context, id int) (bool, error) {
	tag, err := r.pool.Exec(ctx, "DELETE FROM "+r.table+" WHERE id=$1", id)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() > 0 && r.store != nil {
		_ = r.store.DeleteDiaryFiles(r.category, id)
	}
	return tag.RowsAffected() > 0, nil
}

func scanDiary(rows pgx.Rows) (models.Diary, error) {
	var d models.Diary
	var image string
	var imagesJSON []byte
	var createdAt, updatedAt time.Time
	err := rows.Scan(&d.ID, &d.Title, &d.Content, &image, &imagesJSON, &d.Date, &d.Likes, &d.Comments, &createdAt, &updatedAt)
	if err != nil {
		return d, err
	}
	var images []string
	_ = json.Unmarshal(imagesJSON, &images)
	d.Images = util.NormalizeDiaryImages(image, images)
	d.CreatedAt = createdAt
	d.UpdatedAt = updatedAt
	return d, nil
}

func scanDiaryRow(row pgx.Row) (*models.Diary, error) {
	var d models.Diary
	var image string
	var imagesJSON []byte
	var createdAt, updatedAt time.Time
	err := row.Scan(&d.ID, &d.Title, &d.Content, &image, &imagesJSON, &d.Date, &d.Likes, &d.Comments, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	var images []string
	_ = json.Unmarshal(imagesJSON, &images)
	d.Images = util.NormalizeDiaryImages(image, images)
	d.CreatedAt = createdAt
	d.UpdatedAt = updatedAt
	return &d, nil
}
