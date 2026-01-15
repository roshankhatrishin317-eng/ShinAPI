// Package context provides context window management for AI models.
// It handles automatic truncation and token budget allocation to prevent context overflow.
package context

import (
	"sync"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Strategy defines the truncation strategy to use.
type Strategy string

const (
	// StrategySlidingWindow keeps the most recent N messages.
	StrategySlidingWindow Strategy = "sliding-window"

	// StrategyPriority keeps important messages based on priority rules.
	StrategyPriority Strategy = "priority"

	// StrategySummarize uses LLM to summarize old context (future feature).
	StrategySummarize Strategy = "summarize"
)

// ContextConfig holds context management configuration.
type ContextConfig struct {
	// Enabled controls whether context management is active
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Strategy is the truncation strategy to use
	Strategy Strategy `yaml:"strategy" json:"strategy"`

	// ModelLimits maps model names to their context limits
	ModelLimits map[string]int64 `yaml:"model-limits" json:"model_limits"`

	// Reserve holds token reservations for various purposes
	Reserve ReserveConfig `yaml:"reserve" json:"reserve"`

	// AlwaysKeep defines what should never be truncated
	AlwaysKeep AlwaysKeepConfig `yaml:"always-keep" json:"always_keep"`
}

// ReserveConfig holds token reservations.
type ReserveConfig struct {
	// Response reserves tokens for the model response
	Response int64 `yaml:"response" json:"response"`

	// Tools reserves tokens for tool definitions
	Tools int64 `yaml:"tools" json:"tools"`

	// System reserves tokens for system prompts
	System int64 `yaml:"system" json:"system"`
}

// AlwaysKeepConfig defines what should never be truncated.
type AlwaysKeepConfig struct {
	// SystemPrompt keeps the system prompt
	SystemPrompt bool `yaml:"system-prompt" json:"system_prompt"`

	// ToolDefinitions keeps tool definitions
	ToolDefinitions bool `yaml:"tool-definitions" json:"tool_definitions"`

	// RecentMessages keeps the N most recent messages
	RecentMessages int `yaml:"recent-messages" json:"recent_messages"`
}

// DefaultContextConfig returns sensible defaults.
func DefaultContextConfig() ContextConfig {
	return ContextConfig{
		Enabled:  false,
		Strategy: StrategyPriority,
		ModelLimits: map[string]int64{
			"gpt-4":            128000,
			"gpt-4-turbo":      128000,
			"gpt-4o":           128000,
			"claude-opus-4-5":  200000,
			"claude-sonnet-4":  200000,
			"gemini-3-pro":     1000000,
			"gemini-2.5-pro":   1000000,
			"gemini-1.5-pro":   2000000,
		},
		Reserve: ReserveConfig{
			Response: 4096,
			Tools:    2048,
			System:   1024,
		},
		AlwaysKeep: AlwaysKeepConfig{
			SystemPrompt:    true,
			ToolDefinitions: true,
			RecentMessages:  10,
		},
	}
}

// Manager provides context window management.
type Manager struct {
	config ContextConfig
	mu     sync.RWMutex
}

// NewManager creates a new context manager.
func NewManager(cfg ContextConfig) *Manager {
	return &Manager{config: cfg}
}

// Global manager instance
var (
	globalManager     *Manager
	globalManagerOnce sync.Once
)

// GetManager returns the global context manager.
func GetManager() *Manager {
	globalManagerOnce.Do(func() {
		globalManager = NewManager(DefaultContextConfig())
	})
	return globalManager
}

// InitManager initializes the global context manager with config.
func InitManager(cfg ContextConfig) *Manager {
	globalManagerOnce.Do(func() {
		globalManager = NewManager(cfg)
	})
	return globalManager
}

// GetModelLimit returns the context limit for a model.
func (m *Manager) GetModelLimit(model string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit, ok := m.config.ModelLimits[model]; ok {
		return limit
	}

	// Try partial matching
	for name, limit := range m.config.ModelLimits {
		if containsSubstring(model, name) {
			return limit
		}
	}

	// Default limit for unknown models
	return 128000
}

// GetAvailableTokens returns tokens available for messages after reservations.
func (m *Manager) GetAvailableTokens(model string) int64 {
	limit := m.GetModelLimit(model)
	reserved := m.config.Reserve.Response + m.config.Reserve.Tools + m.config.Reserve.System
	return limit - reserved
}

// NeedsTruncation checks if messages exceed the model's context limit.
func (m *Manager) NeedsTruncation(messages []byte, model string, tokenCount int64) bool {
	available := m.GetAvailableTokens(model)
	return tokenCount > available
}

// Truncate reduces messages to fit within the model's context limit.
func (m *Manager) Truncate(messages []byte, model string, currentTokens int64) []byte {
	if !m.config.Enabled {
		return messages
	}

	available := m.GetAvailableTokens(model)
	if currentTokens <= available {
		return messages
	}

	switch m.config.Strategy {
	case StrategySlidingWindow:
		return m.truncateSlidingWindow(messages, available)
	case StrategyPriority:
		return m.truncatePriority(messages, available)
	default:
		return m.truncateSlidingWindow(messages, available)
	}
}

// truncateSlidingWindow keeps the most recent messages.
func (m *Manager) truncateSlidingWindow(messages []byte, targetTokens int64) []byte {
	parsed := gjson.ParseBytes(messages)
	if !parsed.IsArray() {
		return messages
	}

	msgArray := parsed.Array()
	if len(msgArray) == 0 {
		return messages
	}

	// Find system message (if any) and keep it
	var systemMsg gjson.Result
	hasSystem := false
	startIdx := 0

	if msgArray[0].Get("role").String() == "system" {
		systemMsg = msgArray[0]
		hasSystem = true
		startIdx = 1
	}

	// Keep minimum recent messages
	keepRecent := m.config.AlwaysKeep.RecentMessages
	if keepRecent <= 0 {
		keepRecent = 5
	}

	// Calculate how many messages to keep
	// Start by keeping just recent messages and add more if space permits
	totalMsgs := len(msgArray) - startIdx
	if totalMsgs <= keepRecent {
		return messages // Already within limits
	}

	// Build new messages array with system + recent
	result := []byte("[]")

	if hasSystem && m.config.AlwaysKeep.SystemPrompt {
		result, _ = sjson.SetRawBytes(result, "-1", []byte(systemMsg.Raw))
	}

	// Add recent messages
	recentStart := len(msgArray) - keepRecent
	if recentStart < startIdx {
		recentStart = startIdx
	}

	for i := recentStart; i < len(msgArray); i++ {
		result, _ = sjson.SetRawBytes(result, "-1", []byte(msgArray[i].Raw))
	}

	return result
}

// truncatePriority keeps messages based on priority rules.
func (m *Manager) truncatePriority(messages []byte, targetTokens int64) []byte {
	parsed := gjson.ParseBytes(messages)
	if !parsed.IsArray() {
		return messages
	}

	msgArray := parsed.Array()
	if len(msgArray) == 0 {
		return messages
	}

	// Priority order:
	// 1. System prompt (highest)
	// 2. Tool definitions
	// 3. Recent messages (last N)
	// 4. Tool calls and results
	// 5. Old assistant messages (lowest)

	result := []byte("[]")
	keepRecent := m.config.AlwaysKeep.RecentMessages
	if keepRecent <= 0 {
		keepRecent = 10
	}

	// Always keep system message
	startIdx := 0
	if len(msgArray) > 0 && msgArray[0].Get("role").String() == "system" {
		if m.config.AlwaysKeep.SystemPrompt {
			result, _ = sjson.SetRawBytes(result, "-1", []byte(msgArray[0].Raw))
		}
		startIdx = 1
	}

	// Calculate recent message range
	recentStart := len(msgArray) - keepRecent
	if recentStart < startIdx {
		recentStart = startIdx
	}

	// Add recent messages (always keep these)
	for i := recentStart; i < len(msgArray); i++ {
		result, _ = sjson.SetRawBytes(result, "-1", []byte(msgArray[i].Raw))
	}

	return result
}

// Helper function for substring matching
func containsSubstring(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
