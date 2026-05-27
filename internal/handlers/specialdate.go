package handlers

import (
	"net/http"
	"strconv"

	"love-diary-go/internal/models"
	"love-diary-go/internal/repository"

	"github.com/gin-gonic/gin"
)

// SpecialDateHandler 特殊日期（纪念日、生日等）接口。
type SpecialDateHandler struct {
	repo *repository.SpecialDateRepo
}

// NewSpecialDateHandler 创建特殊日期处理器。
func NewSpecialDateHandler(repo *repository.SpecialDateRepo) *SpecialDateHandler {
	return &SpecialDateHandler{repo: repo}
}

// List GET /special-dates?page=&limit=
func (h *SpecialDateHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	result, err := h.repo.List(c.Request.Context(), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetOne GET /special-dates/:id
func (h *SpecialDateHandler) GetOne(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	item, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil || item == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Special date not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

// Create POST /special-dates
func (h *SpecialDateHandler) Create(c *gin.Context) {
	var in models.SpecialDateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}
	item, err := h.repo.Create(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	c.JSON(http.StatusOK, item)
}

// Update PUT /special-dates/:id
func (h *SpecialDateHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in models.SpecialDateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}
	item, err := h.repo.Update(c.Request.Context(), id, in)
	if err != nil || item == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Special date not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

// Delete DELETE /special-dates/:id
func (h *SpecialDateHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	ok, err := h.repo.Delete(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"message": "Special date not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
