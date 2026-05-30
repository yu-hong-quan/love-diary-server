package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"love-diary-go/internal/config"
	"love-diary-go/internal/middleware"
	"love-diary-go/internal/models"
	"love-diary-go/internal/repository"

	"github.com/gin-gonic/gin"
)

// CommentHandler 日记评论接口。
type CommentHandler struct {
	repo        *repository.CommentRepo
	diaryRepo   *repository.DiaryRepo
	notFoundMsg string
}

// NewTravelCommentHandler 旅行日记评论。
func NewTravelCommentHandler(commentRepo *repository.CommentRepo, diaryRepo *repository.DiaryRepo) *CommentHandler {
	return &CommentHandler{
		repo:        commentRepo,
		diaryRepo:   diaryRepo,
		notFoundMsg: "Travel diary not found",
	}
}

// NewDailyCommentHandler 日常日记评论。
func NewDailyCommentHandler(commentRepo *repository.CommentRepo, diaryRepo *repository.DiaryRepo) *CommentHandler {
	return &CommentHandler{
		repo:        commentRepo,
		diaryRepo:   diaryRepo,
		notFoundMsg: "Daily diary not found",
	}
}

// List GET /:base/:id/comments
func (h *CommentHandler) List(c *gin.Context) {
	diaryID, _ := strconv.Atoi(c.Param("id"))
	diary, err := h.diaryRepo.GetByID(c.Request.Context(), diaryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	if diary == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": h.notFoundMsg})
		return
	}

	list, err := h.repo.ListByDiaryID(c.Request.Context(), diaryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to get comments"})
		return
	}
	c.JSON(http.StatusOK, list)
}

// Create POST /:base/:id/comments — 需登录。
func (h *CommentHandler) Create(c *gin.Context) {
	diaryID, _ := strconv.Atoi(c.Param("id"))
	diary, err := h.diaryRepo.GetByID(c.Request.Context(), diaryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	if diary == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": h.notFoundMsg})
		return
	}

	var in models.CommentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}
	content := strings.TrimSpace(in.Content)
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "评论内容不能为空"})
		return
	}

	username, _ := c.Get("username")
	usernameStr, _ := username.(string)
	if usernameStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	comment, err := h.repo.Create(c.Request.Context(), diaryID, usernameStr, content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	c.JSON(http.StatusOK, comment)
}

// Delete DELETE /:base/:id/comments/:commentId — 仅可删除自己的评论。
func (h *CommentHandler) Delete(c *gin.Context) {
	diaryID, _ := strconv.Atoi(c.Param("id"))
	commentID, _ := strconv.Atoi(c.Param("commentId"))

	username, _ := c.Get("username")
	usernameStr, _ := username.(string)
	if usernameStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	ok, err := h.repo.Delete(c.Request.Context(), diaryID, commentID, usernameStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"message": "Comment not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

// RegisterCommentRoutes 注册评论路由（读公开，写需登录）。
func RegisterCommentRoutes(r *gin.Engine, base string, h *CommentHandler, cfg *config.Config) {
	auth := middleware.Auth(cfg)
	r.GET(base+"/:id/comments", h.List)
	r.POST(base+"/:id/comments", auth, h.Create)
	r.DELETE(base+"/:id/comments/:commentId", auth, h.Delete)
}
