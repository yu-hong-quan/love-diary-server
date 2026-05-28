// Package handlers 实现各业务模块的 HTTP 接口，响应格式与前端约定一致。
package handlers

import (
	"net/http"
	"strings"
	"time"

	"love-diary-go/internal/config"
	"love-diary-go/internal/middleware"
	"love-diary-go/internal/models"
	"love-diary-go/internal/repository"
	"love-diary-go/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthHandler 处理登录与用户资料相关请求。
type AuthHandler struct {
	users  *repository.UserRepo
	store  *storage.Store
	secret string
}

// NewAuthHandler 创建登录处理器。
func NewAuthHandler(users *repository.UserRepo, store *storage.Store, cfg *config.Config) *AuthHandler {
	return &AuthHandler{users: users, store: store, secret: cfg.JWTSecret}
}

type loginRequest struct {
	Account  string `json:"account"`
	Username string `json:"username"` // 兼容旧版前端
	Password string `json:"password"`
}

// Login POST /auth/login — 校验 users 表账号密码，签发 7 天 JWT。
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	account := strings.TrimSpace(req.Account)
	if account == "" {
		account = strings.TrimSpace(req.Username)
	}
	if account == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "账号和密码不能为空"})
		return
	}

	user, err := h.users.FindByAccountLegacy(c.Request.Context(), account)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Server error"})
		return
	}
	if user == nil || user.Password != req.Password {
		c.JSON(http.StatusOK, gin.H{
			"success":    false,
			"message":    "账号或密码错误",
			"isLoggedIn": false,
		})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"account":    user.Account,
		"username":   user.Account, // 兼容旧 middleware / 评论逻辑
		"isLoggedIn": true,
		"exp":        time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(h.secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Server error"})
		return
	}

	profile := repository.ToProfile(user)
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "登录成功",
		"isLoggedIn": true,
		"token":      tokenStr,
		"expiresIn":  "7d",
		"account":    profile.Account,
		"nickname":   profile.Nickname,
		"avatar":     profile.Avatar,
		"username":   profile.Account, // 兼容旧前端
	})
}

// GetProfile GET /auth/profile — 获取当前登录用户资料。
func (h *AuthHandler) GetProfile(c *gin.Context) {
	account := accountFromContext(c)
	if account == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	user, err := h.users.FindByAccountLegacy(c.Request.Context(), account)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}
	c.JSON(http.StatusOK, repository.ToProfile(user))
}

// UpdateProfile PUT /auth/profile — 修改昵称与头像。
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	account := accountFromContext(c)
	if account == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	user, err := h.users.FindByAccountLegacy(c.Request.Context(), account)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	var in models.UserProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	nickname := strings.TrimSpace(in.Nickname)
	if nickname == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "昵称不能为空"})
		return
	}

	avatar := user.Avatar
	if in.Avatar != "" {
		if h.store != nil {
			saved, err := h.store.PersistAvatar(user.ID, in.Avatar)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": "头像保存失败"})
				return
			}
			avatar = saved
		} else {
			avatar = in.Avatar
		}
	}

	updated, err := h.users.UpdateProfile(c.Request.Context(), account, nickname, avatar)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Server error"})
		return
	}
	c.JSON(http.StatusOK, repository.ToProfile(updated))
}

// RegisterAuthRoutes 注册登录与资料路由。
func RegisterAuthRoutes(r *gin.Engine, h *AuthHandler, cfg *config.Config) {
	auth := middleware.Auth(cfg)
	r.POST("/auth/login", h.Login)
	r.GET("/auth/profile", auth, h.GetProfile)
	r.PUT("/auth/profile", auth, h.UpdateProfile)
}

func accountFromContext(c *gin.Context) string {
	if v, ok := c.Get("account"); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	if v, ok := c.Get("username"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
