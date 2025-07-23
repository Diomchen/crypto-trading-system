package main

import (
	"crypto_trading_system/internal/user/handler"
	"crypto_trading_system/internal/user/repository"
	"crypto_trading_system/internal/user/service"
	"crypto_trading_system/pkg/app"
	"crypto_trading_system/pkg/auth"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 测试用户服务
func main() {
	app, err := app.NewApp("./config.yaml")
	if err != nil {
		log.Fatal("Failed to initialize app:", err)
	}
	defer app.Close()

	// 初始化服务
	userRepo := repository.NewUserRepository(app.Database.DB)
	jwtService := auth.NewJWTService(app.Config)
	userService := service.NewUserService(userRepo, jwtService, app.Redis, app.Logger)
	userHandler := handler.NewUserHandler(userService, jwtService)

	// 设置路由
	r := gin.Default()

	// 设置中间件
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		if err := app.HealthCheck(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  err.Error(),
				"status": "unhealthy",
			})
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	// api 路由
	api := r.Group("/api/v1")
	userHandler.RegisterRoutes(api)

	// 启动
	app.Logger.Info("🚀[DIOMCHEN]user service is running ...")
	if err := r.Run(":8181"); err != nil {
		app.Logger.Error("Failed to start user service:", err)
	}
}
