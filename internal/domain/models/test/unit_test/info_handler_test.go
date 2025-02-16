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
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestValues(t *testing.T) (*gorm.DB, *handlers.InfoHandler, *httptest.ResponseRecorder, *gin.Context) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	err = db.AutoMigrate(&models.User{}, &models.Purchase{}, &models.Merch{}, &models.Transaction{})
	assert.NoError(t, err)

	handler := &handlers.InfoHandler{Db: db}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	return db, handler, w, c
}

func TestGetInfo_NoUsername(t *testing.T) {
	_, handler, w, c := setupTestValues(t)
	c.Request = httptest.NewRequest(http.MethodGet, "/info", nil)

	handler.GetInfo(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp handlers.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Пользователь не авторизован", resp.Error)
}

func TestGetInfo_InvalidUsernameType(t *testing.T) {
	_, handler, w, c := setupTestValues(t)
	c.Set("username", 123)
	c.Request = httptest.NewRequest(http.MethodGet, "/info", nil)

	handler.GetInfo(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp handlers.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Пользователь не авторизован", resp.Error)
}

func TestGetInfo_UserNotFound(t *testing.T) {
	_, handler, w, c := setupTestValues(t)
	c.Set("username", "nonexistent")
	c.Request = httptest.NewRequest(http.MethodGet, "/info", nil)

	handler.GetInfo(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp handlers.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Не удалось найти пользователя", resp.Error)
}

func TestGetInfo_HappyPath(t *testing.T) {
	db, handler, w, c := setupTestValues(t)

	user := models.User{Username: "testuser", Coins: 2000}
	assert.NoError(t, db.Create(&user).Error)

	merch1 := models.Merch{Name: "item1"}
	merch2 := models.Merch{Name: "item2"}
	assert.NoError(t, db.Create(&merch1).Error)
	assert.NoError(t, db.Create(&merch2).Error)

	purchases := []models.Purchase{
		{UserID: user.ID, MerchID: merch1.ID},
		{UserID: user.ID, MerchID: merch1.ID},
		{UserID: user.ID, MerchID: merch2.ID},
	}
	for _, purchase := range purchases {
		assert.NoError(t, db.Create(&purchase).Error)
	}

	recipient := models.User{Username: "recipient", Coins: 500}
	sender := models.User{Username: "sender", Coins: 300}
	assert.NoError(t, db.Create(&recipient).Error)
	assert.NoError(t, db.Create(&sender).Error)

	transactions := []models.Transaction{
		{FromUserID: user.ID, ToUserID: recipient.ID, Amount: 100},
		{FromUserID: sender.ID, ToUserID: user.ID, Amount: 50},
	}
	for _, tx := range transactions {
		assert.NoError(t, db.Create(&tx).Error)
	}

	c.Set("username", "testuser")
	c.Request = httptest.NewRequest(http.MethodGet, "/info", nil)

	handler.GetInfo(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp handlers.InfoResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 2000, resp.Coins)

	expectedInventory := map[string]int{"item1": 2, "item2": 1}
	for _, item := range resp.Inventory {
		assert.Equal(t, expectedInventory[item.Type], item.Quantity)
	}

	expectedSent := map[string]int{"recipient": 100}
	expectedReceived := map[string]int{"sender": 50}

	for _, sent := range resp.CoinHistory.Sent {
		assert.Equal(t, expectedSent[sent.ToUser], sent.Amount)
	}
	for _, rec := range resp.CoinHistory.Received {
		assert.Equal(t, expectedReceived[rec.FromUser], rec.Amount)
	}
}
