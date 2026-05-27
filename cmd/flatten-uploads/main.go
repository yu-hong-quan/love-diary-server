// 将旧目录 uploads/{category}/{id}/*.jpg 扁平化为 uploads/{category}/{id}-*.jpg，并更新数据库路径。
//
//	go run ./cmd/flatten-uploads
//	go run ./cmd/flatten-uploads -dry-run
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"love-diary-go/internal/config"
	"love-diary-go/internal/database"
	"love-diary-go/internal/storage"
	"love-diary-go/internal/util"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "只打印，不移动文件、不改库")
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
		n, err := flattenTable(ctx, pool, store, job.table, job.category, *dryRun)
		if err != nil {
			log.Fatalf("%s: %v", job.table, err)
		}
		total += n
		log.Printf("%s: updated %d diaries", job.table, n)
	}

	nDirs, _ := flattenDiskDirs(store, *dryRun)
	log.Printf("legacy subdirs processed: %d", nDirs)
	log.Printf("done, diaries updated: %d", total)
	if *dryRun {
		log.Println("dry-run: no changes applied")
	}
}

func flattenTable(ctx context.Context, pool *pgxpool.Pool, store *storage.Store, table, category string, dryRun bool) (int, error) {
	rows, err := pool.Query(ctx, `SELECT id, image, images FROM `+table+` ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	updated := 0
	for rows.Next() {
		var id int
		var image *string
		var imagesJSON []byte
		if err := rows.Scan(&id, &image, &imagesJSON); err != nil {
			return updated, err
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
		if len(prepared) == 0 {
			continue
		}

		flat := make([]string, 0, len(prepared))
		changed := false
		for _, p := range prepared {
			newP, c, err := flattenOne(store, category, id, p, dryRun)
			if err != nil {
				return updated, fmt.Errorf("id %d: %w", id, err)
			}
			flat = append(flat, newP)
			if c {
				changed = true
			}
		}
		if !changed {
			continue
		}
		updated++
		fmt.Fprintf(os.Stdout, "  [%s] id=%d -> %v\n", category, id, flat)
		if dryRun {
			continue
		}
		imgJSON, _ := json.Marshal(flat)
		var first *string
		if len(flat) > 0 {
			first = &flat[0]
		}
		if _, err := pool.Exec(ctx,
			`UPDATE `+table+` SET image=$2, images=$3::jsonb, updated_at=NOW() WHERE id=$1`,
			id, first, imgJSON); err != nil {
			return updated, err
		}
	}
	return updated, rows.Err()
}

func flattenOne(store *storage.Store, category string, diaryID int, urlPath string, dryRun bool) (string, bool, error) {
	if !storage.IsStoredPath(urlPath) {
		return urlPath, false, nil
	}
	rel := strings.TrimPrefix(urlPath, storage.URLPrefix+"/")
	parts := strings.Split(rel, "/")
	if len(parts) != 3 || parts[0] != category {
		return urlPath, false, nil
	}
	subID, _ := strconv.Atoi(parts[1])
	base := parts[2]
	ext := filepath.Ext(base)
	idxStr := strings.TrimSuffix(base, ext)
	idx, _ := strconv.Atoi(idxStr)
	if idx < 1 {
		idx = 1
	}
	if subID == 0 {
		subID = diaryID
	}
	name := storage.DiaryFileName(subID, idx, ext)
	newPath := storage.PublicPath(category, name)
	oldAbs := filepath.Join(store.CategoryDir(category), parts[1], base)
	newAbs := filepath.Join(store.CategoryDir(category), name)
	if !dryRun {
		if _, err := os.Stat(oldAbs); err == nil {
			_ = os.Rename(oldAbs, newAbs)
		}
		legacyDir := filepath.Join(store.CategoryDir(category), parts[1])
		_ = tryRemoveEmptyDir(legacyDir)
	}
	return newPath, true, nil
}

func flattenDiskDirs(store *storage.Store, dryRun bool) (int, error) {
	count := 0
	for _, cat := range []string{storage.TravelDiaries, storage.DailyDiaries} {
		dir := store.CategoryDir(cat)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return count, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			diaryID, err := strconv.Atoi(e.Name())
			if err != nil {
				continue
			}
			sub := filepath.Join(dir, e.Name())
			files, _ := os.ReadDir(sub)
			for i, f := range files {
				if f.IsDir() {
					continue
				}
				ext := filepath.Ext(f.Name())
				idx := i + 1
				if n, err := strconv.Atoi(strings.TrimSuffix(f.Name(), ext)); err == nil && n > 0 {
					idx = n
				}
				name := storage.DiaryFileName(diaryID, idx, ext)
				oldAbs := filepath.Join(sub, f.Name())
				newAbs := filepath.Join(dir, name)
				fmt.Fprintf(os.Stdout, "  move %s -> %s\n", oldAbs, newAbs)
				if !dryRun {
					_ = os.Rename(oldAbs, newAbs)
				}
				count++
			}
			if !dryRun {
				_ = os.RemoveAll(sub)
			}
		}
	}
	return count, nil
}

func tryRemoveEmptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return os.Remove(dir)
	}
	return nil
}
