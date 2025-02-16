package handlers

import (
	"net/http"

	"github.com/defskela/merchmarket/internal/domain/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type InfoHandler struct {
	Db *gorm.DB
}

func NewInfoHandler(db *gorm.DB) *InfoHandler {
	return &InfoHandler{Db: db}
}

type InfoResponse struct {
	Coins       int             `json:"coins"`
	Inventory   []InventoryItem `json:"inventory"`
	CoinHistory CoinHistory     `json:"coinHistory"`
}

type InventoryItem struct {
	Type     string `json:"type"`
	Quantity int    `json:"quantity"`
}

type CoinHistory struct {
	Received []CoinHistoryEntry `json:"received"`
	Sent     []CoinHistoryEntry `json:"sent"`
}

type CoinHistoryEntry struct {
	FromUser string `json:"fromUser,omitempty"`
	ToUser   string `json:"toUser,omitempty"`
	Amount   int    `json:"amount"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// @Summary      Получить информацию о монетах, инвентаре и истории транзакций.
// @Description  Возвращает баланс монет, инвентарь и список транзакций.
// @Tags         Info
// @Produce      json
// @Success      200 {object} InfoResponse "Успешный ответ."
// @Failure      400 {object} ErrorResponse "Неверный запрос."
// @Failure      401 {object} ErrorResponse "Неавторизован."
// @Failure      500 {object} ErrorResponse "Внутренняя ошибка сервера."
// @Router       /info [get]
// @Security     BearerAuth
func (h *InfoHandler) GetInfo(c *gin.Context) {
	// Получаем имя пользователя из контекста (установлено middleware)
	usernameI, exists := c.Get("username")
	if !exists {
		resp := ErrorResponse{Error: "Пользователь не авторизован"}
		c.JSON(http.StatusUnauthorized, resp)
		return
	}
	username, ok := usernameI.(string)
	if !ok {
		resp := ErrorResponse{Error: "Пользователь не авторизован"}
		c.JSON(http.StatusUnauthorized, gin.H{"errors": resp})
		return
	}

	// Извлекаем пользователя из БД с покупками и товарами
	var user models.User
	if err := h.Db.Preload("Purchases.Merch").Where("username = ?", username).First(&user).Error; err != nil {
		resp := ErrorResponse{Error: "Не удалось найти пользователя"}
		c.JSON(http.StatusInternalServerError, gin.H{"errors": resp})
		return
	}

	// Формируем инвентарь на основе покупок
	inventoryMap := make(map[string]int)
	for _, purchase := range user.Purchases {
		inventoryMap[purchase.Merch.Name]++
	}
	var inventory []InventoryItem
	for merchName, quantity := range inventoryMap {
		inventory = append(inventory, InventoryItem{
			Type:     merchName,
			Quantity: quantity,
		})
	}

	// Получаем переводы: отправленные и полученные
	var sentTxs []models.Transaction
	var receivedTxs []models.Transaction

	if err := h.Db.Where("from_user_id = ?", user.ID).Find(&sentTxs).Error; err != nil {
		resp := ErrorResponse{Error: "Не удалось получить отправленные транзакции"}
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	if err := h.Db.Where("to_user_id = ?", user.ID).Find(&receivedTxs).Error; err != nil {
		resp := ErrorResponse{Error: "Не удалось получить полученные транзакции"}
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	// Формируем историю транзакций
	coinHistory := CoinHistory{
		Received: []CoinHistoryEntry{},
		Sent:     []CoinHistoryEntry{},
	}

	for _, tx := range sentTxs {
		var toUser models.User
		if err := h.Db.First(&toUser, tx.ToUserID).Error; err != nil {
			continue
		}
		coinHistory.Sent = append(coinHistory.Sent, CoinHistoryEntry{
			ToUser: toUser.Username,
			Amount: tx.Amount,
		})
	}

	for _, tx := range receivedTxs {
		var fromUser models.User
		if err := h.Db.First(&fromUser, tx.FromUserID).Error; err != nil {
			continue
		}
		coinHistory.Received = append(coinHistory.Received, CoinHistoryEntry{
			FromUser: fromUser.Username,
			Amount:   tx.Amount,
		})
	}

	resp := InfoResponse{
		Coins:       user.Coins,
		Inventory:   inventory,
		CoinHistory: coinHistory,
	}
	c.JSON(http.StatusOK, resp)
}
