package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"love-diary-go/internal/repository"
	"love-diary-go/internal/storage"

	"github.com/gin-gonic/gin"
)

// UploadHandler 图片上传：落盘到 uploads/{category}/{diaryId}/ 并返回 /uploads/... 路径。
type UploadHandler struct {
	store *storage.Store
	users *repository.UserRepo
}

// NewUploadHandler 创建上传处理器。
func NewUploadHandler(store *storage.Store, users *repository.UserRepo) *UploadHandler {
	return &UploadHandler{store: store, users: users}
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

// UploadAvatar POST /upload/avatar — multipart 上传头像（需登录），避免 base64 JSON 过大。
func (h *UploadHandler) UploadAvatar(c *gin.Context) {
	account, _ := c.Get("account")
	accountStr, _ := account.(string)
	if accountStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}
	if h.users == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}

	user, err := h.users.FindByAccountLegacy(c.Request.Context(), accountStr)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No file uploaded"})
		return
	}
	if file.Size > storage.MaxAvatarBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"message": "File too large"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Upload failed"})
		return
	}
	defer f.Close()

	buf := make([]byte, file.Size)
	if _, err := f.Read(buf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Upload failed"})
		return
	}

	urlPath, err := h.store.SaveAvatarBytes(user.ID, buf, extFromMIME(file.Header.Get("Content-Type")))
	if err != nil {
		if strings.Contains(err.Error(), "exceeds") {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"message": "File too large"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Upload failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"url":  urlPath,
			"path": urlPath,
		},
	})
}
