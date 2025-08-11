package main

import (
	"context"
	"crypto_trading_system/internal/trading/handler"
	"crypto_trading_system/internal/trading/matching"
	"crypto_trading_system/internal/trading/repository"
	"crypto_trading_system/internal/trading/service"
	"crypto_trading_system/pkg/app"
	"crypto_trading_system/pkg/auth"
	"crypto_trading_system/pkg/middleware"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	app, err := app.NewApp("./config.yaml")
	if err != nil {
		log.Fatal("Failed to initialize app:", err)
	}
	defer app.Close()

	// 初始撮合引擎
	matchingEngine := matching.NewMatchingEngine(app.Logger)
	matchingEngine.Start(context.Background())
	defer matchingEngine.Stop()

	// 初始化
	orderRepo := repository.NewOrderRepository(app.Database.DB)
	jwtService := auth.NewJWTService(app.Config)
	tradingService := service.NewTraingService(orderRepo, matchingEngine, app.Logger)
	tradingHandler := handler.NewTradingHandler(tradingService, jwtService)

	//设置路由
	r := gin.Default()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	api := r.Group("/api/v1")
	api.Use(middleware.JWTMiddleware(jwtService))
	tradingHandler.RegisterRoutes(api)

	// 启动服务
	app.Logger.Info("🚀[DIOMCHEN]starting trading service ...")
	if err := r.Run(":8282"); err != nil {
		app.Logger.Error("❌[DIOMCHEN]failed to start trading service:", err)
	}

}
