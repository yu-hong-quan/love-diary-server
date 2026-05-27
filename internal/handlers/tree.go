package handlers

import (
	"net/http"
	"time"

	"love-diary-go/internal/config"
	"love-diary-go/internal/models"
	"love-diary-go/internal/repository"

	"github.com/gin-gonic/gin"
)

// TreeHandler 爱情树相关接口。
type TreeHandler struct {
	repo *repository.TreeRepo
	cfg  *config.Config
}

// NewTreeHandler 创建爱情树处理器。
func NewTreeHandler(repo *repository.TreeRepo, cfg *config.Config) *TreeHandler {
	return &TreeHandler{repo: repo, cfg: cfg}
}

// monthsSince 根据种植日期计算在一起多少个月（与 Node dayjs diff 逻辑一致）。
func monthsSince(startDate string) int {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return 0
	}
	now := time.Now()
	years := now.Year() - start.Year()
	months := int(now.Month()) - int(start.Month())
	total := years*12 + months
	if total < 0 {
		return 0
	}
	return total
}

// withMonths 组装返回给前端的树对象（含动态 months 字段）。
func withMonths(t *models.Tree) gin.H {
	return gin.H{
		"id":          t.ID,
		"startDate":   t.StartDate,
		"lastWatered": t.LastWatered,
		"waterCount":  t.WaterCount,
		"logs":        t.Logs,
		"months":      monthsSince(t.StartDate),
		"createdAt":   t.CreatedAt,
		"updatedAt":   t.UpdatedAt,
	}
}

// ensureTree 获取第一棵树，不存在则按今天日期创建默认记录。
func (h *TreeHandler) ensureTree(c *gin.Context) (*models.Tree, error) {
	tree, err := h.repo.GetFirst(c.Request.Context())
	if err != nil {
		return nil, err
	}
	if tree != nil {
		return tree, nil
	}
	today := time.Now().Format("2006-01-02")
	return h.repo.CreateDefault(c.Request.Context(), today)
}

// Get GET /tree
func (h *TreeHandler) Get(c *gin.Context) {
	tree, err := h.ensureTree(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	c.JSON(http.StatusOK, withMonths(tree))
}

// Water POST /tree/water — 更新浇水日期并 waterCount+1。
func (h *TreeHandler) Water(c *gin.Context) {
	tree, err := h.ensureTree(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	today := time.Now().Format("2006-01-02")
	updated, err := h.repo.Water(c.Request.Context(), tree.ID, today, tree.WaterCount+1)
	if err != nil || updated == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "tree": withMonths(updated)})
}

// Logs GET /tree/logs — 返回浇水日志数组（前端 api 已定义，原 Node 未实现）。
func (h *TreeHandler) Logs(c *gin.Context) {
	tree, err := h.ensureTree(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	if tree.Logs == nil {
		tree.Logs = []models.TreeLog{}
	}
	c.JSON(http.StatusOK, gin.H{"logs": tree.Logs})
}
