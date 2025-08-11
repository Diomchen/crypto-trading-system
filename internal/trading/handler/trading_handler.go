package handler

import (
	"crypto_trading_system/internal/trading/service"
	"crypto_trading_system/pkg/auth"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TradingHandler struct {
	tradingService service.TradingService
	jwtService     auth.JWTService
}

func NewTradingHandler(tradingservice service.TradingService, jwtService auth.JWTService) *TradingHandler {
	return &TradingHandler{
		tradingService: tradingservice,
		jwtService:     jwtService,
	}
}

func (h *TradingHandler) RegisterRoutes(r *gin.RouterGroup) {
	trading := r.Group("/trading")
	{
		trading.POST("/orders", h.PlaceOrder)
		trading.DELETE("/orders/:id", h.CancelOrder)
		trading.GET("/orders/:id", h.GetOrder)
		trading.GET("/orders", h.GetUserOrders)
	}

	market := r.Group("/market")
	{
		market.GET("/orderbook/:symbol", h.GetOrderBook)
		market.GET("/trades/:symbol", h.GetRecentTrades)
	}
}

func (h *TradingHandler) PlaceOrder(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.PlaceOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.UserID = userID

	order, err := h.tradingService.PlaceOrder(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    order,
	})
}

func (h *TradingHandler) CancelOrder(c *gin.Context) {
	userID := c.GetUint("user_id")
	orderIDStr := c.Param("id")

	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	if err := h.tradingService.CancelOrder(c.Request.Context(), userID, uint(orderID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Order cancelled successfully",
	})
}

func (h *TradingHandler) GetOrder(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	order, err := h.tradingService.GetOrder(c.Request.Context(), uint(orderID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    order,
	})
}

func (h *TradingHandler) GetUserOrders(c *gin.Context) {
	userID := c.GetUint("user_id")

	offserStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "20")

	offset, _ := strconv.Atoi(offserStr)
	limit, _ := strconv.Atoi(limitStr)
	order, err := h.tradingService.GetUserOrders(c.Request.Context(), userID, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    order,
	})

}
func (h *TradingHandler) GetOrderBook(c *gin.Context) {
	symbol := c.Param("symbol")

	orderBook := h.tradingService.GetOrderBook(symbol)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    orderBook,
	})
}

func (h *TradingHandler) GetRecentTrades(c *gin.Context) {
	symbol := c.Param("symbol")

	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	trades, err := h.tradingService.GetRecentTrades(c.Request.Context(), symbol, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    trades,
	})
}
