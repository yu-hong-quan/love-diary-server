package handlers

import (
	"net/http"

	"love-diary-go/internal/repository"

	"github.com/gin-gonic/gin"
)

// RomanticToastHandler 首页浪漫 toast 接口。
type RomanticToastHandler struct {
	repo *repository.RomanticToastRepo
}

// NewRomanticToastHandler 创建处理器。
func NewRomanticToastHandler(repo *repository.RomanticToastRepo) *RomanticToastHandler {
	return &RomanticToastHandler{repo: repo}
}

// Random GET /romantic-toasts/random — 随机一条浪漫文案（HTTP 200，与 whisper 等 GET 接口一致）。
func (h *RomanticToastHandler) Random(c *gin.Context) {
	toast, err := h.repo.Random(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	if toast == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "No romantic toasts configured"})
		return
	}
	c.JSON(http.StatusOK, toast)
}
