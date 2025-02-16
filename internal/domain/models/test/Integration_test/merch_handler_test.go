package integrationtest

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

// setupTestDB инициализирует in-memory БД для теста.
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	err = db.AutoMigrate(&models.User{}, &models.Merch{}, &models.Purchase{}, models.Transaction{})
	assert.NoError(t, err)
	return db
}

// TestBuyMerchIntegration имитирует сценарий покупки мерча:
// Создаются тестовый пользователь и товар.
// Имитируется middleware, который устанавливает имя пользователя (например, из JWT).
// Выполняется HTTP-запрос к эндпоинту покупки.
// Проверяется корректность ответа, обновление баланса и создание записи о покупке.
func TestBuyMerchIntegration(t *testing.T) {
	db := setupTestDB(t)

	user := models.User{Username: "testuser", Coins: 100}
	err := db.Create(&user).Error
	assert.NoError(t, err)

	merch := models.Merch{Name: "cup", Price: 20}
	err = db.Create(&merch).Error
	assert.NoError(t, err)

	router := gin.Default()

	// Имитация middleware для установки имени пользователя
	router.Use(func(c *gin.Context) {
		c.Set("username", "testuser")
		c.Next()
	})

	merchHandler := handlers.NewMerchHandler(db)
	router.GET("/buy/:item", merchHandler.BuyItem)

	// Выполняем запрос на покупку товара "cup".
	req, _ := http.NewRequest(http.MethodGet, "/buy/cup", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем, что в ответе содержится сообщение об успешной покупке.
	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["message"], "успешно куплен")

	var updatedUser models.User
	err = db.Where("username = ?", "testuser").First(&updatedUser).Error
	assert.NoError(t, err)
	assert.Equal(t, 80, updatedUser.Coins)

	// Проверяем, что создана запись о покупке.
	var purchase models.Purchase
	err = db.Where("user_id = ? AND merch_id = ?", user.ID, merch.ID).First(&purchase).Error
	assert.NoError(t, err)
}
