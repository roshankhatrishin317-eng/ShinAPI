// Package cache provides caching utilities for the API proxy.
// This file implements a go-redis based Redis client.
package cache

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/redis/go-redis/v9"
)

// GoRedisClient implements RedisClient using go-redis.
type GoRedisClient struct {
	client *redis.Client
}

// GoRedisConfig holds configuration for the go-redis client.
type GoRedisConfig struct {
	Address       string
	Password      string
	Database      int
	PoolSize      int
	DialTimeout   time.Duration
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	EnableTLS     bool
	MaxRetries    int
}

// DefaultGoRedisConfig returns default configuration for localhost:6379.
func DefaultGoRedisConfig() GoRedisConfig {
	return GoRedisConfig{
		Address:      "localhost:6379",
		Password:     "",
		Database:     0,
		PoolSize:     10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		EnableTLS:    false,
		MaxRetries:   3,
	}
}

// NewGoRedisClient creates a new go-redis based client.
func NewGoRedisClient(cfg GoRedisConfig) *GoRedisClient {
	opts := &redis.Options{
		Addr:         cfg.Address,
		Password:     cfg.Password,
		DB:           cfg.Database,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		MaxRetries:   cfg.MaxRetries,
	}

	if cfg.EnableTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return &GoRedisClient{
		client: redis.NewClient(opts),
	}
}

// NewGoRedisClientFromRedisCacheConfig creates a client from RedisCacheConfig.
func NewGoRedisClientFromRedisCacheConfig(cfg RedisCacheConfig) *GoRedisClient {
	return NewGoRedisClient(GoRedisConfig{
		Address:      cfg.Address,
		Password:     cfg.Password,
		Database:     cfg.Database,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  time.Duration(cfg.DialTimeoutMs) * time.Millisecond,
		ReadTimeout:  time.Duration(cfg.ReadTimeoutMs) * time.Millisecond,
		WriteTimeout: time.Duration(cfg.WriteTimeoutMs) * time.Millisecond,
		EnableTLS:    cfg.EnableTLS,
		MaxRetries:   cfg.MaxRetries,
	})
}

// Get retrieves a value from Redis.
func (c *GoRedisClient) Get(ctx context.Context, key string) ([]byte, error) {
	return c.client.Get(ctx, key).Bytes()
}

// Set stores a value in Redis with TTL.
func (c *GoRedisClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Delete removes a key from Redis.
func (c *GoRedisClient) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Exists checks if a key exists in Redis.
func (c *GoRedisClient) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	return n > 0, err
}

// TTL returns the remaining TTL for a key.
func (c *GoRedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

// Keys returns all keys matching a pattern.
func (c *GoRedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.client.Keys(ctx, pattern).Result()
}

// Ping checks Redis connectivity.
func (c *GoRedisClient) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close closes the Redis connection.
func (c *GoRedisClient) Close() error {
	return c.client.Close()
}

// Client returns the underlying go-redis client for advanced operations.
func (c *GoRedisClient) Client() *redis.Client {
	return c.client
}
