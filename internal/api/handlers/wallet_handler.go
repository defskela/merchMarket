package handlers

import (
	"net/http"

	"github.com/defskela/merchmarket/internal/domain/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WalletHandler struct {
	db *gorm.DB
}

func NewWalletHandler(db *gorm.DB) *WalletHandler {
	return &WalletHandler{db: db}
}

type SendCoinRequest struct {
	ToUser string `json:"toUser" binding:"required"`
	Amount int    `json:"amount" binding:"required"`
}

// @Summary      Отправка монет
// @Description  Передаёт монеты от авторизованного пользователя другому.
// @Tags         Wallet
// @Accept       json
// @Produce      json
// @Param        sendCoin body SendCoinRequest true "Данные отправки монет"
// @Success      200 {object} map[string]interface{}
// @Failure      400,401,404,500 {object} map[string]interface{}
// @Router       /sendCoin [post]
// @Security     BearerAuth
func (h *WalletHandler) SendCoin(c *gin.Context) {
	var req SendCoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errors": "Неверный запрос"})
		return
	}

	// Получаем имя отправителя из JWT
	senderNameI, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"errors": "Пользователь не авторизован"})
		return
	}
	senderName, ok := senderNameI.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"errors": "Ошибка авторизации"})
		return
	}

	// Проверяем, что сумма перевода положительная
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"errors": "Сумма перевода должна быть положительной"})
		return
	}

	// Начинаем транзакцию
	tx := h.db.Begin()

	// Получаем отправителя
	var sender models.User
	if err := tx.Where("username = ?", senderName).First(&sender).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"errors": "Ошибка при получении отправителя"})
		return
	}

	// Проверяем баланс
	if sender.Coins < req.Amount {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"errors": "Недостаточно средств"})
		return
	}

	// Получаем получателя
	var receiver models.User
	if err := tx.Where("username = ?", req.ToUser).First(&receiver).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"errors": "Получатель не найден"})
		return
	}

	// Обновляем балансы
	if err := tx.Model(&sender).Update("coins", sender.Coins-req.Amount).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"errors": "Ошибка при обновлении баланса отправителя"})
		return
	}
	if err := tx.Model(&receiver).Update("coins", receiver.Coins+req.Amount).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"errors": "Ошибка при обновлении баланса получателя"})
		return
	}

	// Записываем транзакцию
	transaction := models.Transaction{
		FromUserID: sender.ID,
		ToUserID:   receiver.ID,
		Amount:     req.Amount,
	}
	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"errors": "Ошибка при создании записи транзакции"})
		return
	}

	// Фиксируем транзакцию
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": "Ошибка при сохранении транзакции"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Монеты успешно отправлены"})
}
