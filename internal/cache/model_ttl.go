// Package cache provides caching utilities for the API proxy.
// This file implements model-specific TTL configuration.
package cache

import (
	"strings"
	"sync"
	"time"
)

// ModelTTLConfig manages per-model cache TTL settings.
type ModelTTLConfig struct {
	mu          sync.RWMutex
	defaultTTL  time.Duration
	modelTTLs   map[string]time.Duration
	patternTTLs []patternTTL
}

type patternTTL struct {
	pattern string
	ttl     time.Duration
}

// NewModelTTLConfig creates a new model TTL configuration.
func NewModelTTLConfig(defaultTTL time.Duration) *ModelTTLConfig {
	if defaultTTL <= 0 {
		defaultTTL = 60 * time.Second
	}
	return &ModelTTLConfig{
		defaultTTL:  defaultTTL,
		modelTTLs:   make(map[string]time.Duration),
		patternTTLs: make([]patternTTL, 0),
	}
}

// SetModelTTL sets the TTL for a specific model.
func (c *ModelTTLConfig) SetModelTTL(model string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.modelTTLs[model] = ttl
}

// SetPatternTTL sets the TTL for models matching a pattern.
// Patterns support wildcards: * matches any sequence, ? matches any single character.
func (c *ModelTTLConfig) SetPatternTTL(pattern string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.patternTTLs = append(c.patternTTLs, patternTTL{pattern: pattern, ttl: ttl})
}

// GetTTL returns the TTL for a specific model.
// Checks exact matches first, then patterns, then returns default.
func (c *ModelTTLConfig) GetTTL(model string) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check exact match
	if ttl, exists := c.modelTTLs[model]; exists {
		return ttl
	}

	// Check patterns
	for _, p := range c.patternTTLs {
		if matchPattern(p.pattern, model) {
			return p.ttl
		}
	}

	return c.defaultTTL
}

// SetDefaultTTL sets the default TTL for models without specific configuration.
func (c *ModelTTLConfig) SetDefaultTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.defaultTTL = ttl
}

// matchPattern matches a string against a pattern with * and ? wildcards.
func matchPattern(pattern, s string) bool {
	// Empty pattern only matches empty string
	if pattern == "" {
		return s == ""
	}

	// Simple optimization for patterns without wildcards
	if !strings.ContainsAny(pattern, "*?") {
		return pattern == s
	}

	return matchPatternRecursive(pattern, s)
}

func matchPatternRecursive(pattern, s string) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		case '*':
			// Skip consecutive *
			for len(pattern) > 0 && pattern[0] == '*' {
				pattern = pattern[1:]
			}
			if pattern == "" {
				return true
			}
			for i := 0; i <= len(s); i++ {
				if matchPatternRecursive(pattern, s[i:]) {
					return true
				}
			}
			return false
		case '?':
			if len(s) == 0 {
				return false
			}
			pattern = pattern[1:]
			s = s[1:]
		default:
			if len(s) == 0 || pattern[0] != s[0] {
				return false
			}
			pattern = pattern[1:]
			s = s[1:]
		}
	}
	return s == ""
}

// ModelCacheConfig holds per-model cache configuration.
type ModelCacheConfig struct {
	// Model is the model name or pattern
	Model string `yaml:"model" json:"model"`
	// TTLSeconds is the cache TTL in seconds
	TTLSeconds int `yaml:"ttl-seconds" json:"ttl_seconds"`
	// Enabled controls whether caching is enabled for this model
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	// MaxEntries overrides the max cache entries for this model
	MaxEntries int `yaml:"max-entries,omitempty" json:"max_entries,omitempty"`
	// SimilarityThreshold overrides semantic cache threshold for this model
	SimilarityThreshold float64 `yaml:"similarity-threshold,omitempty" json:"similarity_threshold,omitempty"`
}

// CacheConfigManager manages cache configuration for different models.
type CacheConfigManager struct {
	mu             sync.RWMutex
	defaultConfig  CacheDefaults
	modelConfigs   map[string]*ModelCacheConfig
	patternConfigs []ModelCacheConfig
	ttlConfig      *ModelTTLConfig
}

