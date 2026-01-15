// Package types provides type references from external SDKs for validation and documentation.
package types

import (
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
)

// Anthropic SDK type aliases for reference and validation.
// NOTE: Actual request handling in this codebase uses gjson/sjson for JSON manipulation.
// These types are provided for:
// - Schema validation of incoming requests
// - Type-safe message creation
// - Documentation and IDE support
type (
	// Message represents an Anthropic message response.
	Message = anthropic.Message

	// MessageParam represents message parameters for a request.
	MessageParam = anthropic.MessageParam

	// TextBlock represents a text content block.
	TextBlock = anthropic.TextBlock

	// ToolUseBlock represents a tool use content block (when the model calls a tool).
	ToolUseBlock = anthropic.ToolUseBlock

	// ToolResultBlockParam represents a tool result content block.
	ToolResultBlockParam = anthropic.ToolResultBlockParam

	// ToolParam represents a tool definition for Anthropic.
	ToolParam = anthropic.ToolParam

	// ContentBlockParamUnion represents a union of content block types.
	ContentBlockParamUnion = anthropic.ContentBlockParamUnion
)

// ValidateAnthropicMessage validates a message JSON against the Anthropic schema.
// Returns nil if the JSON is valid, otherwise returns an unmarshaling error.
func ValidateAnthropicMessage(msgJSON []byte) error {
	var msg anthropic.Message
	return json.Unmarshal(msgJSON, &msg)
}

// ValidateAnthropicToolParam validates a tool parameter JSON.
// Returns nil if the JSON is valid, otherwise returns an unmarshaling error.
func ValidateAnthropicToolParam(toolJSON []byte) error {
	var tool anthropic.ToolParam
	return json.Unmarshal(toolJSON, &tool)
}

// NewTextBlock creates a new text content block.
func NewTextBlock(text string) anthropic.ContentBlockParamUnion {
	return anthropic.NewTextBlock(text)
}

// NewToolResultBlock creates a new tool result content block.
func NewToolResultBlock(toolUseID string, content string, isError bool) anthropic.ContentBlockParamUnion {
	return anthropic.NewToolResultBlock(toolUseID, content, isError)
}
