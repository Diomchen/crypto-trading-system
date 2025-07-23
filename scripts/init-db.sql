-- scripts/init-db.sql
-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建订单表
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    symbol VARCHAR(20) NOT NULL,
    side VARCHAR(10) NOT NULL, -- 'buy' or 'sell'
    order_type VARCHAR(20) NOT NULL, -- 'market', 'limit'
    amount DECIMAL(18,8) NOT NULL,
    price DECIMAL(18,8),
    filled_amount DECIMAL(18,8) DEFAULT 0,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建交易记录表
CREATE TABLE IF NOT EXISTS trades (
    id SERIAL PRIMARY KEY,
    buy_order_id INTEGER REFERENCES orders(id),
    sell_order_id INTEGER REFERENCES orders(id),
    symbol VARCHAR(20) NOT NULL,
    amount DECIMAL(18,8) NOT NULL,
    price DECIMAL(18,8) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建钱包表
CREATE TABLE IF NOT EXISTS wallets (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    currency VARCHAR(10) NOT NULL,
    balance DECIMAL(18,8) DEFAULT 0,
    frozen_balance DECIMAL(18,8) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, currency)
);

-- 创建用户信息表
CREATE TABLE IF NOT EXISTS user_profiles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    avatar VARCHAR(255),
    phone VARCHAR(20),
    country VARCHAR(50),
    city VARCHAR(50),
    date_of_birth DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_user_profiles_user_id ON user_profiles(user_id);

-- 插入测试数据
INSERT INTO users (username, email, password_hash) VALUES 
('testuser1', 'test1@example.com', '$2a$10$rQZ9nBwfIp1o8DCFxjyXeOqYQEzxhvQfTqCjmPRNmZxJEWF7oJ8kG'),
('testuser2', 'test2@example.com', '$2a$10$rQZ9nBwfIp1o8DCFxjyXeOqYQEzxhvQfTqCjmPRNmZxJEWF7oJ8kG')
ON CONFLICT DO NOTHING;

-- 插入测试钱包
INSERT INTO wallets (user_id, currency, balance) VALUES 
(1, 'USDT', 10000.0),
(1, 'BTC', 0.0),
(2, 'USDT', 10000.0),
(2, 'BTC', 0.0)
ON CONFLICT DO NOTHING;
