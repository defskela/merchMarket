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

// @Summary      Покупка мерча
// @Description  Списывает монеты и добавляет предмет в инвентарь.
// @Tags         Merch
// @Produce      json
// @Param        item path string true "Название товара"
// @Success      200 {object} map[string]interface{}
// @Failure      400,401,404,500 {object} map[string]interface{}
// @Router       /buy/{item} [get]
// @Security     BearerAuth
func (h *MerchHandler) BuyItem(c *gin.Context) {
	item := c.Param("item")
	if item == "" {
		c.JSON(http.StatusBadRequest, gin.H{"errors": "Не указан предмет для покупки"})
		return
	}

	// Получаем имя пользователя из JWT
	usernameI, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"errors": "Пользователь не авторизован"})
		return
	}
	username, ok := usernameI.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"errors": "Ошибка авторизации"})
		return
	}

	// Получаем информацию о товаре
	var merch models.Merch
	if err := h.db.Where("name = ?", item).First(&merch).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"errors": "Товар не найден"})
		return
	}

	// Начинаем транзакцию
	tx := h.db.Begin()

	// Получаем пользователя
	var user models.User
	if err := tx.Where("username = ?", username).First(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"errors": "Ошибка при получении пользователя"})
		return
	}

	// Проверяем баланс
	if user.Coins < merch.Price {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"errors": "Недостаточно средств"})
		return
	}

	// Обновляем баланс пользователя
	if err := tx.Model(&user).Update("coins", user.Coins-merch.Price).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"errors": "Ошибка при списании монет"})
		return
	}

	// Создаем запись о покупке
	purchase := models.Purchase{
		UserID:  user.ID,
		MerchID: merch.ID,
	}
	if err := tx.Create(&purchase).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"errors": "Ошибка при создании записи покупки"})
		return
	}

	// Фиксируем транзакцию
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": "Ошибка при сохранении данных"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Предмет " + item + " успешно куплен"})
}
