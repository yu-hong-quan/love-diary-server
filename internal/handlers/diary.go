package handlers

import (
	"net/http"
	"strconv"

	"love-diary-go/internal/models"
	"love-diary-go/internal/repository"

	"github.com/gin-gonic/gin"
)

// DiaryHandler 旅行日记与日常日记共用同一套 handler，通过不同 repo 区分表。
type DiaryHandler struct {
	repo        *repository.DiaryRepo
	notFoundMsg string // 404 提示文案
	listErrMsg  string // 列表 500 提示文案
}

// NewTravelHandler 旅行日记 handler。
func NewTravelHandler(repo *repository.DiaryRepo) *DiaryHandler {
	return &DiaryHandler{repo: repo, notFoundMsg: "Travel diary not found", listErrMsg: "Failed to get travel diaries"}
}

// NewDailyHandler 日常日记 handler。
func NewDailyHandler(repo *repository.DiaryRepo) *DiaryHandler {
	return &DiaryHandler{repo: repo, notFoundMsg: "Daily diary not found", listErrMsg: "Failed to get daily diaries"}
}

// List GET /travel-diaries | /daily-diaries?page=&limit=
func (h *DiaryHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	result, err := h.repo.List(c.Request.Context(), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": h.listErrMsg})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetOne GET /:base/:id
func (h *DiaryHandler) GetOne(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	diary, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	if diary == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": h.notFoundMsg})
		return
	}
	c.JSON(http.StatusOK, diary)
}

// Create POST /:base
func (h *DiaryHandler) Create(c *gin.Context) {
	var in models.DiaryInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}
	diary, err := h.repo.Create(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	c.JSON(http.StatusOK, diary)
}

// Update PUT /:base/:id — 支持部分字段更新（合并已有记录）。
func (h *DiaryHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	existing, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil || existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": h.notFoundMsg})
		return
	}

	var patch map[string]interface{}
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	in := models.DiaryInput{
		Title:    existing.Title,
		Content:  existing.Content,
		Date:     existing.Date,
		Images:   existing.Images,
		Likes:    existing.Likes,
		Comments: existing.Comments,
	}
	if v, ok := patch["title"].(string); ok {
		in.Title = v
	}
	if v, ok := patch["content"].(string); ok {
		in.Content = v
	}
	if v, ok := patch["date"].(string); ok {
		in.Date = v
	}
	if v, ok := patch["image"].(string); ok {
		in.Image = v
	}
	if v, ok := patch["images"].([]interface{}); ok {
		imgs := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				imgs = append(imgs, s)
			}
		}
		in.Images = imgs
	}

	diary, err := h.repo.Update(c.Request.Context(), id, in)
	if err != nil || diary == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": h.notFoundMsg})
		return
	}
	c.JSON(http.StatusOK, diary)
}

// Like PUT /:base/:id/like — 点赞数 +1。
func (h *DiaryHandler) Like(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	diary, err := h.repo.IncrementLikes(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	if diary == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": h.notFoundMsg})
		return
	}
	c.JSON(http.StatusOK, diary)
}

// Delete DELETE /:base/:id — 成功返回 204。
func (h *DiaryHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	ok, err := h.repo.Delete(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"message": h.notFoundMsg})
		return
	}
	c.Status(http.StatusNoContent)
}
