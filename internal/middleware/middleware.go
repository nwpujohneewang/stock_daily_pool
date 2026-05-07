package middleware

import (
	"net/http"
	"stock/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func AuthMiddleware(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" {
			c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"code":    1002,
				"message": "missing X-API-Key header",
			})
			c.Abort()
			return
		}
		if key != apiKey {
			c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"code":    1002,
				"message": "invalid API Key",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func IpLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		logger.Info("request", zap.String("ip", ip), zap.String("method", method), zap.String("path", path))
		c.Next()
	}
}
