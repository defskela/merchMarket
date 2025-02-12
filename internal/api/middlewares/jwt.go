package middlewares

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET_KEY"))

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"errors": "Требуется заголовок Authorization"})
			c.Abort()
			return
		}

		// Ожидаем формат: Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"errors": "Неверный формат заголовка Authorization"})
			c.Abort()
			return
		}

		tokenStr := parts[1]
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"errors": "Неверный или просроченный токен"})
			c.Abort()
			return
		}

		// При необходимости можно сохранить данные токена в контексте (например, username)
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("username", claims["username"])
		}

		c.Next()
	}
}
