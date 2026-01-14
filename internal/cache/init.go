// Package cache provides caching utilities for the API proxy.
// This file provides initialization functions for the cache system.
package cache

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// CacheSystem holds all cache instances.
type CacheSystem struct {
	LRU       *LRUCache
	Semantic  *SemanticCache
	Streaming *StreamingCache
	Redis     *RedisCache
	Hybrid    *HybridCache

	config    CacheSystemConfig
	redisOK   bool
	mu        sync.RWMutex
}

// CacheSystemConfig configures the entire cache system.
type CacheSystemConfig struct {
	// LRU cache settings
	LRUCapacity   int
	LRUTTLSeconds int

	// Redis settings
	RedisEnabled        bool
	RedisAddress        string
	RedisPassword       string
	RedisDatabase       int
	RedisKeyPrefix      string
	RedisTTLSeconds     int
	RedisPoolSize       int
	RedisDialTimeoutMs  int
	RedisReadTimeoutMs  int
	RedisWriteTimeoutMs int
	RedisEnableTLS      bool
	RedisMaxRetries     int

	// Semantic cache settings
	SemanticEnabled           bool
	SemanticMaxEntries        int
	SemanticTTLSeconds        int
	SemanticSimilarityThreshold float64

	// Streaming cache settings
	StreamingEnabled        bool
	StreamingMaxEntries     int
	StreamingTTLSeconds     int
	StreamingMaxEventSize   int64
	StreamingMaxTotalSize   int64
	StreamingPreserveTimings bool

	// Hybrid cache settings
	HybridLocalCapacity   int
	HybridLocalTTLSeconds int
	HybridWriteThrough    bool
	HybridReadThrough     bool
}

// DefaultCacheSystemConfig returns sensible defaults.
func DefaultCacheSystemConfig() CacheSystemConfig {
	return CacheSystemConfig{
		LRUCapacity:   1000,
		LRUTTLSeconds: 60,

		RedisEnabled:        false,
		RedisAddress:        "localhost:6379",
		RedisKeyPrefix:      "shinapi:",
		RedisTTLSeconds:     60,
		RedisPoolSize:       10,
		RedisDialTimeoutMs:  5000,
		RedisReadTimeoutMs:  3000,
		RedisWriteTimeoutMs: 3000,
		RedisMaxRetries:     3,

		SemanticEnabled:           false,
		SemanticMaxEntries:        1000,
		SemanticTTLSeconds:        60,
		SemanticSimilarityThreshold: 0.85,

		StreamingEnabled:        true,
		StreamingMaxEntries:     200,
		StreamingTTLSeconds:     60,
		StreamingMaxEventSize:   1024 * 1024,
		StreamingMaxTotalSize:   10 * 1024 * 1024,
		StreamingPreserveTimings: false,

		HybridLocalCapacity:   1000,
		HybridLocalTTLSeconds: 30,
		HybridWriteThrough:    true,
		HybridReadThrough:     true,
	}
}

var (
	globalCacheSystem     *CacheSystem
	globalCacheSystemOnce sync.Once
)

// InitCacheSystem initializes the global cache system.
func InitCacheSystem(cfg CacheSystemConfig) *CacheSystem {
	globalCacheSystemOnce.Do(func() {
		globalCacheSystem = newCacheSystem(cfg)
	})
	return globalCacheSystem
}

// GetCacheSystem returns the global cache system.
func GetCacheSystem() *CacheSystem {
	if globalCacheSystem == nil {
		return InitCacheSystem(DefaultCacheSystemConfig())
	}
	return globalCacheSystem
}

func newCacheSystem(cfg CacheSystemConfig) *CacheSystem {
	cs := &CacheSystem{
		config: cfg,
	}

	// Initialize LRU cache
	cs.LRU = NewLRUCache(cfg.LRUCapacity, time.Duration(cfg.LRUTTLSeconds)*time.Second)
	log.Infof("Cache: LRU cache initialized (capacity=%d, ttl=%ds)", cfg.LRUCapacity, cfg.LRUTTLSeconds)

	// Initialize Redis if enabled
	if cfg.RedisEnabled {
		cs.initRedis(cfg)
	}

	// Initialize semantic cache if enabled
	if cfg.SemanticEnabled {
		cs.Semantic = NewSemanticCache(SemanticCacheConfig{
			MaxEntries:          cfg.SemanticMaxEntries,
			TTLSeconds:          cfg.SemanticTTLSeconds,
			SimilarityThreshold: cfg.SemanticSimilarityThreshold,
			NGramSize:           3,
			NormalizeCase:       true,
			NormalizeWhitespace: true,
		})
		log.Infof("Cache: Semantic cache initialized (max=%d, threshold=%.2f)", 
			cfg.SemanticMaxEntries, cfg.SemanticSimilarityThreshold)
	}

	// Initialize streaming cache if enabled
	if cfg.StreamingEnabled {
		cs.Streaming = NewStreamingCache(StreamingCacheConfig{
			MaxEntries:      cfg.StreamingMaxEntries,
			TTLSeconds:      cfg.StreamingTTLSeconds,
			MaxEventSize:    cfg.StreamingMaxEventSize,
			MaxTotalSize:    cfg.StreamingMaxTotalSize,
			PreserveTimings: cfg.StreamingPreserveTimings,
		})
		log.Infof("Cache: Streaming cache initialized (max=%d)", cfg.StreamingMaxEntries)
	}

	return cs
}

