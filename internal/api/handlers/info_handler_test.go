package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/defskela/merchmarket/internal/domain/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestValues(t *testing.T) (*gorm.DB, *InfoHandler, *httptest.ResponseRecorder, *gin.Context) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	err = db.AutoMigrate(&models.User{}, &models.Purchase{}, &models.Merch{}, &models.Transaction{})
	assert.NoError(t, err)
	handler := &InfoHandler{db: db}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	return db, handler, w, c
}

// Отсутствует имя пользователя в контексте.
func TestGetInfo_NoUsername(t *testing.T) {
	_, handler, w, c := setupTestValues(t)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/info", nil)

	handler.GetInfo(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Пользователь не авторизован", resp.Error)
}

// Имя пользователя имеет неверный тип.
func TestGetInfo_InvalidUsernameType(t *testing.T) {
	_, handler, w, c := setupTestValues(t)

	// Устанавливаем "username" не как строку
	c.Set("username", 123)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/info", nil)

	handler.GetInfo(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	resp, exists := response["errors"]
	assert.True(t, exists)
	assert.Equal(t, "Пользователь не авторизован", resp.Error)
}

// Пользователь не найден в базе.
func TestGetInfo_UserNotFound(t *testing.T) {
	_, handler, w, c := setupTestValues(t)
	c.Set("username", "nonexistent")
	c.Request = httptest.NewRequest(http.MethodGet, "/api/info", nil)

	handler.GetInfo(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	resp, exists := response["errors"]
	assert.True(t, exists)
	assert.Equal(t, "Не удалось найти пользователя", resp.Error)
}

// Успешное получение информации (happy path).
func TestGetInfo_HappyPath(t *testing.T) {
	db, handler, w, c := setupTestValues(t)

	// Создаём пользователя "testuser" с монетами 2000
	user := models.User{
		Username: "testuser",
		Coins:    2000,
	}
	err := db.Create(&user).Error
	assert.NoError(t, err)

	// Создаём товары (merch)
	merch1 := models.Merch{Name: "item1"}
	merch2 := models.Merch{Name: "item2"}
	err = db.Create(&merch1).Error
	assert.NoError(t, err)
	err = db.Create(&merch2).Error
	assert.NoError(t, err)

	// Создаём покупки для пользователя:
	// Две покупки для "item1" и одна для "item2"
	purchase1 := models.Purchase{UserID: user.ID, MerchID: merch1.ID}
	purchase2 := models.Purchase{UserID: user.ID, MerchID: merch1.ID}
	purchase3 := models.Purchase{UserID: user.ID, MerchID: merch2.ID}
	err = db.Create(&purchase1).Error
	assert.NoError(t, err)
	err = db.Create(&purchase2).Error
	assert.NoError(t, err)
	err = db.Create(&purchase3).Error
	assert.NoError(t, err)

	// Создаём дополнительных пользователей для транзакций.
	recipient := models.User{Username: "recipient", Coins: 500}
	sender := models.User{Username: "sender", Coins: 300}
	err = db.Create(&recipient).Error
	assert.NoError(t, err)
	err = db.Create(&sender).Error
	assert.NoError(t, err)

	// Создаём транзакции:
	// Отправленная транзакция: от testuser к recipient на сумму 100
	tx1 := models.Transaction{FromUserID: user.ID, ToUserID: recipient.ID, Amount: 100}
	// Полученная транзакция: от sender к testuser на сумму 50
	tx2 := models.Transaction{FromUserID: sender.ID, ToUserID: user.ID, Amount: 50}
	err = db.Create(&tx1).Error
	assert.NoError(t, err)
	err = db.Create(&tx2).Error
	assert.NoError(t, err)

	c.Set("username", "testuser")
	c.Request = httptest.NewRequest(http.MethodGet, "/api/info", nil)

	handler.GetInfo(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp InfoResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	// Проверяем количество монет
	assert.Equal(t, 2000, resp.Coins)

	// Проверяем инвентарь: должно быть 2 единицы "item1" и 1 единица "item2"
	var item1Found, item2Found bool
	for _, item := range resp.Inventory {
		if item.Type == "item1" {
			assert.Equal(t, 2, item.Quantity)
			item1Found = true
		}
		if item.Type == "item2" {
			assert.Equal(t, 1, item.Quantity)
			item2Found = true
		}
	}
	assert.True(t, item1Found, "В инвентаре должен присутствовать item1")
	assert.True(t, item2Found, "В инвентаре должен присутствовать item2")

	// Проверяем историю транзакций:
	// Отправленная транзакция должна быть к пользователю "recipient" на сумму 100,
	// Полученная транзакция – от пользователя "sender" на сумму 50.
	var sentFound, receivedFound bool
	for _, sent := range resp.CoinHistory.Sent {
		if sent.ToUser == "recipient" {
			assert.Equal(t, 100, sent.Amount)
			sentFound = true
		}
	}
	for _, rec := range resp.CoinHistory.Received {
		if rec.FromUser == "sender" {
			assert.Equal(t, 50, rec.Amount)
			receivedFound = true
		}
	}
	assert.True(t, sentFound, "Отправленная транзакция к recipient должна присутствовать")
	assert.True(t, receivedFound, "Полученная транзакция от sender должна присутствовать")
}