// CacheDefaults holds default cache settings.
type CacheDefaults struct {
	Enabled             bool    `yaml:"enabled" json:"enabled"`
	TTLSeconds          int     `yaml:"ttl-seconds" json:"ttl_seconds"`
	MaxEntries          int     `yaml:"max-entries" json:"max_entries"`
	SimilarityThreshold float64 `yaml:"similarity-threshold" json:"similarity_threshold"`
	StreamingEnabled    bool    `yaml:"streaming-enabled" json:"streaming_enabled"`
	SemanticEnabled     bool    `yaml:"semantic-enabled" json:"semantic_enabled"`
}

// DefaultCacheDefaults returns sensible defaults.
func DefaultCacheDefaults() CacheDefaults {
	return CacheDefaults{
		Enabled:             true,
		TTLSeconds:          60,
		MaxEntries:          1000,
		SimilarityThreshold: 0.85,
		StreamingEnabled:    true,
		SemanticEnabled:     false, // Semantic caching disabled by default
	}
}

// NewCacheConfigManager creates a new cache configuration manager.
func NewCacheConfigManager(defaults CacheDefaults) *CacheConfigManager {
	return &CacheConfigManager{
		defaultConfig:  defaults,
		modelConfigs:   make(map[string]*ModelCacheConfig),
		patternConfigs: make([]ModelCacheConfig, 0),
		ttlConfig:      NewModelTTLConfig(time.Duration(defaults.TTLSeconds) * time.Second),
	}
}

// LoadModelConfigs loads model-specific cache configurations.
func (m *CacheConfigManager) LoadModelConfigs(configs []ModelCacheConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.modelConfigs = make(map[string]*ModelCacheConfig)
	m.patternConfigs = make([]ModelCacheConfig, 0)

	for i := range configs {
		cfg := &configs[i]
		if strings.ContainsAny(cfg.Model, "*?") {
			m.patternConfigs = append(m.patternConfigs, *cfg)
			m.ttlConfig.SetPatternTTL(cfg.Model, time.Duration(cfg.TTLSeconds)*time.Second)
		} else {
			m.modelConfigs[cfg.Model] = cfg
			m.ttlConfig.SetModelTTL(cfg.Model, time.Duration(cfg.TTLSeconds)*time.Second)
		}
	}
}

// GetModelConfig returns the cache configuration for a specific model.
func (m *CacheConfigManager) GetModelConfig(model string) ModelCacheConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check exact match
	if cfg, exists := m.modelConfigs[model]; exists {
		return *cfg
	}

	// Check patterns
	for _, cfg := range m.patternConfigs {
		if matchPattern(cfg.Model, model) {
			return cfg
		}
	}

	// Return defaults
	enabled := m.defaultConfig.Enabled
	return ModelCacheConfig{
		Model:               model,
		TTLSeconds:          m.defaultConfig.TTLSeconds,
		Enabled:             &enabled,
		MaxEntries:          m.defaultConfig.MaxEntries,
		SimilarityThreshold: m.defaultConfig.SimilarityThreshold,
	}
}

// GetTTL returns the TTL for a specific model.
func (m *CacheConfigManager) GetTTL(model string) time.Duration {
	return m.ttlConfig.GetTTL(model)
}

// IsCachingEnabled checks if caching is enabled for a model.
func (m *CacheConfigManager) IsCachingEnabled(model string) bool {
	cfg := m.GetModelConfig(model)
	if cfg.Enabled != nil {
		return *cfg.Enabled
	}
	return m.defaultConfig.Enabled
}

// Global cache configuration manager
var (
	globalCacheConfigManager     *CacheConfigManager
	globalCacheConfigManagerOnce sync.Once
)

// GetCacheConfigManager returns the global cache configuration manager.
func GetCacheConfigManager() *CacheConfigManager {
	globalCacheConfigManagerOnce.Do(func() {
		globalCacheConfigManager = NewCacheConfigManager(DefaultCacheDefaults())
	})
	return globalCacheConfigManager
}

// InitCacheConfigManager initializes the global cache configuration manager.
func InitCacheConfigManager(defaults CacheDefaults, modelConfigs []ModelCacheConfig) *CacheConfigManager {
	globalCacheConfigManagerOnce.Do(func() {
		globalCacheConfigManager = NewCacheConfigManager(defaults)
		globalCacheConfigManager.LoadModelConfigs(modelConfigs)
	})
	return globalCacheConfigManager
}
