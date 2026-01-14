// Package cache provides caching utilities for the API proxy.
// This file implements streaming response caching for replay.
package cache

import (
	"bytes"
	"sync"
	"time"
)

// StreamingCache stores and replays streaming responses.
// It buffers SSE events and can replay them with timing preservation.
type StreamingCache struct {
	mu       sync.RWMutex
	cache    map[string]*streamingEntry
	capacity int
	ttl      time.Duration

	// Metrics
	hits   uint64
	misses uint64
}

// streamingEntry stores a complete streaming response.
type streamingEntry struct {
	events    []StreamEvent
	expiresAt time.Time
	totalSize int64
}

// StreamEvent represents a single SSE event.
type StreamEvent struct {
	Data      []byte        `json:"data"`
	EventType string        `json:"event_type,omitempty"`
	ID        string        `json:"id,omitempty"`
	Delay     time.Duration `json:"delay_ns"` // Delay from previous event
}

// StreamingCacheConfig configures the streaming cache.
type StreamingCacheConfig struct {
	// MaxEntries is the maximum number of cached streaming responses
	MaxEntries int
	// TTLSeconds is the TTL for cached responses
	TTLSeconds int
	// MaxEventSize is the maximum size of a single event in bytes
	MaxEventSize int64
	// MaxTotalSize is the maximum total size of events per response
	MaxTotalSize int64
	// PreserveTimings preserves original timing between events
	PreserveTimings bool
}

// DefaultStreamingCacheConfig returns sensible defaults.
func DefaultStreamingCacheConfig() StreamingCacheConfig {
	return StreamingCacheConfig{
		MaxEntries:      200,
		TTLSeconds:      60,
		MaxEventSize:    1024 * 1024,      // 1MB per event
		MaxTotalSize:    10 * 1024 * 1024, // 10MB total
		PreserveTimings: false,
	}
}

// NewStreamingCache creates a new streaming response cache.
func NewStreamingCache(cfg StreamingCacheConfig) *StreamingCache {
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = 200
	}
	if cfg.TTLSeconds <= 0 {
		cfg.TTLSeconds = 60
	}

	sc := &StreamingCache{
		cache:    make(map[string]*streamingEntry),
		capacity: cfg.MaxEntries,
		ttl:      time.Duration(cfg.TTLSeconds) * time.Second,
	}
	go sc.startCleanup()
	return sc
}

// StreamRecorder records streaming events for caching.
type StreamRecorder struct {
	mu        sync.Mutex
	key       string
	events    []StreamEvent
	lastEvent time.Time
	totalSize int64
	maxSize   int64
	cache     *StreamingCache
	started   bool
}

// NewStreamRecorder creates a recorder for a streaming response.
func (sc *StreamingCache) NewStreamRecorder(key string, maxSize int64) *StreamRecorder {
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024 // 10MB default
	}
	return &StreamRecorder{
		key:       key,
		events:    make([]StreamEvent, 0, 100),
		maxSize:   maxSize,
		cache:     sc,
	}
}

// RecordEvent records a streaming event.
func (r *StreamRecorder) RecordEvent(data []byte, eventType, id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	var delay time.Duration
	if r.started {
		delay = now.Sub(r.lastEvent)
	}
	r.started = true
	r.lastEvent = now

	size := int64(len(data))
	if r.totalSize+size > r.maxSize {
		// Exceeded max size, stop recording
		return
	}

	r.events = append(r.events, StreamEvent{
		Data:      bytes.Clone(data),
		EventType: eventType,
		ID:        id,
		Delay:     delay,
	})
	r.totalSize += size
}

// Commit saves the recorded streaming response to cache.
func (r *StreamRecorder) Commit() {
	r.mu.Lock()
	events := make([]StreamEvent, len(r.events))
	copy(events, r.events)
	totalSize := r.totalSize
	r.mu.Unlock()

	if len(events) == 0 {
		return
	}

	r.cache.set(r.key, events, totalSize)
}

// set stores a streaming response in the cache.
func (sc *StreamingCache) set(key string, events []StreamEvent, totalSize int64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Evict oldest if at capacity
	for len(sc.cache) >= sc.capacity {
		sc.evictOldest()
	}

	sc.cache[key] = &streamingEntry{
		events:    events,
		expiresAt: time.Now().Add(sc.ttl),
		totalSize: totalSize,
	}
}

// Get retrieves a cached streaming response.
func (sc *StreamingCache) Get(key string) ([]StreamEvent, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	entry, exists := sc.cache[key]
	if !exists {
		sc.misses++
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		sc.misses++
		return nil, false
	}

	sc.hits++
	events := make([]StreamEvent, len(entry.events))
	copy(events, entry.events)
	return events, true
}

// Replay sends cached events through a callback with optional timing preservation.
func (sc *StreamingCache) Replay(key string, preserveTimings bool, callback func(event StreamEvent) error) error {
	events, exists := sc.Get(key)
	if !exists {
		return nil
	}

	for _, event := range events {
		if preserveTimings && event.Delay > 0 {
			time.Sleep(event.Delay)
		}
		if err := callback(event); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes an entry from the cache.
func (sc *StreamingCache) Delete(key string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.cache, key)
}

// Clear removes all entries from the cache.
func (sc *StreamingCache) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.cache = make(map[string]*streamingEntry)
}

// Stats returns cache statistics.
func (sc *StreamingCache) Stats() StreamingCacheStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	var totalEvents int
	var totalSize int64
	for _, entry := range sc.cache {
		totalEvents += len(entry.events)
		totalSize += entry.totalSize
	}

	total := sc.hits + sc.misses
	var hitRate float64
	if total > 0 {
		hitRate = float64(sc.hits) / float64(total) * 100
	}

	return StreamingCacheStats{
		Entries:     len(sc.cache),
		TotalEvents: totalEvents,
		TotalSize:   totalSize,
		Hits:        sc.hits,
		Misses:      sc.misses,
		HitRate:     hitRate,
	}
}

func (sc *StreamingCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range sc.cache {
		if oldestKey == "" || entry.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expiresAt
		}
	}

	if oldestKey != "" {
		delete(sc.cache, oldestKey)
	}
}

func (sc *StreamingCache) startCleanup() {
	ticker := time.NewTicker(sc.ttl / 2)
	defer ticker.Stop()
	for range ticker.C {
		sc.purgeExpired()
	}
}

func (sc *StreamingCache) purgeExpired() {
	now := time.Now()
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for key, entry := range sc.cache {
		if now.After(entry.expiresAt) {
			delete(sc.cache, key)
		}
	}
}

// StreamingCacheStats holds statistics for streaming cache.
type StreamingCacheStats struct {
	Entries     int     `json:"entries"`
	TotalEvents int     `json:"total_events"`
	TotalSize   int64   `json:"total_size_bytes"`
	Hits        uint64  `json:"hits"`
	Misses      uint64  `json:"misses"`
	HitRate     float64 `json:"hit_rate_percent"`
}

// Global streaming cache instance
var (
	globalStreamingCache     *StreamingCache
	globalStreamingCacheOnce sync.Once
)

// GetStreamingCache returns the global streaming cache.
func GetStreamingCache() *StreamingCache {
	globalStreamingCacheOnce.Do(func() {
		globalStreamingCache = NewStreamingCache(DefaultStreamingCacheConfig())
	})
	return globalStreamingCache
}

// InitStreamingCache initializes the global streaming cache with custom config.
func InitStreamingCache(cfg StreamingCacheConfig) *StreamingCache {
	globalStreamingCacheOnce.Do(func() {
		globalStreamingCache = NewStreamingCache(cfg)
	})
	return globalStreamingCache
}
