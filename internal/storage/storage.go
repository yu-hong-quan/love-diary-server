// Package storage 将日记图片落盘到 uploads/{category}/ 下（不再按日记 id 分子目录）。
package storage

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	TravelDiaries = "travel-diaries"
	DailyDiaries  = "daily-diaries"
	Avatars       = "avatars"
	URLPrefix     = "/uploads"
	// MaxAvatarBytes 头像解码后最大体积（与上传接口 10MB 一致）。
	MaxAvatarBytes = 10 << 20
)

// Store 管理本地图片目录。
type Store struct {
	root string // 例如 ./uploads
}

// New 创建存储；root 为空时使用 ./uploads。
func New(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		root = "./uploads"
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	return &Store{root: abs}, nil
}

// Root 返回磁盘上的 uploads 根目录绝对路径。
func (s *Store) Root() string {
	return s.root
}

// CategoryDir 返回某业务类型的图片目录（二级目录，如 uploads/travel-diaries）。
func (s *Store) CategoryDir(category string) string {
	return filepath.Join(s.root, category)
}

// PublicPath 生成对外 URL：/uploads/{category}/{filename}
func PublicPath(category, filename string) string {
	return fmt.Sprintf("%s/%s/%s", URLPrefix, category, filename)
}

// DiaryFileName 生成文件名：{日记id}-{序号}.ext，例如 5-2.jpg
func DiaryFileName(diaryID, index int, ext string) string {
	return fmt.Sprintf("%d-%d%s", diaryID, index, ext)
}

// DeleteDiaryFiles 删除某条日记在本分类下的全部图片（含旧版 id 子目录）。
func (s *Store) DeleteDiaryFiles(category string, diaryID int) error {
	dir := s.CategoryDir(category)
	prefix := fmt.Sprintf("%d-", diaryID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return removeLegacyDiaryDir(s, category, diaryID)
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), prefix) {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
	_ = removeLegacyDiaryDir(s, category, diaryID)
	return nil
}

func removeLegacyDiaryDir(s *Store, category string, diaryID int) error {
	legacy := filepath.Join(s.CategoryDir(category), strconv.Itoa(diaryID))
	if _, err := os.Stat(legacy); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(legacy)
}

// IsStoredPath 是否为已落盘的相对路径。
func IsStoredPath(p string) bool {
	return strings.HasPrefix(p, URLPrefix+"/")
}

// PersistDiaryImages 将 data URL 写入磁盘；已是 /uploads/ 或 http(s) 的条目原样保留（旧嵌套路径会扁平化）。
func (s *Store) PersistDiaryImages(category string, diaryID int, images []string) ([]string, error) {
	if len(images) == 0 {
		return []string{}, nil
	}

	dir := s.CategoryDir(category)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	out := make([]string, 0, len(images))
	usedFiles := make(map[string]struct{})
	seq := 0

	for i, img := range images {
		img = strings.TrimSpace(img)
		if img == "" {
			continue
		}

		switch {
		case strings.HasPrefix(img, "data:"):
			seq++
			data, ext, err := decodeDataURL(img)
			if err != nil {
				return nil, fmt.Errorf("diary %d image %d: %w", diaryID, i+1, err)
			}
			name := DiaryFileName(diaryID, seq, ext)
			abs := filepath.Join(dir, name)
			if err := os.WriteFile(abs, data, 0o644); err != nil {
				return nil, err
			}
			pub := PublicPath(category, name)
			out = append(out, pub)
			usedFiles[name] = struct{}{}

		case IsStoredPath(img):
			flat, err := s.ensureFlatPath(category, diaryID, img)
			if err != nil {
				return nil, err
			}
			out = append(out, flat)
			usedFiles[filepath.Base(flat)] = struct{}{}

		case strings.HasPrefix(img, "http://") || strings.HasPrefix(img, "https://"):
			out = append(out, img)

		default:
			if !strings.HasPrefix(img, "/") {
				img = URLPrefix + "/" + strings.TrimPrefix(img, "/")
			}
			out = append(out, img)
		}
	}

	if err := removeOrphanDiaryFiles(dir, diaryID, usedFiles); err != nil {
		return nil, err
	}
	return out, nil
}

