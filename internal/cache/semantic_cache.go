// Package cache provides caching utilities for the API proxy.
// This file implements semantic caching based on prompt similarity.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// SemanticCache provides caching based on prompt similarity rather than exact match.
// It uses normalized text hashing and n-gram similarity for cache lookups.
type SemanticCache struct {
	mu       sync.RWMutex
	cache    *LRUCache
	index    map[string][]semanticEntry // normalized hash -> list of similar entries
	config   SemanticCacheConfig
	
	// Metrics
	semanticHits   uint64
	semanticMisses uint64
}

// semanticEntry stores a cache entry with its similarity data.
type semanticEntry struct {
	key           string
	normalizedKey string
	ngrams        map[string]struct{}
	expiresAt     time.Time
}

// SemanticCacheConfig configures the semantic cache behavior.
type SemanticCacheConfig struct {
	// MaxEntries is the maximum number of cached responses
	MaxEntries int
	// TTLSeconds is the default TTL for cached responses
	TTLSeconds int
	// SimilarityThreshold is the minimum Jaccard similarity (0.0-1.0) for a cache hit
	SimilarityThreshold float64
	// NGramSize is the size of n-grams for similarity calculation (default: 3)
	NGramSize int
	// NormalizeCase lowercases text for comparison
	NormalizeCase bool
	// NormalizeWhitespace collapses whitespace for comparison
	NormalizeWhitespace bool
	// StripPunctuation removes punctuation for comparison
	StripPunctuation bool
}

// DefaultSemanticCacheConfig returns sensible defaults.
func DefaultSemanticCacheConfig() SemanticCacheConfig {
	return SemanticCacheConfig{
		MaxEntries:          1000,
		TTLSeconds:          60,
		SimilarityThreshold: 0.85, // 85% similarity required
		NGramSize:           3,
		NormalizeCase:       true,
		NormalizeWhitespace: true,
		StripPunctuation:    false,
	}
}

// NewSemanticCache creates a new semantic cache.
func NewSemanticCache(cfg SemanticCacheConfig) *SemanticCache {
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = 1000
	}
	if cfg.TTLSeconds <= 0 {
		cfg.TTLSeconds = 60
	}
	if cfg.SimilarityThreshold <= 0 || cfg.SimilarityThreshold > 1 {
		cfg.SimilarityThreshold = 0.85
	}
	if cfg.NGramSize <= 0 {
		cfg.NGramSize = 3
	}

	sc := &SemanticCache{
		cache:  NewLRUCache(cfg.MaxEntries, time.Duration(cfg.TTLSeconds)*time.Second),
		index:  make(map[string][]semanticEntry),
		config: cfg,
	}
	go sc.startCleanup()
	return sc
}

// Get retrieves a cached response based on semantic similarity.
// Returns the response and a boolean indicating if a match was found.
func (sc *SemanticCache) Get(model, prompt string) ([]byte, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	normalizedPrompt := sc.normalize(prompt)
	bucket := sc.bucketKey(normalizedPrompt)
	promptNgrams := sc.generateNgrams(normalizedPrompt)

	entries, exists := sc.index[bucket]
	if !exists {
		sc.semanticMisses++
		return nil, false
	}

	now := time.Now()
	var bestMatch *semanticEntry
	var bestSimilarity float64

	for i := range entries {
		entry := &entries[i]
		if now.After(entry.expiresAt) {
			continue
		}
		similarity := sc.jaccardSimilarity(promptNgrams, entry.ngrams)
		if similarity >= sc.config.SimilarityThreshold && similarity > bestSimilarity {
			bestSimilarity = similarity
			bestMatch = entry
		}
	}

	if bestMatch != nil {
		cacheKey := HashKey(model, bestMatch.key)
		if data := sc.cache.Get(cacheKey); data != nil {
			sc.semanticHits++
			return data, true
		}
	}

	sc.semanticMisses++
	return nil, false
}

