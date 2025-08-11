package service

import (
	"context"

	"crypto_trading_system/internal/trading/matching"
	"crypto_trading_system/internal/trading/model"
	"crypto_trading_system/internal/trading/repository"
	"crypto_trading_system/pkg/logger"
	"fmt"

	"github.com/shopspring/decimal"
)

type TradingService interface {
	PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*model.Order, error)
	CancelOrder(ctx context.Context, userID, orderID uint) error
	GetOrder(ctx context.Context, orderID uint) (*model.Order, error)
	GetUserOrders(ctx context.Context, userID uint, offset, limit int) ([]*model.Order, error)
	GetOrderBook(symbol string) *OrderBookData
	GetRecentTrades(ctx context.Context, symbol string, limit int) ([]*model.Trade, error)
}

type PlaceOrderRequest struct {
	UserID uint            `json:"user_id"`
	Symbol string          `json:"symbol" binding:"required"`
	Side   model.OrderSide `json:"side" binding:"required,oneof=buy sell"`
	Type   model.OrderType `json:"type" binding:"required,oneof=limit market"`
	Amount decimal.Decimal `json:"amount" binding:"required"`
	Price  decimal.Decimal `json:"price"`
}

type OrderBookData struct {
	Symbol    string           `json:"symbol"`
	Bids      []OrderBookLevel `json:"bids"`
	Asks      []OrderBookLevel `json:"asks"`
	LastPrice decimal.Decimal  `json:"last_price"`
}

type OrderBookLevel struct {
	Price  decimal.Decimal `json:"price"`
	Amount decimal.Decimal `json:"amount"`
}

type tradingService struct {
	orderRepo      repository.OrderRepository
	matchingEngine matching.MatchingEngine
	logger         *logger.Logger
}

func NewTraingService(orderRepo repository.OrderRepository, matchingEngine matching.MatchingEngine, logger *logger.Logger) TradingService {
	return &tradingService{
		orderRepo:      orderRepo,
		matchingEngine: matchingEngine,
		logger:         logger,
	}
}

func validateOrder(req *PlaceOrderRequest) error {
	if req.Amount.LessThan(decimal.Zero) {
		return fmt.Errorf("Amount must be greater than zero")
	}

	if req.Type == model.OrderTypeLimit && req.Price.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("Price must be greater than zero for limit order")
	}

	return nil
}

func (s *tradingService) PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*model.Order, error) {
	// 验证订单
	if err := validateOrder(req); err != nil {
		return nil, err
	}

	order := &model.Order{
		UserID: req.UserID,
		Symbol: req.Symbol,
		Side:   req.Side,
		Type:   req.Type,
		Amount: req.Amount,
		Price:  req.Price,
		Status: model.OrderStatusPending,
	}

	// 保存订单
	if err := s.orderRepo.Create(ctx, order); err != nil {
		s.logger.WithError(err).Error("Failed to create order")
		return nil, fmt.Errorf("Failed to create order: %w", err)
	}

	// 提交至撮合引擎
	trades, err := s.matchingEngine.ProcessOrder(ctx, order)
	if err != nil {
		s.logger.WithError(err).Error("Failed to process order")
		return nil, fmt.Errorf("Failed to process order: %w", err)
	}

	// 处理成交记录
	if len(trades) > 0 {
		for _, trade := range trades {
			if err := s.orderRepo.CreateTrade(ctx, trade); err != nil {
				s.logger.WithError(err).Error("Failed to create trade")
			}
		}

		// 更新订单状态
		status := model.OrderStatusPartial
		if order.IsFilled() {
			status = model.OrderStatusFilled
		}
		if err := s.orderRepo.UpdateOrderStatus(ctx, order.ID, status, order.FilledAmount); err != nil {
			s.logger.WithError(err).Error("Failed to update order status")
		}

		order.Status = status
	}

	s.logger.WithField("order_id", order.ID).
		WithField("trades_count", len(trades)).
		Info("Order processed sucessfully")

	return order, nil
}

func (s *tradingService) CancelOrder(ctx context.Context, userID, orderID uint) error {
	// 获取订单
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("Failed to get order: %w", err)
	}

	// 验证订单所有者
	if order.UserID != userID {
		return fmt.Errorf("Unauthorized to cancel order")
	}

	// 验证订单状态
	if !order.IsActive() {
		return fmt.Errorf("order can not be cancelled in current status: %s", order.Status)
	}

	// 更新订单状态
	if err := s.orderRepo.UpdateOrderStatus(ctx, orderID, model.OrderStatusCancelled, order.FilledAmount); err != nil {
		return fmt.Errorf("Failed to cancel order: %w", err)
	}

	s.logger.WithField("order_id", orderID).Info("Order cancelled successfully")

	return nil
}

func (s *tradingService) GetOrder(ctx context.Context, orderID uint) (*model.Order, error) {
	return s.orderRepo.GetByID(ctx, orderID)
}

func (s *tradingService) GetUserOrders(ctx context.Context, userID uint, offset, limit int) ([]*model.Order, error) {
	return s.orderRepo.GetUserOrders(ctx, userID, offset, limit)
}

func (s *tradingService) GetOrderBook(symbol string) *OrderBookData {
	orderBook := s.matchingEngine.GetOrderBook(symbol)
	if orderBook == nil {
		return &OrderBookData{
			Symbol: symbol,
			Bids:   []OrderBookLevel{},
			Asks:   []OrderBookLevel{},
		}
	}

	orderBook.RLock()
	defer orderBook.RUnlock()

	return &OrderBookData{
		Symbol:    symbol,
		Bids:      s.buildOrderBookLevels(orderBook.BuyOrders),
		Asks:      s.buildOrderBookLevels(orderBook.SellOrders),
		LastPrice: orderBook.LastPrice,
	}
}

func (s *tradingService) buildOrderBookLevels(queue *matching.OrderQueue) []OrderBookLevel {
	levels := make([]OrderBookLevel, 0, queue.Len())
	priceMap := make(map[string]decimal.Decimal)

	// 聚合订单
	for _, order := range queue.Orders {
		price := order.Price.String()
		priceMap[price] = priceMap[price].Add(order.GetAvailableAmount())
	}

	// 转化为数组
	for priceStr, amount := range priceMap {
		price, _ := decimal.NewFromString(priceStr)
		levels = append(levels, OrderBookLevel{Amount: amount, Price: price})
	}

	return levels
}

func (s *tradingService) GetRecentTrades(ctx context.Context, symbol string, limit int) ([]*model.Trade, error) {
	return s.orderRepo.GetTradesBySymbol(ctx, symbol, limit)
}
