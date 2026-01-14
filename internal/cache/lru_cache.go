// Package cache provides caching utilities for the API proxy.
package cache

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"
)

// LRUCache is a thread-safe LRU cache with TTL support and metrics.
type LRUCache struct {
	mu       sync.RWMutex
	capacity int
	ttl      time.Duration
	items    map[string]*list.Element
	order    *list.List

	// Metrics
	hits   uint64
	misses uint64
}

type lruEntry struct {
	key       string
	value     []byte
	expiresAt time.Time
}

// NewLRUCache creates a new LRU cache with the specified capacity and TTL.
func NewLRUCache(capacity int, ttl time.Duration) *LRUCache {
	if capacity <= 0 {
		capacity = 1000
	}
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	c := &LRUCache{
		capacity: capacity,
		ttl:      ttl,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
	go c.startCleanup()
	return c
}

// Get retrieves a value from the cache.
// Returns nil if not found or expired.
func (c *LRUCache) Get(key string) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		atomic.AddUint64(&c.misses, 1)
		return nil
	}

	entry := elem.Value.(*lruEntry)
	if time.Now().After(entry.expiresAt) {
		c.removeElement(elem)
		atomic.AddUint64(&c.misses, 1)
		return nil
	}

	// Move to front (most recently used)
	c.order.MoveToFront(elem)
	atomic.AddUint64(&c.hits, 1)
	return entry.value
}

// Set stores a value in the cache.
func (c *LRUCache) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update existing entry
	if elem, ok := c.items[key]; ok {
		entry := elem.Value.(*lruEntry)
		entry.value = value
		entry.expiresAt = time.Now().Add(c.ttl)
		c.order.MoveToFront(elem)
		return
	}

	// Evict oldest if at capacity
	for c.order.Len() >= c.capacity {
		c.removeOldest()
	}

	// Add new entry
	entry := &lruEntry{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	elem := c.order.PushFront(entry)
	c.items[key] = elem
}

// Delete removes a key from the cache.
func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
	}
}

// Clear removes all entries from the cache.
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.order.Init()
}

// Len returns the number of items in the cache.
func (c *LRUCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.order.Len()
}

// Stats returns cache hit/miss statistics.
func (c *LRUCache) Stats() CacheStats {
	hits := atomic.LoadUint64(&c.hits)
	misses := atomic.LoadUint64(&c.misses)
	total := hits + misses
	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}
	return CacheStats{
		Hits:    hits,
		Misses:  misses,
		Size:    c.Len(),
		HitRate: hitRate,
	}
}

// ResetStats resets the hit/miss counters.
func (c *LRUCache) ResetStats() {
	atomic.StoreUint64(&c.hits, 0)
	atomic.StoreUint64(&c.misses, 0)
}

func (c *LRUCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*lruEntry)
	delete(c.items, entry.key)
	c.order.Remove(elem)
}

func (c *LRUCache) removeOldest() {
	elem := c.order.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

func (c *LRUCache) startCleanup() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()
	for range ticker.C {
		c.purgeExpired()
	}
}

func (c *LRUCache) purgeExpired() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	for elem := c.order.Back(); elem != nil; {
		entry := elem.Value.(*lruEntry)
		prev := elem.Prev()
		if now.After(entry.expiresAt) {
			c.removeElement(elem)
		}
		elem = prev
	}
}

// CacheStats holds cache statistics.
type CacheStats struct {
	Hits    uint64  `json:"hits"`
	Misses  uint64  `json:"misses"`
	Size    int     `json:"size"`
	HitRate float64 `json:"hit_rate_percent"`
}

// HashKey creates a cache key from multiple string inputs.
func HashKey(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0}) // separator
	}
	return hex.EncodeToString(h.Sum(nil))[:32]
}
