// Package cache provides caching utilities for the API proxy.
// This file implements Redis-backed caching for distributed deployments.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// RedisClient defines the interface for Redis operations.
// This allows for different Redis client implementations (go-redis, redigo, etc.)
type RedisClient interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	Keys(ctx context.Context, pattern string) ([]string, error)
	Ping(ctx context.Context) error
	Close() error
}

// RedisCacheConfig configures the Redis cache.
type RedisCacheConfig struct {
	// Address is the Redis server address (host:port)
	Address string `yaml:"address" json:"address"`
	// Password is the Redis password (optional)
	Password string `yaml:"password" json:"password"`
	// Database is the Redis database number
	Database int `yaml:"database" json:"database"`
	// KeyPrefix is prepended to all cache keys
	KeyPrefix string `yaml:"key-prefix" json:"key_prefix"`
	// DefaultTTL is the default TTL for cached items
	DefaultTTLSeconds int `yaml:"default-ttl-seconds" json:"default_ttl_seconds"`
	// MaxRetries is the maximum number of retries for failed operations
	MaxRetries int `yaml:"max-retries" json:"max_retries"`
	// PoolSize is the maximum number of connections
	PoolSize int `yaml:"pool-size" json:"pool_size"`
	// DialTimeout is the timeout for establishing new connections
	DialTimeoutMs int `yaml:"dial-timeout-ms" json:"dial_timeout_ms"`
	// ReadTimeout is the timeout for read operations
	ReadTimeoutMs int `yaml:"read-timeout-ms" json:"read_timeout_ms"`
	// WriteTimeout is the timeout for write operations
	WriteTimeoutMs int `yaml:"write-timeout-ms" json:"write_timeout_ms"`
	// EnableTLS enables TLS for Redis connections
	EnableTLS bool `yaml:"enable-tls" json:"enable_tls"`
	// Enabled controls whether Redis caching is active
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// DefaultRedisCacheConfig returns sensible defaults.
func DefaultRedisCacheConfig() RedisCacheConfig {
	return RedisCacheConfig{
		Address:           "localhost:6379",
		Database:          0,
		KeyPrefix:         "shinapi:",
		DefaultTTLSeconds: 60,
		MaxRetries:        3,
		PoolSize:          10,
		DialTimeoutMs:     5000,
		ReadTimeoutMs:     3000,
		WriteTimeoutMs:    3000,
		EnableTLS:         false,
		Enabled:           false,
	}
}

// RedisCache provides Redis-backed caching with metrics.
type RedisCache struct {
	client    RedisClient
	config    RedisCacheConfig
	ttlConfig *ModelTTLConfig

	// Metrics
	hits      uint64
	misses    uint64
	errors    uint64
	latencyNs atomic.Int64

	mu     sync.RWMutex
	closed bool
}

// NewRedisCache creates a new Redis cache with the given client and config.
func NewRedisCache(client RedisClient, cfg RedisCacheConfig) *RedisCache {
	if cfg.DefaultTTLSeconds <= 0 {
		cfg.DefaultTTLSeconds = 60
	}
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "shinapi:"
	}

	return &RedisCache{
		client:    client,
		config:    cfg,
		ttlConfig: NewModelTTLConfig(time.Duration(cfg.DefaultTTLSeconds) * time.Second),
	}
}

// Get retrieves a value from Redis.
func (c *RedisCache) Get(model, key string) ([]byte, bool) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, false
	}
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.config.ReadTimeoutMs)*time.Millisecond)
	defer cancel()

	start := time.Now()
	fullKey := c.makeKey(model, key)

	data, err := c.client.Get(ctx, fullKey)
	c.latencyNs.Store(time.Since(start).Nanoseconds())

	if err != nil {
		atomic.AddUint64(&c.misses, 1)
		return nil, false
	}

	atomic.AddUint64(&c.hits, 1)
	return data, true
}

// Set stores a value in Redis with model-specific TTL.
func (c *RedisCache) Set(model, key string, value []byte) error {
	return c.SetWithTTL(model, key, value, c.ttlConfig.GetTTL(model))
}

// SetWithTTL stores a value in Redis with a custom TTL.
func (c *RedisCache) SetWithTTL(model, key string, value []byte, ttl time.Duration) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("cache is closed")
	}
	c.mu.RUnlock()

	if len(value) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.config.WriteTimeoutMs)*time.Millisecond)
	defer cancel()

	start := time.Now()
	fullKey := c.makeKey(model, key)

	err := c.client.Set(ctx, fullKey, value, ttl)
	c.latencyNs.Store(time.Since(start).Nanoseconds())

	if err != nil {
		atomic.AddUint64(&c.errors, 1)
		return err
	}

	return nil
}