func (cs *CacheSystem) initRedis(cfg CacheSystemConfig) {
	redisCfg := RedisCacheConfig{
		Address:           cfg.RedisAddress,
		Password:          cfg.RedisPassword,
		Database:          cfg.RedisDatabase,
		KeyPrefix:         cfg.RedisKeyPrefix,
		DefaultTTLSeconds: cfg.RedisTTLSeconds,
		MaxRetries:        cfg.RedisMaxRetries,
		PoolSize:          cfg.RedisPoolSize,
		DialTimeoutMs:     cfg.RedisDialTimeoutMs,
		ReadTimeoutMs:     cfg.RedisReadTimeoutMs,
		WriteTimeoutMs:    cfg.RedisWriteTimeoutMs,
		EnableTLS:         cfg.RedisEnableTLS,
		Enabled:           true,
	}

	// Create go-redis client
	goRedisClient := NewGoRedisClientFromRedisCacheConfig(redisCfg)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.RedisDialTimeoutMs)*time.Millisecond)
	defer cancel()

	if err := goRedisClient.Ping(ctx); err != nil {
		log.Warnf("Cache: Redis connection failed (%s): %v - running without Redis", cfg.RedisAddress, err)
		cs.redisOK = false
		return
	}

	cs.Redis = NewRedisCache(goRedisClient, redisCfg)
	cs.redisOK = true
	SetGlobalRedisCache(cs.Redis)

	log.Infof("Cache: Redis connected (%s, db=%d, prefix=%s)", 
		cfg.RedisAddress, cfg.RedisDatabase, cfg.RedisKeyPrefix)

	// Initialize hybrid cache if Redis is available
	cs.Hybrid = NewHybridCache(cs.Redis, HybridCacheConfig{
		LocalCapacity:   cfg.HybridLocalCapacity,
		LocalTTLSeconds: cfg.HybridLocalTTLSeconds,
		WriteThrough:    cfg.HybridWriteThrough,
		ReadThrough:     cfg.HybridReadThrough,
	})
	log.Info("Cache: Hybrid cache initialized (L1: LRU, L2: Redis)")
}

// IsRedisAvailable returns whether Redis is connected and available.
func (cs *CacheSystem) IsRedisAvailable() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.redisOK && cs.Redis != nil
}

// Get retrieves from the best available cache.
func (cs *CacheSystem) Get(model, key string) ([]byte, bool) {
	// Try hybrid cache first if available
	if cs.Hybrid != nil {
		return cs.Hybrid.Get(model, key)
	}

	// Fall back to LRU
	cacheKey := HashKey(model, key)
	if data := cs.LRU.Get(cacheKey); data != nil {
		return data, true
	}

	return nil, false
}

// Set stores in the best available cache.
func (cs *CacheSystem) Set(model, key string, value []byte) {
	// Use hybrid cache if available
	if cs.Hybrid != nil {
		cs.Hybrid.Set(model, key, value)
		return
	}

	// Fall back to LRU
	cacheKey := HashKey(model, key)
	cs.LRU.Set(cacheKey, value)
}

// Stats returns combined cache statistics.
func (cs *CacheSystem) Stats() CacheSystemStats {
	stats := CacheSystemStats{
		LRU: cs.LRU.Stats(),
	}

	if cs.Redis != nil {
		redisStats := cs.Redis.Stats()
		stats.Redis = &redisStats
		stats.RedisConnected = cs.redisOK
	}

	if cs.Semantic != nil {
		semanticStats := cs.Semantic.Stats()
		stats.Semantic = &semanticStats
	}

	if cs.Streaming != nil {
		streamingStats := cs.Streaming.Stats()
		stats.Streaming = &streamingStats
	}

	if cs.Hybrid != nil {
		hybridStats := cs.Hybrid.Stats()
		stats.Hybrid = &hybridStats
	}

	return stats
}

// Close closes all cache connections.
func (cs *CacheSystem) Close() error {
	if cs.Redis != nil {
		return cs.Redis.Close()
	}
	return nil
}

// CacheSystemStats holds stats for all caches.
type CacheSystemStats struct {
	LRU            CacheStats          `json:"lru"`
	Redis          *RedisCacheStats    `json:"redis,omitempty"`
	RedisConnected bool                `json:"redis_connected"`
	Semantic       *SemanticCacheStats `json:"semantic,omitempty"`
	Streaming      *StreamingCacheStats `json:"streaming,omitempty"`
	Hybrid         *HybridCacheStats   `json:"hybrid,omitempty"`
}
