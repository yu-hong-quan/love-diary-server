package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// LimitBody 限制请求体大小，避免超大 base64 JSON 占满内存或触发网关超时。
func LimitBody(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