// Set stores a response in the semantic cache.
func (sc *SemanticCache) Set(model, prompt string, response []byte) {
	if len(response) == 0 {
		return
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	normalizedPrompt := sc.normalize(prompt)
	bucket := sc.bucketKey(normalizedPrompt)
	cacheKey := HashKey(model, prompt)

	// Store in underlying LRU cache
	sc.cache.Set(cacheKey, response)

	// Add to semantic index
	entry := semanticEntry{
		key:           prompt,
		normalizedKey: normalizedPrompt,
		ngrams:        sc.generateNgrams(normalizedPrompt),
		expiresAt:     time.Now().Add(time.Duration(sc.config.TTLSeconds) * time.Second),
	}

	entries := sc.index[bucket]
	// Check if we already have this exact entry
	for i := range entries {
		if entries[i].key == prompt {
			entries[i] = entry
			return
		}
	}
	sc.index[bucket] = append(entries, entry)
}

// SetWithTTL stores a response with a custom TTL.
func (sc *SemanticCache) SetWithTTL(model, prompt string, response []byte, ttl time.Duration) {
	if len(response) == 0 {
		return
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	normalizedPrompt := sc.normalize(prompt)
	bucket := sc.bucketKey(normalizedPrompt)
	cacheKey := HashKey(model, prompt)

	// Create a temporary cache entry with custom TTL
	tempCache := NewLRUCache(1, ttl)
	tempCache.Set(cacheKey, response)
	sc.cache.Set(cacheKey, response)

	entry := semanticEntry{
		key:           prompt,
		normalizedKey: normalizedPrompt,
		ngrams:        sc.generateNgrams(normalizedPrompt),
		expiresAt:     time.Now().Add(ttl),
	}

	entries := sc.index[bucket]
	for i := range entries {
		if entries[i].key == prompt {
			entries[i] = entry
			return
		}
	}
	sc.index[bucket] = append(entries, entry)
}

// normalize applies normalization rules to a prompt.
func (sc *SemanticCache) normalize(text string) string {
	if sc.config.NormalizeCase {
		text = strings.ToLower(text)
	}
	if sc.config.NormalizeWhitespace {
		text = normalizeWhitespace(text)
	}
	if sc.config.StripPunctuation {
		text = stripPunctuation(text)
	}
	return text
}

// bucketKey creates a bucket key for indexing similar prompts.
// Uses first N characters of hash for bucketing.
func (sc *SemanticCache) bucketKey(normalizedText string) string {
	h := sha256.Sum256([]byte(normalizedText))
	return hex.EncodeToString(h[:])[:8]
}

// generateNgrams creates n-grams from normalized text.
func (sc *SemanticCache) generateNgrams(text string) map[string]struct{} {
	ngrams := make(map[string]struct{})
	n := sc.config.NGramSize
	if len(text) < n {
		ngrams[text] = struct{}{}
		return ngrams
	}
	for i := 0; i <= len(text)-n; i++ {
		ngrams[text[i:i+n]] = struct{}{}
	}
	return ngrams
}

// jaccardSimilarity calculates the Jaccard similarity between two n-gram sets.
func (sc *SemanticCache) jaccardSimilarity(a, b map[string]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	intersection := 0
	for k := range a {
		if _, exists := b[k]; exists {
			intersection++
		}
	}

	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}

// Stats returns semantic cache statistics.
func (sc *SemanticCache) Stats() SemanticCacheStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	lruStats := sc.cache.Stats()
	indexSize := 0
	for _, entries := range sc.index {
		indexSize += len(entries)
	}

	total := sc.semanticHits + sc.semanticMisses
	var hitRate float64
	if total > 0 {
		hitRate = float64(sc.semanticHits) / float64(total) * 100
	}

	return SemanticCacheStats{
		CacheStats:      lruStats,
		SemanticHits:    sc.semanticHits,
		SemanticMisses:  sc.semanticMisses,
		SemanticHitRate: hitRate,
		IndexSize:       indexSize,
		BucketCount:     len(sc.index),
	}
}

// Clear removes all entries from the cache.
func (sc *SemanticCache) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.cache.Clear()
	sc.index = make(map[string][]semanticEntry)
}

func (sc *SemanticCache) startCleanup() {
	ticker := time.NewTicker(time.Duration(sc.config.TTLSeconds/2) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		sc.purgeExpired()
	}
}

func (sc *SemanticCache) purgeExpired() {
	now := time.Now()
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for bucket, entries := range sc.index {
		var valid []semanticEntry
		for _, entry := range entries {
			if now.Before(entry.expiresAt) {
				valid = append(valid, entry)
			}
		}
		if len(valid) == 0 {
			delete(sc.index, bucket)
		} else {
			sc.index[bucket] = valid
		}
	}
}

