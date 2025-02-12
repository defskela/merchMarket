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

// @Summary      Отправить монеты другому пользователю.
// @Description  Передаёт монеты от авторизованного пользователя другому.
// @Tags         Wallet
// @Accept       json
// @Produce      json
// @Param        sendCoin body SendCoinRequest true "Данные отправки монет"
// @Success      200 {object} "Успешный ответ."
// @Failure      400 {object} ErrorResponse "Неверный запрос."
// @Failure      401 {object} ErrorResponse "Неавторизован."
// @Failure      500 {object} ErrorResponse "Внутренняя ошибка сервера."
// @Router       /sendCoin [post]
// @Security     BearerAuth
func (h *WalletHandler) SendCoin(c *gin.Context) {
	var req SendCoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := ErrorResponse{Error: "Неверный запрос"}
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	// Получаем имя отправителя из JWT
	senderNameI, exists := c.Get("username")
	if !exists {
		resp := ErrorResponse{Error: "Пользователь не авторизован"}
		c.JSON(http.StatusUnauthorized, resp)
		return
	}
	senderName, ok := senderNameI.(string)
	if !ok {
		resp := ErrorResponse{Error: "Ошибка авторизации"}
		c.JSON(http.StatusUnauthorized, resp)
		return
	}

	// Проверяем, что сумма перевода положительная
	if req.Amount <= 0 {
		resp := ErrorResponse{Error: "Сумма перевода должна быть положительной"}
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	// Начинаем транзакцию
	tx := h.db.Begin()

	// Получаем отправителя
	var sender models.User
	if err := tx.Where("username = ?", senderName).First(&sender).Error; err != nil {
		tx.Rollback()
		resp := ErrorResponse{Error: "Ошибка при получении отправителя"}
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	// Проверяем баланс
	if sender.Coins < req.Amount {
		tx.Rollback()
		resp := ErrorResponse{Error: "Недостаточно средств"}
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	// Получаем получателя
	var receiver models.User
	if err := tx.Where("username = ?", req.ToUser).First(&receiver).Error; err != nil {
		tx.Rollback()
		resp := ErrorResponse{Error: "Получатель не найден"}
		c.JSON(http.StatusNotFound, resp)
		return
	}

	// Обновляем балансы
	if err := tx.Model(&sender).Update("coins", sender.Coins-req.Amount).Error; err != nil {
		tx.Rollback()
		resp := ErrorResponse{Error: "Ошибка при обновлении баланса отправителя"}
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	if err := tx.Model(&receiver).Update("coins", receiver.Coins+req.Amount).Error; err != nil {
		tx.Rollback()
		resp := ErrorResponse{Error: "Ошибка при обновлении баланса получателя"}
		c.JSON(http.StatusInternalServerError, resp)
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
		resp := ErrorResponse{Error: "Ошибка при создании записи транзакции"}
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	// Фиксируем транзакцию
	if err := tx.Commit().Error; err != nil {
		resp := ErrorResponse{Error: "Ошибка при сохранении транзакции"}
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Монетки успешно отправлены"})
}
