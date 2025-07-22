// cmd/test-connection/main.go
package main

import (
	"context"
	"crypto_trading_system/pkg/config"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	log.Printf("Testing connections to server: %s", cfg.Database.Host)

	// 测试PostgreSQL连接
	if err := testPostgreSQL(cfg); err != nil {
		log.Printf("❌ PostgreSQL connection failed: %v", err)
	} else {
		log.Println("✅ PostgreSQL connection successful")
	}

	// 测试Redis连接
	if err := testRedis(cfg); err != nil {
		log.Printf("❌ Redis connection failed: %v", err)
	} else {
		log.Println("✅ Redis connection successful")
	}

	// 测试RabbitMQ连接
	if err := testRabbitMQ(cfg); err != nil {
		log.Printf("❌ RabbitMQ connection failed: %v", err)
	} else {
		log.Println("✅ RabbitMQ connection successful")
	}

	log.Println("🎉 All connection tests completed!")
}

func testPostgreSQL(cfg *config.Config) error {
	dsn := cfg.GetDatabaseDSN()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// 测试查询
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to query users: %w", err)
	}

	log.Printf("PostgreSQL: Found %d users in database", count)
	return nil
}

func testRedis(cfg *config.Config) error {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 测试ping
	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping Redis: %w", err)
	}

	// 测试设置和获取
	testKey := "test:connection"
	testValue := "Hello from Windows!"

	if err := rdb.Set(ctx, testKey, testValue, time.Minute).Err(); err != nil {
		return fmt.Errorf("failed to set value: %w", err)
	}

	val, err := rdb.Get(ctx, testKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get value: %w", err)
	}

	if val != testValue {
		return fmt.Errorf("value mismatch: expected %s, got %s", testValue, val)
	}

	log.Printf("Redis: Set/Get test successful")
	return nil
}

func testRabbitMQ(cfg *config.Config) error {
	conn, err := amqp.Dial(cfg.GetRabbitMQURL())
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	// 测试队列声明
	testQueue := "test.connection"
	_, err = ch.QueueDeclare(
		testQueue,
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// 测试发送消息
	err = ch.Publish(
		"",        // exchange
		testQueue, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("Hello from Windows!"),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("RabbitMQ: Message publish test successful")
	return nil
}
