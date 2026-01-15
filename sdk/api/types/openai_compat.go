// Package types provides type references from external SDKs for validation and documentation.
// These types are used for schema validation only; actual request handling uses gjson/sjson.
package types

import (
	"encoding/json"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAI SDK type aliases for reference and validation.
// NOTE: Actual request handling in this codebase uses gjson/sjson for JSON manipulation.
// These types are provided for:
// - Schema validation of incoming requests
// - Type-safe tool definition creation
// - Documentation and IDE support
type (
	// ChatCompletionRequest represents an OpenAI chat completion request.
	ChatCompletionRequest = openai.ChatCompletionRequest

	// ChatCompletionResponse represents an OpenAI chat completion response.
	ChatCompletionResponse = openai.ChatCompletionResponse

	// ChatCompletionMessage represents a message in a chat completion request.
	ChatCompletionMessage = openai.ChatCompletionMessage

	// Tool represents an OpenAI tool definition.
	Tool = openai.Tool

	// ToolCall represents a tool call in an assistant message.
	ToolCall = openai.ToolCall

	// FunctionDefinition represents a function definition within a tool.
	FunctionDefinition = openai.FunctionDefinition

	// ChatCompletionChoice represents a choice in a chat completion response.
	ChatCompletionChoice = openai.ChatCompletionChoice

	// Usage represents token usage in a response.
	Usage = openai.Usage
)

// ValidateToolDefinition validates a tool definition JSON against the OpenAI schema.
// Returns nil if the JSON is valid, otherwise returns an unmarshaling error.
func ValidateToolDefinition(toolJSON []byte) error {
	var tool openai.Tool
	return json.Unmarshal(toolJSON, &tool)
}

// ValidateChatCompletionRequest validates a chat completion request JSON.
// Returns nil if the JSON is valid, otherwise returns an unmarshaling error.
func ValidateChatCompletionRequest(reqJSON []byte) error {
	var req openai.ChatCompletionRequest
	return json.Unmarshal(reqJSON, &req)
}

// NewTool creates a new tool definition with the given function.
func NewTool(name, description string, parameters any) Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        name,
			Description: description,
			Parameters:  parameters,
		},
	}
}

// ToolTypeFunction is the type identifier for function tools.
const ToolTypeFunction = openai.ToolTypeFunction
