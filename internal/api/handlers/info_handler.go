package handlers

import (
	"net/http"

	"github.com/defskela/merchmarket/internal/domain/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type InfoHandler struct {
	db *gorm.DB
}

func NewInfoHandler(db *gorm.DB) *InfoHandler {
	return &InfoHandler{db: db}
}

type InventoryItem struct {
	Type     string `json:"type"`     // предмет
	Quantity int    `json:"quantity"` // количество предметов
}

type CoinHistoryEntry struct {
	FromUser string `json:"fromUser,omitempty"`
	ToUser   string `json:"toUser,omitempty"`
	Amount   int    `json:"amount"`
}

type InfoResponse struct {
	Coins       int                           `json:"coins"`       // количество монет
	Inventory   []InventoryItem               `json:"inventory"`   // инвентарь
	CoinHistory map[string][]CoinHistoryEntry `json:"coinHistory"` // история транзакций с монетами (ключи: received, sent)
}

func (h *InfoHandler) GetInfo(c *gin.Context) {
	// Получаем имя пользователя из контекста (установлено middleware)
	usernameI, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"errors": "Пользователь не авторизован"})
		return
	}
	username, ok := usernameI.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"errors": "Неверные данные авторизации"})
		return
	}

	// Извлекаем пользователя из БД с покупками и товарами
	var user models.User
	if err := h.db.Preload("Purchases.Merch").Where("username = ?", username).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": "Не удалось найти пользователя"})
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

	if err := h.db.Where("from_user_id = ?", user.ID).Find(&sentTxs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": "Не удалось получить отправленные транзакции"})
		return
	}
	if err := h.db.Where("to_user_id = ?", user.ID).Find(&receivedTxs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": "Не удалось получить полученные транзакции"})
		return
	}

	// Формируем историю транзакций
	coinHistory := make(map[string][]CoinHistoryEntry)

	for _, tx := range sentTxs {
		var toUser models.User
		if err := h.db.First(&toUser, tx.ToUserID).Error; err != nil {
			continue
		}
		coinHistory["sent"] = append(coinHistory["sent"], CoinHistoryEntry{
			ToUser: toUser.Username,
			Amount: tx.Amount,
		})
	}

	for _, tx := range receivedTxs {
		var fromUser models.User
		if err := h.db.First(&fromUser, tx.FromUserID).Error; err != nil {
			continue
		}
		coinHistory["received"] = append(coinHistory["received"], CoinHistoryEntry{
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
