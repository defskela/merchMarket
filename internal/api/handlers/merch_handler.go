package handlers

import (
	"net/http"

	"github.com/defskela/merchmarket/internal/domain/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MerchHandler struct {
	db *gorm.DB
}

func NewMerchHandler(db *gorm.DB) *MerchHandler {
	return &MerchHandler{db: db}
}

// @Summary      Купить предмет за монеты.
// @Description  Списывает монеты и добавляет предмет в инвентарь.
// @Tags         Merch
// @Produce      json
// @Param        item path string true "Название товара"
// @Success 	 200 {null} nil "Успешный ответ"
// @Failure      400 {object} ErrorResponse "Неверный запрос."
// @Failure      401 {object} ErrorResponse "Неавторизован."
// @Failure      500 {object} ErrorResponse "Внутренняя ошибка сервера."
// @Router       /buy/{item} [get]
// @Security     BearerAuth
func (h *MerchHandler) BuyItem(c *gin.Context) {
	item := c.Param("item")
	if item == "" {
		resp := ErrorResponse{Error: "Не указан предмет для покупки"}
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	// Получаем имя пользователя из JWT
	usernameI, exists := c.Get("username")
	if !exists {
		resp := ErrorResponse{Error: "Пользователь не авторизован"}
		c.JSON(http.StatusUnauthorized, resp)
		return
	}
	username, ok := usernameI.(string)
	if !ok {
		resp := ErrorResponse{Error: "Ошибка авторизации"}
		c.JSON(http.StatusUnauthorized, resp)
		return
	}

	// Получаем информацию о товаре
	var merch models.Merch
	if err := h.db.Where("name = ?", item).First(&merch).Error; err != nil {
		resp := ErrorResponse{Error: "Товар не найден"}
		c.JSON(http.StatusNotFound, resp)
		return
	}

	// Начинаем транзакцию
	tx := h.db.Begin()

	// Получаем пользователя
	var user models.User
	if err := tx.Where("username = ?", username).First(&user).Error; err != nil {
		tx.Rollback()
		resp := ErrorResponse{Error: "Пользователь не найден"}
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	// Проверяем баланс
	if user.Coins < merch.Price {
		tx.Rollback()
		resp := ErrorResponse{Error: "Недостаточно средств"}
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	// Обновляем баланс пользователя
	if err := tx.Model(&user).Update("coins", user.Coins-merch.Price).Error; err != nil {
		tx.Rollback()
		resp := ErrorResponse{Error: "Ошибка при списании монет"}
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	// Создаем запись о покупке
	purchase := models.Purchase{
		UserID:  user.ID,
		MerchID: merch.ID,
	}
	if err := tx.Create(&purchase).Error; err != nil {
		tx.Rollback()
		resp := ErrorResponse{Error: "Ошибка при создании записи покупки"}
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	// Фиксируем транзакцию
	if err := tx.Commit().Error; err != nil {
		resp := ErrorResponse{Error: "Ошибка при сохранении данных"}
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	// c.JSON(http.StatusOK, gin.H{"message": "Предмет " + item + " успешно куплен"})
}
