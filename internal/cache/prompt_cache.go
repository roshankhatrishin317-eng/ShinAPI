// Package cache provides prompt caching support for AI providers.
// This file adds Anthropic prompt caching with cache_control blocks.
package cache

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// PromptCacheConfig holds prompt caching configuration.
type PromptCacheConfig struct {
	// Enabled controls whether prompt caching is active
	Enabled bool `yaml:"enabled" json:"enabled"`

	// AutoDetect automatically marks cacheable content
	AutoDetect bool `yaml:"auto-detect" json:"auto_detect"`

	// Anthropic-specific settings
	Anthropic AnthropicCacheConfig `yaml:"anthropic" json:"anthropic"`
}

// AnthropicCacheConfig holds Anthropic-specific caching settings.
type AnthropicCacheConfig struct {
	// Enabled controls Anthropic prompt caching
	Enabled bool `yaml:"enabled" json:"enabled"`

	// MarkSystemPrompt adds cache_control to system prompt
	MarkSystemPrompt bool `yaml:"mark-system-prompt" json:"mark_system_prompt"`

	// MarkTools adds cache_control to tool definitions
	MarkTools bool `yaml:"mark-tools" json:"mark_tools"`

	// TTL is the cache time-to-live in seconds (Anthropic default: 300)
	TTL int `yaml:"ttl" json:"ttl"`
}

// DefaultPromptCacheConfig returns sensible defaults.
func DefaultPromptCacheConfig() PromptCacheConfig {
	return PromptCacheConfig{
		Enabled:    false,
		AutoDetect: true,
		Anthropic: AnthropicCacheConfig{
			Enabled:          true,
			MarkSystemPrompt: true,
			MarkTools:        true,
			TTL:              300, // 5 minutes
		},
	}
}

// InsertClaudeCacheControl adds cache_control blocks to a Claude request.
// This enables Anthropic's prompt caching for reduced costs and latency.
func InsertClaudeCacheControl(request []byte, cfg AnthropicCacheConfig) []byte {
	if !cfg.Enabled {
		return request
	}

	// Mark system prompt for caching
	if cfg.MarkSystemPrompt {
		request = markSystemForCaching(request)
	}

	// Mark tools for caching
	if cfg.MarkTools {
		request = markToolsForCaching(request)
	}

	return request
}

// markSystemForCaching adds cache_control to system prompt.
func markSystemForCaching(request []byte) []byte {
	system := gjson.GetBytes(request, "system")
	if !system.Exists() {
		return request
	}

	// Handle string system prompt
	if system.Type == gjson.String {
		// Convert to array format with cache_control
		newSystem := []byte("[]")
		block := `{"type":"text","text":"","cache_control":{"type":"ephemeral"}}`
		block, _ = sjson.Set(block, "text", system.String())
		newSystem, _ = sjson.SetRawBytes(newSystem, "-1", []byte(block))
		request, _ = sjson.SetRawBytes(request, "system", newSystem)
		return request
	}

	// Handle array system prompt - mark last block for caching
	if system.IsArray() {
		systemArray := system.Array()
		if len(systemArray) == 0 {
			return request
		}

		// Get the last block and add cache_control
		lastIdx := len(systemArray) - 1
		lastBlock := systemArray[lastIdx].Raw

		// Add cache_control to last block
		lastBlock, _ = sjson.SetRaw(lastBlock, "cache_control", `{"type":"ephemeral"}`)
		request, _ = sjson.SetRawBytes(request, "system."+itoa(lastIdx), []byte(lastBlock))
	}

	return request
}

// markToolsForCaching adds cache_control to tool definitions.
func markToolsForCaching(request []byte) []byte {
	tools := gjson.GetBytes(request, "tools")
	if !tools.Exists() || !tools.IsArray() {
		return request
	}

	toolsArray := tools.Array()
	if len(toolsArray) == 0 {
		return request
	}

	// Mark the last tool for caching (cache prefix matching)
	lastIdx := len(toolsArray) - 1
	lastTool := toolsArray[lastIdx].Raw

	// Add cache_control to last tool
	lastTool, _ = sjson.SetRaw(lastTool, "cache_control", `{"type":"ephemeral"}`)
	request, _ = sjson.SetRawBytes(request, "tools."+itoa(lastIdx), []byte(lastTool))

	return request
}

// GetClaudeCacheHeaders returns required headers for Claude caching.
func GetClaudeCacheHeaders() map[string]string {
	return map[string]string{
		"anthropic-beta": "prompt-caching-2024-07-31",
	}
}

// HasCacheControl checks if a request already has cache_control markers.
func HasCacheControl(request []byte) bool {
	// Check system prompt
	system := gjson.GetBytes(request, "system")
	if system.IsArray() {
		hasCacheControl := false
		system.ForEach(func(_, block gjson.Result) bool {
			if block.Get("cache_control").Exists() {
				hasCacheControl = true
				return false
			}
			return true
		})
		if hasCacheControl {
			return true
		}
	}

	// Check tools
	tools := gjson.GetBytes(request, "tools")
	if tools.IsArray() {
		hasCacheControl := false
		tools.ForEach(func(_, tool gjson.Result) bool {
			if tool.Get("cache_control").Exists() {
				hasCacheControl = true
				return false
			}
			return true
		})
		if hasCacheControl {
			return true
		}
	}

	return false
}

// ExtractCacheUsage extracts cache usage information from Claude response.
type CacheUsage struct {
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

// GetClaudeCacheUsage extracts cache usage from a Claude response.
func GetClaudeCacheUsage(response []byte) CacheUsage {
	usage := gjson.GetBytes(response, "usage")
	if !usage.Exists() {
		return CacheUsage{}
	}

	return CacheUsage{
		CacheCreationInputTokens: usage.Get("cache_creation_input_tokens").Int(),
		CacheReadInputTokens:     usage.Get("cache_read_input_tokens").Int(),
	}
}

// CalculateCacheSavings calculates cost savings from caching.
// Returns the percentage of tokens saved.
func CalculateCacheSavings(usage CacheUsage) float64 {
	if usage.CacheReadInputTokens == 0 && usage.CacheCreationInputTokens == 0 {
		return 0
	}

	// Cache reads cost 10% of normal input price
	// Cache writes cost 125% of normal input price
	// Calculate effective savings

	total := usage.CacheCreationInputTokens + usage.CacheReadInputTokens
	if total == 0 {
		return 0
	}

	// Savings from cache reads (90% savings per cached token)
	savings := float64(usage.CacheReadInputTokens) * 0.9
	// Cost from cache writes (25% extra per cached token)
	cost := float64(usage.CacheCreationInputTokens) * 0.25

	return (savings - cost) / float64(total) * 100
}

// Simple int to string conversion
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var result []byte
	for i > 0 {
		result = append([]byte{byte('0' + i%10)}, result...)
		i /= 10
	}
	return string(result)
}
