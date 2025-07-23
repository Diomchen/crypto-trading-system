package app

import (
	"crypto_trading_system/pkg/cache"
	"crypto_trading_system/pkg/config"
	"crypto_trading_system/pkg/database"
	"crypto_trading_system/pkg/logger"
	"fmt"
)

type App struct {
	Config   *config.Config
	Database *database.Database
	Redis    *cache.RedisClient
	Logger   *logger.Logger
}

func NewApp(configPath string) (*App, error) {
	// cfg, err := config.LoadConfig(configPath)
	//TODO: use the default for dev
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w")
	}

	logger := logger.NewLogger(cfg)
	logger.Info("🚀[DIOMCHEN]starting app ...")

	database, err := database.NewDatabase(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w")
	}
	logger.Info("📚connected to database")

	redis, err := cache.NewRedisClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w")
	}

	return &App{
		Config:   cfg,
		Database: database,
		Redis:    redis,
		Logger:   logger,
	}, nil

}

func (a *App) Close() error {
	a.Logger.Info("🔌closing app ...")
	if err := a.Redis.Close(); err != nil {
		a.Logger.Error("failed to close redis client: %v", err)
	}

	if err := a.Database.Close(); err != nil {
		a.Logger.Error("failed to close database: %v", err)
	}

	return nil
}

func (a *App) HealthCheck() error {
	if err := a.Database.Health(); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	if err := a.Redis.Health(); err != nil {
		return fmt.Errorf("Redis health check failed: %w", err)
	}

	return nil
}
