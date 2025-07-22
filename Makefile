# Makefile
.PHONY: help build run test clean

# 服务器IP地址（需要替换为实际IP）
SERVER_IP=YOUR_SERVER_IP

help: ## 显示帮助信息
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

install: ## 安装依赖
	go mod download
	go mod tidy

build: ## 构建所有服务
	go build -o bin/user-service.exe ./cmd/user-service
	go build -o bin/trading-service.exe ./cmd/trading-service
	go build -o bin/api-gateway.exe ./cmd/api-gateway

run-user: ## 运行用户服务
	@echo "Starting user service..."
	set SERVER_IP=$(SERVER_IP) && go run ./cmd/user-service

run-trading: ## 运行交易服务
	@echo "Starting trading service..."
	set SERVER_IP=$(SERVER_IP) && go run ./cmd/trading-service

run-gateway: ## 运行API网关
	@echo "Starting API gateway..."
	set SERVER_IP=$(SERVER_IP) && go run ./cmd/api-gateway

test: ## 运行测试
	go test -v ./...

test-cover: ## 运行测试并生成覆盖率报告
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean: ## 清理构建文件
	if exist bin rmdir /s /q bin
	if exist coverage.out del coverage.out
	if exist coverage.html del coverage.html

lint: ## 代码检查
	golangci-lint run

fmt: ## 格式化代码
	gofmt -s -w .
	go mod tidy

check-server: ## 检查服务器连接
	@echo "Checking server connection..."
	@echo "PostgreSQL:" && telnet $(SERVER_IP) 5432
	@echo "Redis:" && telnet $(SERVER_IP) 6379
	@echo "RabbitMQ:" && telnet $(SERVER_IP) 5672

setup-server: ## 设置服务器IP
	@echo "Setting SERVER_IP to $(SERVER_IP)"
	@powershell -Command "(Get-Content configs/config.yaml) -replace 'YOUR_SERVER_IP', '$(SERVER_IP)' | Set-Content configs/config.yaml"

init: ## 初始化项目
	@echo "Initializing project..."
	go mod init crypto-trading-system
	go mod tidy
	@echo "Project initialized successfully!"
