package middleware

import (
	"net/http"
	"strings"

	"love-diary-go/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Auth 校验 Authorization: Bearer <token>，通过后把 username 写入上下文。
// 目前仅 /tree/water 需要鉴权；其它接口与旧版 Node 一样对游客开放读操作。
func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization header is required"})
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader || tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Token is required"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
			c.Abort()
			return
		}
		username, _ := claims["username"].(string)
		account, _ := claims["account"].(string)
		if account == "" {
			account = username
		}
		c.Set("account", account)
		c.Set("username", account)
		c.Next()
	}
}
