// Package cache provides caching utilities for the API proxy.
package cache

import (
	"sync"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/registry"
)

// ModelListCache caches model list responses with TTL.
type ModelListCache struct {
	mu      sync.RWMutex
	entries map[string]*modelListEntry
	ttl     time.Duration
}

type modelListEntry struct {
	models    []*registry.ModelInfo
	expiresAt time.Time
}

// DefaultModelListCacheTTL is the default TTL for model list cache.
const DefaultModelListCacheTTL = 5 * time.Minute

// NewModelListCache creates a new model list cache.
func NewModelListCache(ttl time.Duration) *ModelListCache {
	if ttl <= 0 {
		ttl = DefaultModelListCacheTTL
	}
	c := &ModelListCache{
		entries: make(map[string]*modelListEntry),
		ttl:     ttl,
	}
	go c.startCleanup()
	return c
}

// Get retrieves cached models for an auth ID.
func (c *ModelListCache) Get(authID string) []*registry.ModelInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[authID]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil
	}
	return entry.models
}

// Set stores models for an auth ID.
func (c *ModelListCache) Set(authID string, models []*registry.ModelInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[authID] = &modelListEntry{
		models:    models,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Invalidate removes cached models for an auth ID.
func (c *ModelListCache) Invalidate(authID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, authID)
}

// Clear removes all cached entries.
func (c *ModelListCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*modelListEntry)
}

func (c *ModelListCache) startCleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()
	for range ticker.C {
		c.purgeExpired()
	}
}

func (c *ModelListCache) purgeExpired() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
}

// Global model list cache instance
var (
	globalModelListCache     *ModelListCache
	globalModelListCacheOnce sync.Once
)

// GetModelListCache returns the global model list cache.
func GetModelListCache() *ModelListCache {
	globalModelListCacheOnce.Do(func() {
		globalModelListCache = NewModelListCache(DefaultModelListCacheTTL)
	})
	return globalModelListCache
}
