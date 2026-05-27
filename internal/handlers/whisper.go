package handlers

import (
	"net/http"
	"strconv"

	"love-diary-go/internal/models"
	"love-diary-go/internal/repository"

	"github.com/gin-gonic/gin"
)

// WhisperHandler 悄悄话接口。
type WhisperHandler struct {
	repo *repository.WhisperRepo
}

// NewWhisperHandler 创建悄悄话处理器。
func NewWhisperHandler(repo *repository.WhisperRepo) *WhisperHandler {
	return &WhisperHandler{repo: repo}
}

// List GET /whispers?page=&limit=
func (h *WhisperHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	result, err := h.repo.List(c.Request.Context(), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetOne GET /whispers/:id
func (h *WhisperHandler) GetOne(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	w, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	if w == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Whisper not found"})
		return
	}
	c.JSON(http.StatusOK, w)
}

// Create POST /whispers
func (h *WhisperHandler) Create(c *gin.Context) {
	var in models.WhisperInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}
	w, err := h.repo.Create(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	c.JSON(http.StatusOK, w)
}

// Update PUT /whispers/:id
func (h *WhisperHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in models.WhisperInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}
	w, err := h.repo.Update(c.Request.Context(), id, in)
	if err != nil || w == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Whisper not found"})
		return
	}
	c.JSON(http.StatusOK, w)
}

// Like PUT /whispers/:id/like
func (h *WhisperHandler) Like(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	w, err := h.repo.IncrementLikes(c.Request.Context(), id)
	if err != nil || w == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Whisper not found"})
		return
	}
	c.JSON(http.StatusOK, w)
}

// Delete DELETE /whispers/:id
func (h *WhisperHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	ok, err := h.repo.Delete(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"message": "Whisper not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
