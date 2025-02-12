package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/defskela/merchmarket/internal/api/handlers"
	"github.com/defskela/merchmarket/internal/api/middlewares"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB) {
	// Публичный эндпоинт для аутентификации
	authHandler := handlers.NewAuthHandler(db)
	router.POST("/api/auth", authHandler.Authenticate)

	// Защищённая группа – все остальные эндпоинты требуют JWT
	api := router.Group("/api")
	api.Use(middlewares.JWTAuthMiddleware())
	{
		// Получение информации о монетах, инвентаре и истории транзакций
		infoHandler := handlers.NewInfoHandler(db)
		api.GET("/info", infoHandler.GetInfo)

		// Отправка монет другому пользователю
		walletHandler := handlers.NewWalletHandler(db)
		api.POST("/sendCoin", walletHandler.SendCoin)

		// Покупка мерча – параметр item передаётся в пути
		merchHandler := handlers.NewMerchHandler(db)
		api.GET("/buy/:item", middlewares.ExtractItemMiddleware(), merchHandler.BuyItem)

	}
}
