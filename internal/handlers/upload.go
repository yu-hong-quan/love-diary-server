package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"love-diary-go/internal/storage"

	"github.com/gin-gonic/gin"
)

// UploadHandler 图片上传：落盘到 uploads/{category}/{diaryId}/ 并返回 /uploads/... 路径。
type UploadHandler struct {
	store *storage.Store
}

// NewUploadHandler 创建上传处理器。
func NewUploadHandler(store *storage.Store) *UploadHandler {
	return &UploadHandler{store: store}
}

func extFromMIME(mime string) string {
	switch mime {
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

func parseCategory(raw string) (string, bool) {
	switch strings.TrimSpace(raw) {
	case storage.TravelDiaries, "travel":
		return storage.TravelDiaries, true
	case storage.DailyDiaries, "daily":
		return storage.DailyDiaries, true
	default:
		return "", false
	}
}

// UploadImage POST /upload/image?category=travel-diaries&diary_id=1
// 表单字段 image；diary_id 缺省时使用 0（临时目录，创建日记后请随正文一并提交 base64 或带 id 再传）。
func (h *UploadHandler) UploadImage(c *gin.Context) {
	category, ok := parseCategory(c.Query("category"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"message": "category 必填：travel-diaries 或 daily-diaries"})
		return
	}
	diaryID, _ := strconv.Atoi(c.DefaultQuery("diary_id", "0"))

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No file uploaded"})
		return
	}
	if file.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "File too large"})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Upload failed", "error": err.Error()})
		return
	}
	defer f.Close()

	buf := make([]byte, file.Size)
	if _, err := f.Read(buf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Upload failed", "error": err.Error()})
		return
	}

	mime := file.Header.Get("Content-Type")
	urlPath, err := h.store.SaveUploadedFile(category, diaryID, buf, extFromMIME(mime))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Upload failed", "error": err.Error()})
		return
	}

	filename := fmt.Sprintf("image-%d%s", time.Now().UnixMilli(), extFromMIME(mime))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"url":      urlPath,
			"filename": filename,
			"path":     urlPath,
		},
	})
}

// UploadImages POST /upload/images?category=...&diary_id=...
func (h *UploadHandler) UploadImages(c *gin.Context) {
	category, ok := parseCategory(c.Query("category"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"message": "category 必填：travel-diaries 或 daily-diaries"})
		return
	}
	diaryID, _ := strconv.Atoi(c.DefaultQuery("diary_id", "0"))

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No files uploaded"})
		return
	}
	files := form.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No files uploaded"})
		return
	}
	if len(files) > 9 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Too many files"})
		return
	}

	uploaded := make([]gin.H, 0, len(files))
	for _, file := range files {
		if file.Size > 10*1024*1024 {
			continue
		}
		f, err := file.Open()
		if err != nil {
			continue
		}
		buf := make([]byte, file.Size)
		_, _ = f.Read(buf)
		f.Close()
		mime := file.Header.Get("Content-Type")
		urlPath, err := h.store.SaveUploadedFile(category, diaryID, buf, extFromMIME(mime))
		if err != nil {
			continue
		}
		uploaded = append(uploaded, gin.H{
			"url":      urlPath,
			"filename": file.Filename,
			"path":     urlPath,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"images": uploaded,
			"count":  len(uploaded),
		},
	})
}