// SemanticCacheStats holds statistics for semantic caching.
type SemanticCacheStats struct {
	CacheStats      CacheStats `json:"cache_stats"`
	SemanticHits    uint64     `json:"semantic_hits"`
	SemanticMisses  uint64     `json:"semantic_misses"`
	SemanticHitRate float64    `json:"semantic_hit_rate_percent"`
	IndexSize       int        `json:"index_size"`
	BucketCount     int        `json:"bucket_count"`
}

// normalizeWhitespace collapses multiple whitespace characters to single spaces.
func normalizeWhitespace(s string) string {
	var builder strings.Builder
	lastWasSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !lastWasSpace {
				builder.WriteRune(' ')
				lastWasSpace = true
			}
		} else {
			builder.WriteRune(r)
			lastWasSpace = false
		}
	}
	return strings.TrimSpace(builder.String())
}

// stripPunctuation removes punctuation from a string.
func stripPunctuation(s string) string {
	var builder strings.Builder
	for _, r := range s {
		if !unicode.IsPunct(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// Global semantic cache instance
var (
	globalSemanticCache     *SemanticCache
	globalSemanticCacheOnce sync.Once
)

// GetSemanticCache returns the global semantic cache.
func GetSemanticCache() *SemanticCache {
	globalSemanticCacheOnce.Do(func() {
		globalSemanticCache = NewSemanticCache(DefaultSemanticCacheConfig())
	})
	return globalSemanticCache
}

// InitSemanticCache initializes the global semantic cache with custom config.
func InitSemanticCache(cfg SemanticCacheConfig) *SemanticCache {
	globalSemanticCacheOnce.Do(func() {
		globalSemanticCache = NewSemanticCache(cfg)
	})
	return globalSemanticCache
}

// CacheKeyConfig configures how cache keys are generated.
type CacheKeyConfig struct {
	// IncludeModel includes model name in cache key
	IncludeModel bool `yaml:"include-model" json:"include_model"`
	// IncludeSystemPrompt includes system prompt in cache key
	IncludeSystemPrompt bool `yaml:"include-system-prompt" json:"include_system_prompt"`
	// IncludeTemperature includes temperature in cache key
	IncludeTemperature bool `yaml:"include-temperature" json:"include_temperature"`
	// IncludeMaxTokens includes max_tokens in cache key
	IncludeMaxTokens bool `yaml:"include-max-tokens" json:"include_max_tokens"`
	// IncludeTools includes tools/functions in cache key
	IncludeTools bool `yaml:"include-tools" json:"include_tools"`
	// ExcludeFields lists field names to exclude from cache key
	ExcludeFields []string `yaml:"exclude-fields" json:"exclude_fields"`
}

// DefaultCacheKeyConfig returns sensible defaults.
func DefaultCacheKeyConfig() CacheKeyConfig {
	return CacheKeyConfig{
		IncludeModel:        true,
		IncludeSystemPrompt: true,
		IncludeTemperature:  false, // Usually don't cache different temps
		IncludeMaxTokens:    false,
		IncludeTools:        true,
		ExcludeFields:       []string{"stream", "user", "metadata"},
	}
}

// GenerateCacheKey creates a cache key based on the configuration.
func GenerateCacheKey(cfg CacheKeyConfig, model, systemPrompt, userPrompt string, temperature float64, maxTokens int, tools []string) string {
	var parts []string

	if cfg.IncludeModel && model != "" {
		parts = append(parts, "model:"+model)
	}
	if cfg.IncludeSystemPrompt && systemPrompt != "" {
		parts = append(parts, "sys:"+systemPrompt)
	}
	if userPrompt != "" {
		parts = append(parts, "user:"+userPrompt)
	}
	if cfg.IncludeTemperature {
		parts = append(parts, "temp:"+strings.TrimRight(strings.TrimRight(
			strings.Replace(string(rune(int(temperature*1000))), ".", "", 1), "0"), "."))
	}
	if cfg.IncludeMaxTokens && maxTokens > 0 {
		parts = append(parts, "max:"+string(rune(maxTokens)))
	}
	if cfg.IncludeTools && len(tools) > 0 {
		sort.Strings(tools)
		parts = append(parts, "tools:"+strings.Join(tools, ","))
	}

	combined := strings.Join(parts, "|")
	return HashKey(combined)
}
