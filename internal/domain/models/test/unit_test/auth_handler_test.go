package test

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
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB инициализирует in-memory БД для тестов.
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	err = db.AutoMigrate(&models.User{}, &models.Purchase{}, &models.Merch{}, &models.Transaction{})
	assert.NoError(t, err)
	return db
}

func performAuthRequest(handler *handlers.AuthHandler, requestBody interface{}) (*httptest.ResponseRecorder, error) {
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

	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Authenticate(c)
	return w, nil
}

func TestAuthenticate_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	handler := &handlers.AuthHandler{Db: db}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodPost, "/auth", bytes.NewBuffer([]byte("not a json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Authenticate(c)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	// Исправлен ключ ошибки "errors" → "error"
	assert.Equal(t, "Неверный запрос", resp["error"])
}

func TestAuthenticate_NewUser(t *testing.T) {
	db := setupTestDB(t)
	handler := &handlers.AuthHandler{Db: db}

	reqBody := handlers.AuthRequest{Username: "newuser", Password: "password123"}
	w, err := performAuthRequest(handler, reqBody)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp handlers.AuthResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Token)

	var user models.User
	err = db.Where("username = ?", "newuser").First(&user).Error
	assert.NoError(t, err)
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("password123"))
	assert.NoError(t, err)
	assert.Equal(t, 1000, user.Coins)
}

func TestAuthenticate_ExistingUser_ValidPassword(t *testing.T) {
	db := setupTestDB(t)
	handler := &handlers.AuthHandler{Db: db}

	password := "password123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := models.User{Username: "existinguser", Password: string(hash), Coins: 1000}
	db.Create(&user)

	reqBody := handlers.AuthRequest{Username: "existinguser", Password: password}
	w, err := performAuthRequest(handler, reqBody)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp handlers.AuthResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
}

func TestAuthenticate_ExistingUser_InvalidPassword(t *testing.T) {
	db := setupTestDB(t)
	handler := &handlers.AuthHandler{Db: db}

	password := "correctpassword"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := models.User{Username: "existinguser", Password: string(hash), Coins: 1000}
	db.Create(&user)

	reqBody := handlers.AuthRequest{Username: "existinguser", Password: "wrongpassword"}
	w, err := performAuthRequest(handler, reqBody)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	// Исправлен ключ ошибки "errors" → "error"
	assert.Equal(t, "Неверный пароль", resp["error"])
}
