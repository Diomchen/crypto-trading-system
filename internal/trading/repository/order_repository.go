package repository

import (
	"context"
	"crypto_trading_system/internal/trading/model"
	"errors"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	ErrOrderNotFound       = errors.New("order not found")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

type OrderRepository interface {
	Create(ctx context.Context, order *model.Order) error
	GetByID(ctx context.Context, id uint) (*model.Order, error)
	GetActiveOrdersBySymbol(ctx context.Context, symbol string, side model.OrderSide) ([]*model.Order, error)
	GetUserOrders(ctx context.Context, userID uint, offset, limit int) ([]*model.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID uint, status model.OrderStatus, filledAmount decimal.Decimal) error
	CreateTrade(ctx context.Context, trade *model.Trade) error
	GetTradesBySymbol(ctx context.Context, symbol string, limit int) ([]*model.Trade, error)
}

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

// 创建订单
func (r *orderRepository) Create(ctx context.Context, order *model.Order) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *orderRepository) GetByID(ctx context.Context, id uint) (*model.Order, error) {
	var order model.Order
	err := r.db.WithContext(ctx).
		Preload("Trades").
		First(&order, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	return &order, nil
}

// 通过交易对获取活动订单
func (r *orderRepository) GetActiveOrdersBySymbol(ctx context.Context, symbol string, side model.OrderSide) ([]*model.Order, error) {
	var orders []*model.Order
	query := r.db.WithContext(ctx).
		Where("symbol = ? AND  side = ?", symbol, side).
		Where("status IN ?", []model.OrderStatus{model.OrderStatusPending, model.OrderStatusPartial})

	// 根据订单类别排序
	if side == model.OrderSideBuy {
		query = query.Order("price DESC, created_at ASC")
	} else {
		query = query.Order("price ASC, created_at ASC")
	}

	err := query.Find(&orders).Error
	return orders, err
}

// 获取用户订单
func (r *orderRepository) GetUserOrders(ctx context.Context, userID uint, offset, limit int) ([]*model.Order, error) {
	var orders []*model.Order

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&orders).Error

	return orders, err
}

func (r *orderRepository) UpdateOrderStatus(ctx context.Context, orderID uint, status model.OrderStatus, filledAmount decimal.Decimal) error {
	return r.db.WithContext(ctx).Model(&model.Order{}).
		Where("id = ?", orderID).
		Updates(map[string]interface{}{
			"status":        status,
			"filled_amount": filledAmount,
		}).Error
}

func (r *orderRepository) CreateTrade(ctx context.Context, trade *model.Trade) error {
	return r.db.WithContext(ctx).Create(trade).Error
}

func (r *orderRepository) GetTradesBySymbol(ctx context.Context, symbol string, limit int) ([]*model.Trade, error) {
	var trades []*model.Trade
	err := r.db.WithContext(ctx).
		Where("symbol = ?", symbol).
		Order("created_at DESC").
		Limit(limit).Find(&trades).Error

	return trades, err
}