// Delete removes a key from Redis.
func (c *RedisCache) Delete(model, key string) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.config.WriteTimeoutMs)*time.Millisecond)
	defer cancel()

	fullKey := c.makeKey(model, key)
	return c.client.Delete(ctx, fullKey)
}

// Clear removes all keys with the configured prefix.
func (c *RedisCache) Clear() error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pattern := c.config.KeyPrefix + "*"
	keys, err := c.client.Keys(ctx, pattern)
	if err != nil {
		return err
	}

	for _, key := range keys {
		if err := c.client.Delete(ctx, key); err != nil {
			atomic.AddUint64(&c.errors, 1)
		}
	}

	return nil
}

// SetModelTTL sets the TTL for a specific model.
func (c *RedisCache) SetModelTTL(model string, ttl time.Duration) {
	c.ttlConfig.SetModelTTL(model, ttl)
}

// SetPatternTTL sets the TTL for models matching a pattern.
func (c *RedisCache) SetPatternTTL(pattern string, ttl time.Duration) {
	c.ttlConfig.SetPatternTTL(pattern, ttl)
}

// GetTTL returns the TTL for a specific model.
func (c *RedisCache) GetTTL(model string) time.Duration {
	return c.ttlConfig.GetTTL(model)
}

// Ping checks Redis connectivity.
func (c *RedisCache) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.config.DialTimeoutMs)*time.Millisecond)
	defer cancel()
	return c.client.Ping(ctx)
}

// Close closes the Redis connection.
func (c *RedisCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	return c.client.Close()
}

// Stats returns Redis cache statistics.
func (c *RedisCache) Stats() RedisCacheStats {
	hits := atomic.LoadUint64(&c.hits)
	misses := atomic.LoadUint64(&c.misses)
	errors := atomic.LoadUint64(&c.errors)
	latencyNs := c.latencyNs.Load()

	total := hits + misses
	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return RedisCacheStats{
		Hits:            hits,
		Misses:          misses,
		Errors:          errors,
		HitRate:         hitRate,
		LastLatencyMs:   float64(latencyNs) / 1e6,
		Connected:       c.Ping() == nil,
		KeyPrefix:       c.config.KeyPrefix,
		DefaultTTLSec:   c.config.DefaultTTLSeconds,
	}
}

// makeKey creates a full Redis key with prefix.
func (c *RedisCache) makeKey(model, key string) string {
	return c.config.KeyPrefix + model + ":" + HashKey(key)
}

// RedisCacheStats holds Redis cache statistics.
type RedisCacheStats struct {
	Hits          uint64  `json:"hits"`
	Misses        uint64  `json:"misses"`
	Errors        uint64  `json:"errors"`
	HitRate       float64 `json:"hit_rate_percent"`
	LastLatencyMs float64 `json:"last_latency_ms"`
	Connected     bool    `json:"connected"`
	KeyPrefix     string  `json:"key_prefix"`
	DefaultTTLSec int     `json:"default_ttl_seconds"`
}

// CachedStreamingResponse stores a streaming response for Redis.
type CachedStreamingResponse struct {
	Events    []StreamEvent `json:"events"`
	TotalSize int64         `json:"total_size"`
	CreatedAt time.Time     `json:"created_at"`
}

// GetStreamingResponse retrieves a cached streaming response from Redis.
func (c *RedisCache) GetStreamingResponse(key string) ([]StreamEvent, bool) {
	data, found := c.Get("streaming", key)
	if !found {
		return nil, false
	}

	var resp CachedStreamingResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		atomic.AddUint64(&c.errors, 1)
		return nil, false
	}

	return resp.Events, true
}

