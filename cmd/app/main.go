package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/defskela/merchmarket/docs"
	"github.com/defskela/merchmarket/internal/api/routes"
	"github.com/defskela/merchmarket/internal/domain/models"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// @title API Avito shop
// @version 1.0.0
// @description API для отбора на Стажировку в Авито
// @host localhost:8080
// @BasePath /api

// @host localhost:8080

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token

func initDB() (*gorm.DB, error) {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No .env file found, using defaults")
		return nil, err
	}
	connectionData := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_PORT"))

	db, err := gorm.Open(postgres.Open(connectionData), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	// Лишь для того, чтобы при запуске контейнера с новой БД всё работало, как надо. В обычной жизни эти строки не нужны
	if err := db.Migrator().CreateTable(models.Merch{}, models.Purchase{}, models.Transaction{}, models.User{}); err != nil {
		fmt.Printf("Ошибка при создании таблиц: %v", err)
	}

	if err := db.AutoMigrate(models.Merch{}, models.Purchase{}, models.Transaction{}, models.User{}); err != nil {
		fmt.Printf("Ошибка при миграции: %v", err)
	}
	db.Create(&models.Merch{Name: "t-shirt", Price: 80})
	db.Create(&models.Merch{Name: "cup", Price: 20})
	db.Create(&models.Merch{Name: "book", Price: 50})
	db.Create(&models.Merch{Name: "pen", Price: 10})
	db.Create(&models.Merch{Name: "powerbank", Price: 200})
	db.Create(&models.Merch{Name: "hoody", Price: 300})
	db.Create(&models.Merch{Name: "umbrella", Price: 200})
	db.Create(&models.Merch{Name: "socks", Price: 10})
	db.Create(&models.Merch{Name: "wallet", Price: 50})
	db.Create(&models.Merch{Name: "pink-hoody", Price: 500})
	return db, nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using defaults")
	}

	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	router := gin.Default()

	routes.SetupRoutes(router, db)

	err = router.Run(":8080")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
