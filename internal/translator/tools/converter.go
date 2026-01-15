// Package tools provides tool calling format conversion between different AI providers.
// It handles translation of tool definitions and tool call responses between
// OpenAI, Anthropic Claude, and Google Gemini formats.
package tools

import (
	"encoding/json"
	"sync"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ToolConverter provides bidirectional conversion of tool formats between providers.
type ToolConverter struct {
	mu sync.RWMutex
}

// NewToolConverter creates a new tool format converter.
func NewToolConverter() *ToolConverter {
	return &ToolConverter{}
}

// Global converter instance
var (
	globalConverter     *ToolConverter
	globalConverterOnce sync.Once
)

// GetToolConverter returns the global tool converter instance.
func GetToolConverter() *ToolConverter {
	globalConverterOnce.Do(func() {
		globalConverter = NewToolConverter()
	})
	return globalConverter
}

// ToolDefinition represents a normalized tool definition.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	InputSchema json.RawMessage `json:"input_schema,omitempty"`
}

// ToolCall represents a normalized tool call from a model response.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Index     int    `json:"index,omitempty"`
}

// ToolResult represents the result of executing a tool.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error,omitempty"`
}

// Provider constants
const (
	ProviderOpenAI    = "openai"
	ProviderClaude    = "claude"
	ProviderGemini    = "gemini"
	ProviderAntigravity = "antigravity"
)

// ConvertToolDefinitions converts tool definitions from source format to target format.
func (tc *ToolConverter) ConvertToolDefinitions(tools []byte, from, to string) []byte {
	if from == to {
		return tools
	}

	// First normalize to internal format, then convert to target
	switch from {
	case ProviderOpenAI:
		return tc.convertFromOpenAITools(tools, to)
	case ProviderClaude:
		return tc.convertFromClaudeTools(tools, to)
	case ProviderGemini:
		return tc.convertFromGeminiTools(tools, to)
	default:
		return tools
	}
}

// ConvertToolCalls converts tool calls from provider response format to target format.
func (tc *ToolConverter) ConvertToolCalls(response []byte, from, to string) []byte {
	if from == to {
		return response
	}

	switch from {
	case ProviderOpenAI:
		return tc.convertFromOpenAIToolCalls(response, to)
	case ProviderClaude:
		return tc.convertFromClaudeToolCalls(response, to)
	case ProviderGemini:
		return tc.convertFromGeminiToolCalls(response, to)
	default:
		return response
	}
}

// ConvertToolResults converts tool results to the format expected by the target provider.
func (tc *ToolConverter) ConvertToolResults(results []ToolResult, to string) []byte {
	switch to {
	case ProviderOpenAI:
		return tc.toolResultsToOpenAI(results)
	case ProviderClaude:
		return tc.toolResultsToClaudeFormat(results)
	case ProviderGemini:
		return tc.toolResultsToGemini(results)
	default:
		return nil
	}
}

// ExtractToolCalls extracts tool calls from a provider response into normalized format.
func (tc *ToolConverter) ExtractToolCalls(response []byte, provider string) []ToolCall {
	switch provider {
	case ProviderOpenAI:
		return tc.extractOpenAIToolCalls(response)
	case ProviderClaude:
		return tc.extractClaudeToolCalls(response)
	case ProviderGemini:
		return tc.extractGeminiToolCalls(response)
	default:
		return nil
	}
}

// HasToolCalls checks if a response contains tool calls.
func (tc *ToolConverter) HasToolCalls(response []byte, provider string) bool {
	switch provider {
	case ProviderOpenAI:
		return gjson.GetBytes(response, "choices.0.message.tool_calls").Exists()
	case ProviderClaude:
		content := gjson.GetBytes(response, "content")
		if !content.Exists() || !content.IsArray() {
			return false
		}
		for _, block := range content.Array() {
			if block.Get("type").String() == "tool_use" {
				return true
			}
		}
		return false
	case ProviderGemini:
		return gjson.GetBytes(response, "candidates.0.content.parts.0.functionCall").Exists()
	default:
		return false
	}
}

// GetFinishReason returns the finish reason from a provider response.
func (tc *ToolConverter) GetFinishReason(response []byte, provider string) string {
	switch provider {
	case ProviderOpenAI:
		return gjson.GetBytes(response, "choices.0.finish_reason").String()
	case ProviderClaude:
		return gjson.GetBytes(response, "stop_reason").String()
	case ProviderGemini:
		return gjson.GetBytes(response, "candidates.0.finishReason").String()
	default:
		return ""
	}
}

// IsToolCallFinish checks if the finish reason indicates tool calls.
func (tc *ToolConverter) IsToolCallFinish(response []byte, provider string) bool {
	reason := tc.GetFinishReason(response, provider)
	switch provider {
	case ProviderOpenAI:
		return reason == "tool_calls"
	case ProviderClaude:
		return reason == "tool_use"
	case ProviderGemini:
		return reason == "STOP" && tc.HasToolCalls(response, provider)
	default:
		return false
	}
}

// Helper to create empty JSON array
func emptyJSONArray() []byte {
	return []byte("[]")
}

// Helper to set JSON value safely
func setJSON(data []byte, path string, value interface{}) []byte {
	result, err := sjson.SetBytes(data, path, value)
	if err != nil {
		return data
	}
	return result
}

// Helper to set raw JSON value safely
func setJSONRaw(data []byte, path string, value string) []byte {
	result, err := sjson.SetRawBytes(data, path, []byte(value))
	if err != nil {
		return data
	}
	return result
}


