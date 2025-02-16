package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/defskela/merchmarket/internal/api/handlers"
	"github.com/defskela/merchmarket/internal/domain/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func performBuyItemRequest(handler *handlers.MerchHandler, item string, username interface{}) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "item", Value: item}}
	if username != nil {
		c.Set("username", username)
	}
	handler.BuyItem(c)
	return w
}

func TestBuyItem_MissingItem(t *testing.T) {
	db := setupTestDB(t)
	handler := handlers.NewMerchHandler(db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Не устанавливаем параметр "item"
	c.Set("username", "testuser")
	handler.BuyItem(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Не указан предмет для покупки", resp["error"])
}

func TestBuyItem_Unauthorized(t *testing.T) {
	db := setupTestDB(t)
	handler := handlers.NewMerchHandler(db)

	// Передаём корректный параметр, но не устанавливаем "username"
	w := performBuyItemRequest(handler, "cup", nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Пользователь не авторизован", resp["error"])
}

func TestBuyItem_ItemNotFound(t *testing.T) {
	db := setupTestDB(t)
	handler := handlers.NewMerchHandler(db)

	// Пользователь авторизован, но товара "cup" нет в БД
	w := performBuyItemRequest(handler, "cup", "testuser")

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Товар не найден", resp["error"])
}

func TestBuyItem_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	// Создаём товар, чтобы он был найден
	merch := models.Merch{Name: "cup", Price: 20}
	err := db.Create(&merch).Error
	assert.NoError(t, err)

	handler := handlers.NewMerchHandler(db)
	// Пытаемся купить товар от несуществующего пользователя
	w := performBuyItemRequest(handler, "cup", "nonexistent")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Пользователь не найден", resp["error"])
}

func TestBuyItem_NotEnoughCoins(t *testing.T) {
	db := setupTestDB(t)
	// Создаём товар с ценой 20 монет
	merch := models.Merch{Name: "cup", Price: 20}
	err := db.Create(&merch).Error
	assert.NoError(t, err)
	// Создаём пользователя с недостаточным балансом
	user := models.User{Username: "testuser", Coins: 10}
	err = db.Create(&user).Error
	assert.NoError(t, err)

	handler := handlers.NewMerchHandler(db)
	w := performBuyItemRequest(handler, "cup", "testuser")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Недостаточно средств", resp["error"])
}

func TestBuyItem_Success(t *testing.T) {
	db := setupTestDB(t)
	// Создаём товар и пользователя с достаточным балансом
	merch := models.Merch{Name: "cup", Price: 20}
	err := db.Create(&merch).Error
	assert.NoError(t, err)
	user := models.User{Username: "testuser", Coins: 100}
	err = db.Create(&user).Error
	assert.NoError(t, err)

	handler := handlers.NewMerchHandler(db)
	w := performBuyItemRequest(handler, "cup", "testuser")

	assert.Equal(t, http.StatusOK, w.Code)
	// var resp map[string]string
	// err = json.Unmarshal(w.Body.Bytes(), &resp)
	// assert.NoError(t, err)
	// assert.Contains(t, resp["message"], "успешно куплен")

	// Проверяем, что баланс пользователя уменьшился на стоимость товара
	var updatedUser models.User
	err = db.Where("username = ?", "testuser").First(&updatedUser).Error
	assert.NoError(t, err)
	assert.Equal(t, 80, updatedUser.Coins)

	// Проверяем, что создана запись о покупке
	var purchase models.Purchase
	err = db.Where("user_id = ? AND merch_id = ?", user.ID, merch.ID).First(&purchase).Error
	assert.NoError(t, err)
}
