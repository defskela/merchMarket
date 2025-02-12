package handlers

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/defskela/merchmarket/internal/domain/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	db *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

type AuthRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

var jwtSecret = []byte(os.Getenv("JWT_SECRET_KEY"))

func (h *AuthHandler) Authenticate(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errors": "Неверный запрос"})
		return
	}

	var user models.User
	err := h.db.Where("username = ?", req.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"errors": "Ошибка при генерации хеша"})
				return
			}
			user = models.User{
				Username: req.Username,
				Password: string(hash),
				Coins:    1000,
			}
			if err := h.db.Create(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"errors": "Не удалось создать пользователя"})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Ошибка при поиске пользователя"})
			return
		}
	} else {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"errors": "Неверный пароль"})
			return
		}
	}

	// Создаём JWT-токен с данными пользователя.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": req.Username,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": "Не удалось создать токен"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{Token: tokenString})
}
