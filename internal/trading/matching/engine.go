package matching

import (
	"container/heap"
	"context"
	"crypto_trading_system/internal/trading/model"
	"crypto_trading_system/pkg/logger"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// 撮合引擎接口
type MatchingEngine interface {
	ProcessOrder(ctx context.Context, order *model.Order) ([]*model.Trade, error)
	GetOrderBook(symbol string) *OrderBook
	Start(ctx context.Context)
	Stop()
}

// 订单簿
type OrderBook struct {
	Symbol     string
	BuyOrders  *OrderQueue // 买单队列(最大堆)
	SellOrders *OrderQueue // 卖单队列（最小堆）
	mu         sync.RWMutex
	LastPrice  decimal.Decimal
	LastUpdate time.Time
}

func (ob *OrderBook) RLock() {
	ob.mu.RLock()
}

func (ob *OrderBook) RUnlock() {
	ob.mu.RUnlock()
}

// TODO: 是否可以移动到工具包中？
// 订单队列（优先级队列）
type OrderQueue struct {
	Orders []*model.Order
	isBuy  bool
}

func (q *OrderQueue) Len() int {
	return len(q.Orders)
}

func (q *OrderQueue) Less(i, j int) bool {
	// 价格优先，时间优先
	if q.Orders[i].Price.Equal(q.Orders[j].Price) {
		return q.Orders[i].CreatedAt.Before(q.Orders[j].CreatedAt)
	}

	if q.isBuy {
		// 买单，价格高优先
		return q.Orders[i].Price.GreaterThan(q.Orders[j].Price)
	} else {
		// 卖单，价格低优先
		return q.Orders[i].Price.LessThan(q.Orders[j].Price)
	}
}

func (q *OrderQueue) Swap(i, j int) {
	q.Orders[i], q.Orders[j] = q.Orders[j], q.Orders[i]
}

func (q *OrderQueue) Push(x interface{}) {
	q.Orders = append(q.Orders, x.(*model.Order))
}

func (q *OrderQueue) Pop() interface{} {
	old := q.Orders
	n := len(old)
	item := old[n-1]
	q.Orders = old[0 : n-1]
	return item
}

func (q *OrderQueue) Peek() *model.Order {
	if len(q.Orders) == 0 {
		return nil
	}
	return q.Orders[0]
}

type matchingEngine struct {
	orderBooks map[string]*OrderBook
	mu         sync.RWMutex
	logger     *logger.Logger
	// 通道用于处理订单
	orderChan chan *OrderRequest
	// 通道用于处理成交
	tradeChan chan *model.Trade
	// 停止信号
	stopChan chan struct{}
	wg       sync.WaitGroup
}

type OrderRequest struct {
	Order      *model.Order
	ResultChan chan *MatchResult
}

type MatchResult struct {
	Trades []*model.Trade
	Error  error
}

func NewMatchingEngine(logger *logger.Logger) MatchingEngine {
	return &matchingEngine{
		orderBooks: make(map[string]*OrderBook),
		logger:     logger,
		orderChan:  make(chan *OrderRequest, 1000),
		tradeChan:  make(chan *model.Trade, 100),
		stopChan:   make(chan struct{}),
	}
}

func (e *matchingEngine) Start(ctx context.Context) {
	e.logger.Info("matching engine start")

	e.wg.Add(1)
	go e.processOrder(ctx)
}

func (e *matchingEngine) Stop() {
	e.logger.Info("matching engine stop")
	close(e.stopChan)
	e.wg.Wait()
}

func (e *matchingEngine) ProcessOrder(ctx context.Context, order *model.Order) ([]*model.Trade, error) {
	resultChan := make(chan *MatchResult, 1)

	select {
	case e.orderChan <- &OrderRequest{Order: order, ResultChan: resultChan}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	select {
	case result := <-resultChan:
		return result.Trades, result.Error
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (e *matchingEngine) GetOrderBook(symbol string) *OrderBook {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.orderBooks[symbol]
}

func (e *matchingEngine) processOrder(ctx context.Context) {
	defer e.wg.Done()

	for {
		select {
		case orderReq := <-e.orderChan:
			trades, err := e.matchOrder(orderReq.Order)
			orderReq.ResultChan <- &MatchResult{Trades: trades, Error: err}
		case <-e.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (e *matchingEngine) matchOrder(order *model.Order) ([]*model.Trade, error) {
	orderBook := e.getOrCreateOrderBook(order.Symbol)
	orderBook.mu.Lock()
	defer orderBook.mu.Unlock()

	var trades []*model.Trade

	// 获取对手盘订单
	var counterQueue *OrderQueue
	if order.Side == model.OrderSideBuy {
		sellOrders := orderBook.SellOrders
		counterQueue = sellOrders
	} else {
		buyOrders := orderBook.BuyOrders
		counterQueue = buyOrders
	}

	// 执行撮合
	for counterQueue.Len() > 0 && order.GetAvailableAmount().GreaterThan(decimal.Zero) {
		counterOrder := counterQueue.Peek()

		// 检查是否可以撮合
		if !order.CanMatch(counterOrder) {
			break
		}

		// 计算成交量和价格
		tradeAmount := decimal.Min(order.GetAvailableAmount(), counterOrder.GetAvailableAmount())
		tradePrice := counterOrder.Price // 使用对手盘价格

		// 创建交易记录
		trade := &model.Trade{
			Symbol: order.Symbol,
			Amount: tradeAmount,
			Price:  tradePrice,
		}

		if order.Side == model.OrderSideBuy {
			trade.BuyOrderID = order.ID
			trade.SellOrderID = counterOrder.ID
		} else {
			trade.BuyOrderID = counterOrder.ID
			trade.SellOrderID = order.ID
		}

		trades = append(trades, trade)

		// 更新订单状态
		order.FilledAmount = order.FilledAmount.Add(tradeAmount)
		counterOrder.FilledAmount = counterOrder.FilledAmount.Add(tradeAmount)

		// 更新最新价格
		orderBook.LastPrice = tradePrice
		orderBook.LastUpdate = time.Now()

		// 检查对手单是否完全成交
		if counterOrder.IsFilled() {
			// 完全移出买/卖订单队列
			heap.Pop(counterQueue)
		}

	}

	// 如果是限价订单且未完全成交，就加入订单簿
	if order.Type == model.OrderTypeLimit && order.GetAvailableAmount().GreaterThan(decimal.Zero) {
		if order.Side == model.OrderSideBuy {
			heap.Push(orderBook.BuyOrders, order)
		} else {
			heap.Push(orderBook.SellOrders, order)
		}
	}

	return trades, nil
}

func (e *matchingEngine) getOrCreateOrderBook(symbol string) *OrderBook {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 已经存在就返回
	if orderBook, exists := e.orderBooks[symbol]; exists {
		return orderBook
	}

	// 不存在就创建
	orderBook := &OrderBook{
		Symbol:     symbol,
		BuyOrders:  &OrderQueue{isBuy: true},
		SellOrders: &OrderQueue{isBuy: false},
		LastPrice:  decimal.Zero,
		LastUpdate: time.Now(),
	}

	// 初始化各方向优先队列
	heap.Init(orderBook.BuyOrders)
	heap.Init(orderBook.SellOrders)

	e.orderBooks[symbol] = orderBook
	e.logger.WithField("symbol", symbol).Info("Created new order book.")

	return orderBook
}
