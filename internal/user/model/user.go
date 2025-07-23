package model

import (
	trading "crypto_trading_system/internal/trading/model"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        uint             `json:"id" gorm:"primaryKey"`
	Username  string           `json:"username" gorm:"uniqueIndex;size:50;not null"`
	Email     string           `json:"email" gorm:"uniqueIndex;size:100;not null"`
	Password  string           `json:"-" gorm:"size:255;not null"`
	Status    UserStatus       `json:"status" gorm:"type:varchar(20);default:'active'"`
	Profile   *UserProfile     `json:"profile,omitempty" gorm:"foreignKey:UserID"`
	Wallets   []trading.Wallet `json:"wallets,omitempty" gorm:"foreignKey:UserID"`
	Orders    []trading.Order  `json:"orders,omitempty" gorm:"foreignKey:UserID"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusBanned   UserStatus = "banned"
)

type UserProfile struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"user_id" gorm:"not null"`
	FirstName   string    `json:"first_name" gorm:"size:50"`
	LastName    string    `json:"last_name" gorm:"size:50"`
	Avatar      string    `json:"avatar" gorm:"size:255"`
	Phone       string    `json:"phone" gorm:"size:20"`
	Country     string    `json:"country" gorm:"size:50"`
	City        string    `json:"city" gorm:"size:50"`
	DateOfBirth time.Time `json:"date_of_birth"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}
