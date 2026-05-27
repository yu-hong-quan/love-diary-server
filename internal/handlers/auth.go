// Package handlers 实现各业务模块的 HTTP 接口，响应格式与前端约定一致。
package handlers

import (
	"net/http"
	"time"

	"love-diary-go/internal/config"
	"love-diary-go/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthHandler 处理登录相关请求。
type AuthHandler struct {
	users  *repository.UserRepo
	secret string
}

// NewAuthHandler 创建登录处理器。
func NewAuthHandler(users *repository.UserRepo, cfg *config.Config) *AuthHandler {
	return &AuthHandler{users: users, secret: cfg.JWTSecret}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login POST /auth/login — 校验 users 表账号密码，签发 7 天 JWT。
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	user, err := h.users.FindByUsername(c.Request.Context(), req.Username)
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
		"username":   user.Username,
		"isLoggedIn": true,
		"exp":        time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(h.secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "登录成功",
		"isLoggedIn": true,
		"token":      tokenStr,
		"expiresIn":  "7d", // 前端 login store 用于计算 localStorage 过期时间
	})
}
