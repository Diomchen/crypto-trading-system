package cache

import (
	"context"
	"crypto_trading_system/pkg/config"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	ErrKeyNotFound = fmt.Errorf("key not found")
)

type RedisClient struct {
	client  *redis.Client
	prefix  string
	metrics *CacheMetrics
}

type CacheMetrics struct {
	hits   int64
	misses int64
	errors int64
}

func NewRedisClient(cfg *config.Config) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.GetRedisAddr(),
		Password:     cfg.GetRedisPassword(),
		DB:           cfg.GetRedisDB(),
		PoolSize:     20,
		PoolTimeout:  30 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		DialTimeout:  30 * time.Second,
		MinIdleConns: 10,
		MaxRetries:   3,
	})

	// 测试 redis 是否能连接上
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{
		client:  rdb,
		prefix:  cfg.GetPrefix(),
		metrics: &CacheMetrics{},
	}, nil
}

func (r *RedisClient) buildKey(key string) string {
	if r.prefix != "" {
		return fmt.Sprintf("%s:%s", r.prefix, key)
	}
	return key
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	fullKey := r.buildKey(key)
	val, err := r.client.Get(ctx, fullKey).Result()
	if err == redis.Nil {
		r.metrics.misses++
		return "", ErrKeyNotFound
	} else if err != nil {
		r.metrics.errors++
		return "", fmt.Errorf("failed to get key %s from redis: %w", fullKey, err)
	}

	r.metrics.hits++
	return val, nil
}

func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	fullKey := r.buildKey(key)
	var val string
	switch v := value.(type) {
	case string:
		val = v
	case []byte:
		val = string(v)
	default:
		data, err := json.Marshal(value)
		if err != nil {
			r.metrics.errors++
			return fmt.Errorf("failed to marshal value to json: %w", err)
		}
		val = string(data)
	}

	if err := r.client.Set(ctx, fullKey, val, expiration).Err(); err != nil {
		r.metrics.errors++
		return fmt.Errorf("failed to set key %s to redis: %w", fullKey, err)
	}
	return nil
}

func (r *RedisClient) GetJSON(ctx context.Context, key string, target interface{}) error {
	val, err := r.Get(ctx, key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(val), target); err != nil {
		r.metrics.errors++
		return fmt.Errorf("failed to unmarshal value from json: %w", err)
	}
	return nil
}

func (r *RedisClient) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Set(ctx, key, value, expiration)
}

func (r *RedisClient) Del(ctx context.Context, keys ...string) error {
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = r.buildKey(key)
	}

	if err := r.client.Del(ctx, fullKeys...).Err(); err != nil {
		r.metrics.errors++
		return fmt.Errorf("failed to delete keys from redis: %w", err)
	}
	return nil
}

func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := r.buildKey(key)
	count, err := r.client.Exists(ctx, fullKey).Result()
	if err != nil {
		r.metrics.errors++
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}

	return count > 0, nil
}

func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	fullKey := r.buildKey(key)
	if err := r.client.Expire(ctx, fullKey, expiration).Err(); err != nil {
		r.metrics.errors++
		return fmt.Errorf("failed to set expiration: %w", err)
	}

	return nil
}

func (r *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := r.buildKey(key)
	ttl, err := r.client.TTL(ctx, fullKey).Result()
	if err != nil {
		r.metrics.errors++
		return 0, fmt.Errorf("failed to get TTL: %w", err)
	}

	return ttl, nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

func (r *RedisClient) Health() error {
	return r.client.Ping(context.Background()).Err()
}