// ensureFlatPath 把旧路径 /uploads/cat/5/1.jpg 转为 /uploads/cat/5-1.jpg 并移动文件。
func (s *Store) ensureFlatPath(category string, diaryID int, urlPath string) (string, error) {
	rel := strings.TrimPrefix(urlPath, URLPrefix+"/")
	parts := strings.Split(rel, "/")
	if len(parts) < 2 {
		return urlPath, nil
	}
	// 已是扁平：travel-diaries/5-1.jpg
	if len(parts) == 2 && parts[0] == category {
		return urlPath, nil
	}
	// 旧嵌套：travel-diaries/5/1.jpg
	if len(parts) == 3 && parts[0] == category {
		id, _ := strconv.Atoi(parts[1])
		if id != diaryID {
			id = diaryID
		}
		base := parts[2]
		ext := filepath.Ext(base)
		idxStr := strings.TrimSuffix(base, ext)
		idx, _ := strconv.Atoi(idxStr)
		if idx < 1 {
			idx = 1
		}
		name := DiaryFileName(id, idx, ext)
		newPath := PublicPath(category, name)
		oldAbs := filepath.Join(s.CategoryDir(category), parts[1], base)
		newAbs := filepath.Join(s.CategoryDir(category), name)
		if _, err := os.Stat(oldAbs); err == nil {
			_ = os.Rename(oldAbs, newAbs)
		}
		return newPath, nil
	}
	return urlPath, nil
}

// SaveUploadedFile 保存 multipart 上传的二进制，返回对外 URL 路径。
func (s *Store) SaveUploadedFile(category string, diaryID int, data []byte, ext string) (string, error) {
	if ext == "" {
		ext = ".jpg"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	dir := s.CategoryDir(category)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	var name string
	if diaryID > 0 {
		name = fmt.Sprintf("%d-%d-%d%s", diaryID, time.Now().UnixMilli(), time.Now().UnixNano()%1e9, ext)
	} else {
		name = fmt.Sprintf("%d-%d%s", time.Now().UnixMilli(), time.Now().UnixNano()%1e9, ext)
	}
	abs := filepath.Join(dir, name)
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		return "", err
	}
	return PublicPath(category, name), nil
}

// PersistAvatar 将头像 data URL 落盘；已是 /uploads/ 或 http(s) 则原样返回。
func (s *Store) PersistAvatar(userID int, avatar string) (string, error) {
	avatar = strings.TrimSpace(avatar)
	if avatar == "" {
		return "", nil
	}
	if IsStoredPath(avatar) || strings.HasPrefix(avatar, "http://") || strings.HasPrefix(avatar, "https://") {
		return avatar, nil
	}
	if !strings.HasPrefix(avatar, "data:") {
		if !strings.HasPrefix(avatar, "/") {
			avatar = URLPrefix + "/" + strings.TrimPrefix(avatar, "/")
		}
		return avatar, nil
	}

	data, ext, err := decodeDataURL(avatar)
	if err != nil {
		return "", err
	}
	dir := s.CategoryDir(Avatars)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%d%s", userID, ext)
	abs := filepath.Join(dir, name)
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		return "", err
	}
	return PublicPath(Avatars, name), nil
}

func removeOrphanDiaryFiles(dir string, diaryID int, keep map[string]struct{}) error {
	prefix := fmt.Sprintf("%d-", diaryID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		if _, ok := keep[e.Name()]; !ok {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
	return nil
}

func decodeDataURL(s string) ([]byte, string, error) {
	const prefix = "data:"
	if !strings.HasPrefix(s, prefix) {
		return nil, "", fmt.Errorf("not a data URL")
	}
	rest := s[len(prefix):]
	comma := strings.Index(rest, ",")
	if comma < 0 {
		return nil, "", fmt.Errorf("invalid data URL")
	}
	meta, payload := rest[:comma], rest[comma+1:]
	ext := ".jpg"
	if strings.Contains(meta, "image/png") {
		ext = ".png"
	} else if strings.Contains(meta, "image/gif") {
		ext = ".gif"
	} else if strings.Contains(meta, "image/webp") {
		ext = ".webp"
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, "", err
	}
	if len(data) > MaxAvatarBytes {
		return nil, "", fmt.Errorf("avatar exceeds %d bytes", MaxAvatarBytes)
	}
	return data, ext, nil
}

// SaveAvatarBytes 将二进制头像写入 uploads/avatars/{userID}.ext。
func (s *Store) SaveAvatarBytes(userID int, data []byte, ext string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty avatar")
	}
	if len(data) > MaxAvatarBytes {
		return "", fmt.Errorf("avatar exceeds %d bytes", MaxAvatarBytes)
	}
	if ext == "" {
		ext = ".jpg"
	}
	dir := s.CategoryDir(Avatars)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%d%s", userID, ext)
	abs := filepath.Join(dir, name)
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		return "", err
	}
	return PublicPath(Avatars, name), nil
}
