package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
