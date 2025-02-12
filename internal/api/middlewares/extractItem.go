package middlewares

import "github.com/gin-gonic/gin"

// ExtractItemMiddleware извлекает параметр item из запроса и сохраняет его в контексте.
func ExtractItemMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		item := c.Param("item")
		if item != "" {
			// Сохраняем item в контексте
			c.Set("item", item)
		}
		c.Next()
	}
}
