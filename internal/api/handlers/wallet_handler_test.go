package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/defskela/merchmarket/internal/domain/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func performSendCoinRequest(handler *WalletHandler, requestBody interface{}, username interface{}) (*httptest.ResponseRecorder, error) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var body []byte
	var err error
	if requestBody != nil {
		body, err = json.Marshal(requestBody)
		if err != nil {
			return nil, err
		}
	}
	c.Request = httptest.NewRequest(http.MethodPost, "/sendCoin", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if username != nil {
		c.Set("username", username)
	}
	handler.SendCoin(c)
	return w, nil
}

func TestSendCoin_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	handler := NewWalletHandler(db)

	// Передаём невалидный JSON (при маршалинге строка обернется в кавычки, но структура не соответствует SendCoinRequest)
	w, err := performSendCoinRequest(handler, "not a json", "sender")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Неверный запрос", resp["error"])
}

func TestSendCoin_Unauthorized(t *testing.T) {
	db := setupTestDB(t)
	handler := NewWalletHandler(db)

	reqBody := SendCoinRequest{ToUser: "receiver", Amount: 100}
	// Не устанавливаем username в контексте
	w, err := performSendCoinRequest(handler, reqBody, nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Пользователь не авторизован", resp["error"])
}

func TestSendCoin_InvalidAmount(t *testing.T) {
	db := setupTestDB(t)
	handler := NewWalletHandler(db)

	// Сумма перевода отрицательная
	reqBody := SendCoinRequest{ToUser: "receiver", Amount: -1}
	w, err := performSendCoinRequest(handler, reqBody, "sender")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Сумма перевода должна быть положительной", resp["error"])
}

func TestSendCoin_SenderNotFound(t *testing.T) {
	db := setupTestDB(t)
	handler := NewWalletHandler(db)

	// Создаём получателя, но не создаём отправителя
	receiver := models.User{Username: "receiver", Coins: 100}
	err := db.Create(&receiver).Error
	assert.NoError(t, err)

	reqBody := SendCoinRequest{ToUser: "receiver", Amount: 50}
	// Отправитель с именем "nonexistentSender" отсутствует в БД
	w, err := performSendCoinRequest(handler, reqBody, "nonexistentSender")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Ошибка при получении отправителя", resp["error"])
}

func TestSendCoin_NotEnoughCoins(t *testing.T) {
	db := setupTestDB(t)
	handler := NewWalletHandler(db)

	// Создаём отправителя с недостаточным балансом
	sender := models.User{Username: "sender", Coins: 30}
	err := db.Create(&sender).Error
	assert.NoError(t, err)

	// Создаём получателя
	receiver := models.User{Username: "receiver", Coins: 100}
	err = db.Create(&receiver).Error
	assert.NoError(t, err)

	reqBody := SendCoinRequest{ToUser: "receiver", Amount: 50}
	w, err := performSendCoinRequest(handler, reqBody, "sender")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Недостаточно средств", resp["error"])
}

func TestSendCoin_ReceiverNotFound(t *testing.T) {
	db := setupTestDB(t)
	handler := NewWalletHandler(db)

	// Создаём отправителя с достаточным балансом
	sender := models.User{Username: "sender", Coins: 100}
	err := db.Create(&sender).Error
	assert.NoError(t, err)

	// Получатель отсутствует в БД
	reqBody := SendCoinRequest{ToUser: "nonexistentReceiver", Amount: 50}
	w, err := performSendCoinRequest(handler, reqBody, "sender")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Получатель не найден", resp["error"])
}

func TestSendCoin_Success(t *testing.T) {
	db := setupTestDB(t)
	handler := NewWalletHandler(db)

	// Создаём отправителя и получателя с начальным балансом
	sender := models.User{Username: "sender", Coins: 100}
	err := db.Create(&sender).Error
	assert.NoError(t, err)
	receiver := models.User{Username: "receiver", Coins: 100}
	err = db.Create(&receiver).Error
	assert.NoError(t, err)

	reqBody := SendCoinRequest{ToUser: "receiver", Amount: 50}
	w, err := performSendCoinRequest(handler, reqBody, "sender")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["message"], "Монетки успешно отправлены")

	// Проверяем, что баланс отправителя уменьшился
	var updatedSender models.User
	err = db.Where("username = ?", "sender").First(&updatedSender).Error
	assert.NoError(t, err)
	assert.Equal(t, 50, updatedSender.Coins)

	// Проверяем, что баланс получателя увеличился
	var updatedReceiver models.User
	err = db.Where("username = ?", "receiver").First(&updatedReceiver).Error
	assert.NoError(t, err)
	assert.Equal(t, 150, updatedReceiver.Coins)

	// Проверяем, что транзакция записана
	var transaction models.Transaction
	err = db.Where("from_user_id = ? AND to_user_id = ? AND amount = ?", sender.ID, receiver.ID, 50).First(&transaction).Error
	assert.NoError(t, err)
}
