// Package reasoning provides extended thinking/reasoning support for various AI models.
// It handles extraction and enhancement of reasoning content from Claude, Gemini, and DeepSeek models.
package reasoning

// ReasoningConfig holds configuration for reasoning/thinking features.
type ReasoningConfig struct {
	// Enabled controls whether reasoning features are active
	Enabled bool `yaml:"enabled" json:"enabled"`

	// ExtractThinking controls whether to extract thinking from responses
	ExtractThinking bool `yaml:"extract-thinking" json:"extract_thinking"`

	// ShowThinkingToClient controls whether thinking is returned to client
	ShowThinkingToClient bool `yaml:"show-thinking-to-client" json:"show_thinking_to_client"`

	// Claude-specific settings
	Claude ClaudeReasoningConfig `yaml:"claude" json:"claude"`

	// Gemini-specific settings
	Gemini GeminiReasoningConfig `yaml:"gemini" json:"gemini"`

	// DeepSeek-specific settings
	DeepSeek DeepSeekReasoningConfig `yaml:"deepseek" json:"deepseek"`
}

// ClaudeReasoningConfig holds Claude-specific reasoning settings.
type ClaudeReasoningConfig struct {
	// DefaultEffort is the default effort level (low, medium, high)
	DefaultEffort string `yaml:"default-effort" json:"default_effort"`

	// EnableThinking enables extended thinking mode
	EnableThinking bool `yaml:"enable-thinking" json:"enable_thinking"`

	// BudgetTokens is the default thinking budget in tokens
	BudgetTokens int `yaml:"budget-tokens" json:"budget_tokens"`

	// InterleavedTools enables tool use with thinking (requires beta header)
	InterleavedTools bool `yaml:"interleaved-tools" json:"interleaved_tools"`
}

// GeminiReasoningConfig holds Gemini-specific reasoning settings.
type GeminiReasoningConfig struct {
	// DefaultThinkingLevel is the default thinking level (low, high)
	DefaultThinkingLevel string `yaml:"default-thinking-level" json:"default_thinking_level"`

	// IncludeThoughts includes thought summaries in response
	IncludeThoughts bool `yaml:"include-thoughts" json:"include_thoughts"`

	// PreserveSignatures preserves thought signatures for multi-turn (CRITICAL)
	PreserveSignatures bool `yaml:"preserve-signatures" json:"preserve_signatures"`

	// ForceTemperature1 forces temperature to 1.0 for Gemini 3
	ForceTemperature1 bool `yaml:"force-temperature-1" json:"force_temperature_1"`
}

// DeepSeekReasoningConfig holds DeepSeek-specific reasoning settings.
type DeepSeekReasoningConfig struct {
	// ExtractThinkTags extracts <think>...</think> tags
	ExtractThinkTags bool `yaml:"extract-think-tags" json:"extract_think_tags"`
}

// DefaultReasoningConfig returns the default reasoning configuration.
func DefaultReasoningConfig() ReasoningConfig {
	return ReasoningConfig{
		Enabled:              false,
		ExtractThinking:      true,
		ShowThinkingToClient: false,
		Claude: ClaudeReasoningConfig{
			DefaultEffort:    "high",
			EnableThinking:   true,
			BudgetTokens:     16000,
			InterleavedTools: true,
		},
		Gemini: GeminiReasoningConfig{
			DefaultThinkingLevel: "high",
			IncludeThoughts:      true,
			PreserveSignatures:   true,
			ForceTemperature1:    true,
		},
		DeepSeek: DeepSeekReasoningConfig{
			ExtractThinkTags: true,
		},
	}
}

// ThinkingResult holds extracted thinking content and metadata.
type ThinkingResult struct {
	// Thinking is the extracted thinking/reasoning content
	Thinking string `json:"thinking,omitempty"`

	// ThinkingSummary is a summarized version (for Claude)
	ThinkingSummary string `json:"thinking_summary,omitempty"`

	// Answer is the final answer with thinking removed
	Answer string `json:"answer,omitempty"`

	// ThinkingTokens is the count of thinking tokens used
	ThinkingTokens int64 `json:"thinking_tokens,omitempty"`

	// Redacted indicates if thinking was redacted for safety
	Redacted bool `json:"redacted,omitempty"`

	// Signature is the thinking signature (for Gemini multi-turn)
	Signature string `json:"signature,omitempty"`
}

// TokenUsage holds token usage breakdown including thinking tokens.
type TokenUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	ThinkingTokens   int64 `json:"thinking_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

// ReasoningProvider identifies which provider's reasoning format to use.
type ReasoningProvider string

const (
	ProviderClaude   ReasoningProvider = "claude"
	ProviderGemini   ReasoningProvider = "gemini"
	ProviderDeepSeek ReasoningProvider = "deepseek"
	ProviderOpenAI   ReasoningProvider = "openai"
)

// IsReasoningModel checks if a model name is a known reasoning model.
func IsReasoningModel(model string) bool {
	reasoningModels := map[string]bool{
		// Claude models with thinking
		"claude-opus-4-5":           true,
		"claude-opus-4-5-20251101":  true,
		"claude-sonnet-4":           true,
		"claude-sonnet-4-20250514":  true,
		
		// Gemini models with thinking
		"gemini-3-pro":              true,
		"gemini-3-pro-preview":      true,
		"gemini-2.5-pro":            true,
		"gemini-2.5-flash":          true,
		
		// DeepSeek reasoning models
		"deepseek-r1":               true,
		"deepseek-reasoner":         true,
		
		// OpenAI o-series
		"o1":                        true,
		"o1-preview":                true,
		"o1-mini":                   true,
		"o3":                        true,
		"o3-mini":                   true,
	}

	return reasoningModels[model]
}

// GetReasoningProvider determines the reasoning provider for a model.
func GetReasoningProvider(model string) ReasoningProvider {
	switch {
	case containsAny(model, "claude", "opus", "sonnet", "haiku"):
		return ProviderClaude
	case containsAny(model, "gemini"):
		return ProviderGemini
	case containsAny(model, "deepseek", "r1"):
		return ProviderDeepSeek
	case containsAny(model, "o1", "o3"):
		return ProviderOpenAI
	default:
		return ""
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
