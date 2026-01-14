// Package cache provides caching utilities for the API proxy.
package cache

import (
	"sync"
	"time"
)

// ResponseCache caches API responses for identical requests.
// Uses LRU eviction and TTL expiration.
type ResponseCache struct {
	cache *LRUCache
}

// ResponseCacheConfig configures the response cache.
type ResponseCacheConfig struct {
	// MaxEntries is the maximum number of cached responses (default: 500)
	MaxEntries int
	// TTLSeconds is the TTL for cached responses in seconds (default: 30)
	TTLSeconds int
}

// DefaultResponseCacheConfig returns sensible defaults.
func DefaultResponseCacheConfig() ResponseCacheConfig {
	return ResponseCacheConfig{
		MaxEntries: 500,
		TTLSeconds: 30,
	}
}

// NewResponseCache creates a new response cache.
func NewResponseCache(cfg ResponseCacheConfig) *ResponseCache {
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = 500
	}
	if cfg.TTLSeconds <= 0 {
		cfg.TTLSeconds = 30
	}
	return &ResponseCache{
		cache: NewLRUCache(cfg.MaxEntries, time.Duration(cfg.TTLSeconds)*time.Second),
	}
}

// Get retrieves a cached response for the given request hash.
func (c *ResponseCache) Get(model string, requestHash string) []byte {
	key := HashKey(model, requestHash)
	return c.cache.Get(key)
}

// Set stores a response in the cache.
func (c *ResponseCache) Set(model string, requestHash string, response []byte) {
	// Don't cache empty responses
	if len(response) == 0 {
		return
	}
	key := HashKey(model, requestHash)
	c.cache.Set(key, response)
}

// Stats returns cache statistics.
func (c *ResponseCache) Stats() CacheStats {
	return c.cache.Stats()
}

// Clear removes all cached responses.
func (c *ResponseCache) Clear() {
	c.cache.Clear()
}

// Global response cache instance
var (
	globalResponseCache     *ResponseCache
	globalResponseCacheOnce sync.Once
)

// GetResponseCache returns the global response cache.
func GetResponseCache() *ResponseCache {
	globalResponseCacheOnce.Do(func() {
		globalResponseCache = NewResponseCache(DefaultResponseCacheConfig())
	})
	return globalResponseCache
}

// RequestDeduplicator prevents duplicate in-flight requests.
type RequestDeduplicator struct {
	mu       sync.Mutex
	inflight map[string]*inflightRequest
}

type inflightRequest struct {
	done     chan struct{}
	response []byte
	err      error
}

// NewRequestDeduplicator creates a new request deduplicator.
func NewRequestDeduplicator() *RequestDeduplicator {
	return &RequestDeduplicator{
		inflight: make(map[string]*inflightRequest),
	}
}

// Do executes the function, deduplicating identical concurrent requests.
// If a request with the same key is already in-flight, waits for its result.
func (d *RequestDeduplicator) Do(key string, fn func() ([]byte, error)) ([]byte, error) {
	d.mu.Lock()

	// Check if request is already in-flight
	if req, ok := d.inflight[key]; ok {
		d.mu.Unlock()
		<-req.done
		return req.response, req.err
	}

	// Create new in-flight request
	req := &inflightRequest{
		done: make(chan struct{}),
	}
	d.inflight[key] = req
	d.mu.Unlock()

	// Execute the function
	req.response, req.err = fn()
	close(req.done)

	// Clean up
	d.mu.Lock()
	delete(d.inflight, key)
	d.mu.Unlock()

	return req.response, req.err
}

// Global request deduplicator
var (
	globalDeduplicator     *RequestDeduplicator
	globalDeduplicatorOnce sync.Once
)

// GetRequestDeduplicator returns the global request deduplicator.
func GetRequestDeduplicator() *RequestDeduplicator {
	globalDeduplicatorOnce.Do(func() {
		globalDeduplicator = NewRequestDeduplicator()
	})
	return globalDeduplicator
}
