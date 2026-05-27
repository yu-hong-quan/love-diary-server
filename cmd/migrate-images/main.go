// 一次性工具：把数据库里日记的 base64 图片导出到 uploads/ 并写回 /uploads/... 路径。
//
// 用法（在 love-diary-go 目录）:
//
//	go run ./cmd/migrate-images
//	go run ./cmd/migrate-images -dry-run
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"love-diary-go/internal/config"
	"love-diary-go/internal/database"
	"love-diary-go/internal/storage"
	"love-diary-go/internal/util"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "只统计，不写文件、不更新数据库")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	store, err := storage.New(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	total := 0
	for _, job := range []struct {
		table    string
		category string
	}{
		{"travel_diaries", storage.TravelDiaries},
		{"daily_diaries", storage.DailyDiaries},
	} {
		n, err := migrateTable(ctx, pool, store, job.table, job.category, *dryRun)
		if err != nil {
			log.Fatalf("%s: %v", job.table, err)
		}
		total += n
		log.Printf("%s: migrated %d diaries", job.table, n)
	}

	log.Printf("done, total diaries with images converted: %d (upload root: %s)", total, store.Root())
	if *dryRun {
		log.Println("dry-run: no files or DB rows were changed")
	}
}

func migrateTable(ctx context.Context, pool *pgxpool.Pool, store *storage.Store, table, category string, dryRun bool) (int, error) {
	rows, err := pool.Query(ctx, `SELECT id, image, images FROM `+table+` ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int
		var image *string
		var imagesJSON []byte
		if err := rows.Scan(&id, &image, &imagesJSON); err != nil {
			return count, err
		}

		var images []string
		if len(imagesJSON) > 0 {
			_ = json.Unmarshal(imagesJSON, &images)
		}
		imgStr := ""
		if image != nil {
			imgStr = *image
		}
		prepared := util.PrepareDiaryImages(imgStr, images)
		if !storage.HasDataURL(prepared) {
			continue
		}

		count++
		fmt.Fprintf(os.Stdout, "  [%s] id=%d images=%d (base64)\n", category, id, len(prepared))
		if dryRun {
			continue
		}

		paths, err := store.PersistDiaryImages(category, id, prepared)
		if err != nil {
			return count, fmt.Errorf("id %d: %w", id, err)
		}
		imgJSON, err := json.Marshal(paths)
		if err != nil {
			return count, err
		}
		var first *string
		if len(paths) > 0 {
			first = &paths[0]
		}
		_, err = pool.Exec(ctx,
			`UPDATE `+table+` SET image=$2, images=$3::jsonb, updated_at=NOW() WHERE id=$1`,
			id, first, imgJSON)
		if err != nil {
			return count, err
		}
	}
	return count, rows.Err()
}
