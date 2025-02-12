package models

import "gorm.io/gorm"

// User представляет сотрудника системы.
type User struct {
	gorm.Model
	Username  string     `gorm:"unique;not null" json:"username"`
	Password  string     `gorm:"not null" json:"-"` // хранится в виде хэша
	Coins     int        `gorm:"not null;default:1000" json:"coins"`
	Purchases []Purchase `json:"purchases"`
}

// Merch представляет товар в магазине.
type Merch struct {
	gorm.Model
	Name  string `gorm:"unique;not null" json:"name"`
	Price int    `gorm:"not null" json:"price"`
}

// Purchase фиксирует покупку мерча пользователем.
type Purchase struct {
	gorm.Model
	UserID  uint  `gorm:"not null" json:"userId"`
	MerchID uint  `gorm:"not null" json:"merchId"`
	Merch   Merch `gorm:"foreignKey:MerchID" json:"merch"`
}

// Transaction представляет перевод монет между пользователями.
type Transaction struct {
	gorm.Model
	FromUserID uint `gorm:"not null" json:"fromUserId"`
	ToUserID   uint `gorm:"not null" json:"toUserId"`
	Amount     int  `gorm:"not null" json:"amount"`
}