// SetStreamingResponse stores a streaming response in Redis.
func (c *RedisCache) SetStreamingResponse(key string, events []StreamEvent, ttl time.Duration) error {
	var totalSize int64
	for _, e := range events {
		totalSize += int64(len(e.Data))
	}

	resp := CachedStreamingResponse{
		Events:    events,
		TotalSize: totalSize,
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	return c.SetWithTTL("streaming", key, data, ttl)
}

// HybridCache combines in-memory LRU cache with Redis for multi-tier caching.
type HybridCache struct {
	local  *LRUCache
	redis  *RedisCache
	config HybridCacheConfig
}

// HybridCacheConfig configures the hybrid cache.
type HybridCacheConfig struct {
	// LocalCapacity is the capacity of the local LRU cache.
	LocalCapacity int `yaml:"local-capacity" json:"local_capacity"`
	// LocalTTLSeconds is the TTL for local cache entries.
	LocalTTLSeconds int `yaml:"local-ttl-seconds" json:"local_ttl_seconds"`
	// WriteThrough writes to both local and Redis on Set.
	WriteThrough bool `yaml:"write-through" json:"write_through"`
	// ReadThrough reads from Redis on local cache miss and populates local.
	ReadThrough bool `yaml:"read-through" json:"read_through"`
}

// DefaultHybridCacheConfig returns sensible defaults.
func DefaultHybridCacheConfig() HybridCacheConfig {
	return HybridCacheConfig{
		LocalCapacity:   1000,
		LocalTTLSeconds: 30,
		WriteThrough:    true,
		ReadThrough:     true,
	}
}

// NewHybridCache creates a new hybrid cache.
func NewHybridCache(redis *RedisCache, cfg HybridCacheConfig) *HybridCache {
	if cfg.LocalCapacity <= 0 {
		cfg.LocalCapacity = 1000
	}
	if cfg.LocalTTLSeconds <= 0 {
		cfg.LocalTTLSeconds = 30
	}

	return &HybridCache{
		local:  NewLRUCache(cfg.LocalCapacity, time.Duration(cfg.LocalTTLSeconds)*time.Second),
		redis:  redis,
		config: cfg,
	}
}

// Get retrieves a value, checking local cache first, then Redis.
func (h *HybridCache) Get(model, key string) ([]byte, bool) {
	// Check local cache first
	cacheKey := HashKey(model, key)
	if data := h.local.Get(cacheKey); data != nil {
		return data, true
	}

	// Check Redis if read-through is enabled
	if h.config.ReadThrough && h.redis != nil {
		if data, found := h.redis.Get(model, key); found {
			// Populate local cache
			h.local.Set(cacheKey, data)
			return data, true
		}
	}

	return nil, false
}

// Set stores a value in both local and Redis caches.
func (h *HybridCache) Set(model, key string, value []byte) error {
	cacheKey := HashKey(model, key)

	// Always write to local cache
	h.local.Set(cacheKey, value)

	// Write to Redis if write-through is enabled
	if h.config.WriteThrough && h.redis != nil {
		return h.redis.Set(model, key, value)
	}

	return nil
}

// SetWithTTL stores a value with custom TTL.
func (h *HybridCache) SetWithTTL(model, key string, value []byte, ttl time.Duration) error {
	cacheKey := HashKey(model, key)
	h.local.Set(cacheKey, value)

	if h.config.WriteThrough && h.redis != nil {
		return h.redis.SetWithTTL(model, key, value, ttl)
	}

	return nil
}

// Delete removes a value from both caches.
func (h *HybridCache) Delete(model, key string) error {
	cacheKey := HashKey(model, key)
	h.local.Delete(cacheKey)

	if h.redis != nil {
		return h.redis.Delete(model, key)
	}

	return nil
}

// Clear removes all values from both caches.
func (h *HybridCache) Clear() error {
	h.local.Clear()

	if h.redis != nil {
		return h.redis.Clear()
	}

	return nil
}

// Stats returns combined cache statistics.
func (h *HybridCache) Stats() HybridCacheStats {
	stats := HybridCacheStats{
		Local: h.local.Stats(),
	}

	if h.redis != nil {
		redisStats := h.redis.Stats()
		stats.Redis = &redisStats
	}

	return stats
}

// HybridCacheStats holds hybrid cache statistics.
type HybridCacheStats struct {
	Local CacheStats       `json:"local"`
	Redis *RedisCacheStats `json:"redis,omitempty"`
}

// Global Redis cache instance
var (
	globalRedisCache     *RedisCache
	globalRedisCacheMu   sync.RWMutex
)

// SetGlobalRedisCache sets the global Redis cache instance.
func SetGlobalRedisCache(cache *RedisCache) {
	globalRedisCacheMu.Lock()
	defer globalRedisCacheMu.Unlock()
	globalRedisCache = cache
}

// GetGlobalRedisCache returns the global Redis cache instance.
func GetGlobalRedisCache() *RedisCache {
	globalRedisCacheMu.RLock()
	defer globalRedisCacheMu.RUnlock()
	return globalRedisCache
}
