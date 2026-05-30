package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"love-diary-go/internal/models"
	"love-diary-go/internal/repository"

	"github.com/gin-gonic/gin"
)

// GameHandler 游戏模块接口。
type GameHandler struct {
	repo *repository.GameRepo
}

// NewGameHandler 创建游戏处理器。
func NewGameHandler(repo *repository.GameRepo) *GameHandler {
	return &GameHandler{repo: repo}
}

// RegisterGameRoutes 注册游戏路由。
func RegisterGameRoutes(r *gin.Engine, h *GameHandler) {
	g := r.Group("/games")
	g.GET("/types", h.ListTypes)
	g.GET("/sessions", h.ListSessions)

	gomoku := g.Group("/gomoku")
	gomoku.GET("/records", h.ListGomokuRecords)
	gomoku.POST("/sessions", h.CreateGomokuSession)
	gomoku.GET("/sessions/:id", h.GetGomokuSession)
	gomoku.POST("/sessions/:id/moves", h.PlaceGomokuMove)

	dice := g.Group("/dice")
	dice.GET("/records", h.ListDiceRecords)
	dice.POST("/sessions", h.CreateDiceSession)
	dice.GET("/sessions/:id", h.GetDiceSession)
	dice.POST("/sessions/:id/roll", h.RollDice)
}

// ListTypes GET /games/types
func (h *GameHandler) ListTypes(c *gin.Context) {
	c.JSON(http.StatusOK, h.repo.ListGameTypes())
}

// ListSessions GET /games/sessions?gameType=&status=&page=&limit=
func (h *GameHandler) ListSessions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	result, err := h.repo.ListSessions(c.Request.Context(), c.Query("gameType"), c.Query("status"), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ListGomokuRecords GET /games/gomoku/records — 已结束的五子棋对局。
func (h *GameHandler) ListGomokuRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	result, err := h.repo.ListFinishedSessions(c.Request.Context(), models.GameTypeGomoku, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// CreateGomokuSession POST /games/gomoku/sessions
func (h *GameHandler) CreateGomokuSession(c *gin.Context) {
	var in models.CreateGomokuSessionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}
	in.Player1Name = strings.TrimSpace(in.Player1Name)
	in.Player2Name = strings.TrimSpace(in.Player2Name)
	if in.Player1Name == "" || in.Player2Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请填写两位角色名称"})
		return
	}
	detail, err := h.repo.CreateGomokuSession(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	c.JSON(http.StatusOK, detail)
}

// GetGomokuSession GET /games/gomoku/sessions/:id
func (h *GameHandler) GetGomokuSession(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	detail, err := h.repo.GetGomokuSession(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	if detail == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Session not found"})
		return
	}
	c.JSON(http.StatusOK, detail)
}

// PlaceGomokuMove POST /games/gomoku/sessions/:id/moves
func (h *GameHandler) PlaceGomokuMove(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in models.GomokuMoveInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}
	detail, err := h.repo.PlaceGomokuMove(c.Request.Context(), id, in.Row, in.Col)
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "finished"), strings.Contains(msg, "occupied"), strings.Contains(msg, "out of board"):
			c.JSON(http.StatusBadRequest, gin.H{"message": msg})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		}
		return
	}
	if detail == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Session not found"})
		return
	}
	c.JSON(http.StatusOK, detail)
}

// ListDiceRecords GET /games/dice/records
func (h *GameHandler) ListDiceRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	result, err := h.repo.ListFinishedSessions(c.Request.Context(), models.GameTypeDice, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// CreateDiceSession POST /games/dice/sessions
func (h *GameHandler) CreateDiceSession(c *gin.Context) {
	var in models.CreateDiceSessionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}
	in.Player1Name = strings.TrimSpace(in.Player1Name)
	in.Player2Name = strings.TrimSpace(in.Player2Name)
	if in.Player1Name == "" || in.Player2Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请填写两位角色名称"})
		return
	}
	detail, err := h.repo.CreateDiceSession(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	c.JSON(http.StatusOK, detail)
}

// GetDiceSession GET /games/dice/sessions/:id
func (h *GameHandler) GetDiceSession(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	detail, err := h.repo.GetDiceSession(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	if detail == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Session not found"})
		return
	}
	c.JSON(http.StatusOK, detail)
}

// RollDice POST /games/dice/sessions/:id/roll
func (h *GameHandler) RollDice(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in models.DiceRollInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}
	result, err := h.repo.RollDice(c.Request.Context(), id, in.Player)
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "waiting"), strings.Contains(msg, "finished"), strings.Contains(msg, "invalid"):
			c.JSON(http.StatusBadRequest, gin.H{"message": msg})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		}
		return
	}
	if result == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Session not found"})
		return
	}
	c.JSON(http.StatusOK, result)
}
