package model

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Order struct {
	ID           uint            `json:"id" gorm:"primaryKey"`
	UserID       uint            `json:"user_id" gorm:"not null;index"`
	Symbol       string          `json:"symbol" gorm:"size:20;not null;index"`
	Side         OrderSide       `json:"side" gorm:"type:varchar(10);not null"`
	Type         OrderType       `json:"type" gorm:"type:varchar(20);not null"`
	Amount       decimal.Decimal `json:"amount" gorm:"type:decimal(18,8);not null"`
	Price        decimal.Decimal `json:"price" gorm:"type:decimal(18,8)"`
	FilledAmount decimal.Decimal `json:"filled_amount" gorm:"type:decimal(18,8);default:0"`
	Status       OrderStatus     `json:"status" gorm:"type:varchar(20);default:'pending'"`
	Trades       []Trade         `json:"trades,omitempty" gorm:"foreignKey:BuyOrderID;foreignKey:SellOrderID"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	DeletedAt    gorm.DeletedAt  `json:"-" gorm:"index"`
}

type OrderSide string
type OrderType string
type OrderStatus string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

const (
	OrderTypeMarket OrderType = "market"
	OrderTypeLimit  OrderType = "limit"
)

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPartial   OrderStatus = "partial"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusRejected  OrderStatus = "rejected"
)

type Trade struct {
	ID          uint            `json:"id" gorm:"primaryKey"`
	BuyOrderID  uint            `json:"buy_order_id" gorm:"not null;index"`
	SellOrderID uint            `json:"sell_order_id" gorm:"not null;index"`
	BuyOrder    *Order          `json:"buy_order,omitempty" gorm:"foreignKey:BuyOrderID"`
	SellOrder   *Order          `json:"sell_order,omitempty" gorm:"foreignKey:SellOrderID"`
	Symbol      string          `json:"symbol" gorm:"size:20;not null;index"`
	Amount      decimal.Decimal `json:"amount" gorm:"type:decimal(18,8);not null"`
	Price       decimal.Decimal `json:"price" gorm:"type:decimal(18,8);not null"`
	CreatedAt   time.Time       `json:"created_at"`
}

type Wallet struct {
	ID            uint            `json:"id" gorm:"primaryKey"`
	UserID        uint            `json:"user_id" gorm:"not null;index"`
	Currency      string          `json:"currency" gorm:"size:10;not null"`
	Balance       decimal.Decimal `json:"balance" gorm:"type:decimal(18,8);default:0"`
	FrozenBalance decimal.Decimal `json:"frozen_balance" gorm:"type:decimal(18,8);default:0"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

func (o *Order) GetAvailableAmount() decimal.Decimal {
	return o.Amount.Sub(o.FilledAmount)
}

func (o *Order) IsFilled() bool {
	return o.FilledAmount.Equal(o.Amount)
}

func (o *Order) IsActive() bool {
	return o.Status == OrderStatusPending || o.Status == OrderStatusPartial
}

func (o *Order) CanMatch(other *Order) bool {
	if o.Symbol != other.Symbol || o.Side == other.Side {
		return false
	}

	if o.Type == OrderTypeMarket || other.Type == OrderTypeMarket {
		return true
	}

	if o.Side == OrderSideBuy {
		return o.Price.GreaterThanOrEqual(other.Price)
	}

	return o.Price.LessThanOrEqual(other.Price)
}
