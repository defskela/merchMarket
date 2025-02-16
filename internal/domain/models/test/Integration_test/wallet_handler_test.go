package integrationtest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/defskela/merchmarket/internal/api/handlers"
	"github.com/defskela/merchmarket/internal/domain/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestSendCoinIntegration проверяет сценарий перевода монет:
//
//	Создаются отправитель и получатель.
//	Имитируется middleware, который устанавливает отправителя (из JWT).
//	Выполняется POST-запрос на endpoint /sendCoin.
//	Проверяется, что баланс отправителя уменьшился, получателя увеличился,
//	а запись о транзакции создана в базе.
func TestSendCoinIntegration(t *testing.T) {
	db := setupTestDB(t)

	// Создаём отправителя и получателя.
	sender := models.User{Username: "sender", Coins: 100}
	receiver := models.User{Username: "receiver", Coins: 50}
	err := db.Create(&sender).Error
	assert.NoError(t, err)
	err = db.Create(&receiver).Error
	assert.NoError(t, err)

	router := gin.Default()

	// Имитируем middleware для установки имени отправителя
	router.Use(func(c *gin.Context) {
		c.Set("username", "sender")
		c.Next()
	})

	walletHandler := handlers.NewWalletHandler(db)
	router.POST("/sendCoin", walletHandler.SendCoin)

	payload := handlers.SendCoinRequest{
		ToUser: "receiver",
		Amount: 30,
	}
	payloadBytes, err := json.Marshal(payload)
	assert.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/sendCoin", bytes.NewBuffer(payloadBytes))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем, что в ответе содержится сообщение об успешном переводе.
	// var resp map[string]string
	// err = json.Unmarshal(w.Body.Bytes(), &resp)
	// assert.NoError(t, err)
	// assert.Contains(t, resp["message"], "Монетки успешно отправлены")

	// Проверяем, что баланс отправителя уменьшился (100 - 30 = 70).
	var updatedSender models.User
	err = db.Where("username = ?", "sender").First(&updatedSender).Error
	assert.NoError(t, err)
	assert.Equal(t, 70, updatedSender.Coins)

	// Проверяем, что баланс получателя увеличился (50 + 30 = 80).
	var updatedReceiver models.User
	err = db.Where("username = ?", "receiver").First(&updatedReceiver).Error
	assert.NoError(t, err)
	assert.Equal(t, 80, updatedReceiver.Coins)

	// Проверяем, что создана запись о транзакции.
	var transaction models.Transaction
	err = db.Where("from_user_id = ? AND to_user_id = ? AND amount = ?", sender.ID, receiver.ID, 30).First(&transaction).Error
	assert.NoError(t, err)
}
