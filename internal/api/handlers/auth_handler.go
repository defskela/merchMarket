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
	Db *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{Db: db}
}

type AuthRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

var jwtSecret = []byte(os.Getenv("JWT_SECRET_KEY"))

// @Summary      Аутентификация и получение JWT-токена.
// @Description  Регистрирует нового пользователя или авторизует существующего. Возвращает JWT-токен.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body AuthRequest true "Логин и пароль"
// @Success      200 {object} AuthResponse "Успешная аутентификация."
// @Failure      400 {object} ErrorResponse "Неверный запрос."
// @Failure      401 {object} ErrorResponse "Неавторизован."
// @Failure      500 {object} ErrorResponse "Внутренняя ошибка сервера."
// @Router       /auth [post]
// @Security     BearerAuth
func (h *AuthHandler) Authenticate(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := ErrorResponse{Error: "Неверный запрос"}
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	var user models.User
	err := h.Db.Where("username = ?", req.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			var hash []byte
			hash, err = bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				resp := ErrorResponse{Error: "Ошибка при генерации хеша"}
				c.JSON(http.StatusInternalServerError, resp)
				return
			}
			user = models.User{
				Username: req.Username,
				Password: string(hash),
				Coins:    1000,
			}
			if err := h.Db.Create(&user).Error; err != nil {
				resp := ErrorResponse{Error: "Не удалось создать пользователя"}
				c.JSON(http.StatusInternalServerError, resp)
				return
			}
		} else {
			// fmt.Println(err)
			resp := ErrorResponse{Error: "Ошибка при поиске пользователя"}
			c.JSON(http.StatusInternalServerError, resp)
			return
		}
	} else {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			resp := ErrorResponse{Error: "Неверный пароль"}
			c.JSON(http.StatusUnauthorized, resp)
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
		resp := ErrorResponse{Error: "Не удалось создать токен"}
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	c.JSON(http.StatusOK, AuthResponse{Token: tokenString})
}
