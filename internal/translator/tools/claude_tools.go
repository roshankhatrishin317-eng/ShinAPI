// Package tools provides tool calling format conversion between different AI providers.
package tools

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// convertFromClaudeTools converts Claude tool definitions to target format.
// Claude format: tools[].{name, description, input_schema}
func (tc *ToolConverter) convertFromClaudeTools(tools []byte, to string) []byte {
	switch to {
	case ProviderOpenAI:
		return tc.claudeToolsToOpenAI(tools)
	case ProviderGemini:
		return tc.claudeToolsToGemini(tools)
	default:
		return tools
	}
}

// claudeToolsToOpenAI converts Claude tool format to OpenAI format.
// OpenAI format: tools[].{type:"function", function:{name, description, parameters}}
func (tc *ToolConverter) claudeToolsToOpenAI(tools []byte) []byte {
	parsed := gjson.ParseBytes(tools)
	if !parsed.IsArray() {
		return tools
	}

	result := []byte("[]")

	parsed.ForEach(func(_, tool gjson.Result) bool {
		openAITool := `{"type":"function","function":{}}`
		openAITool, _ = sjson.Set(openAITool, "function.name", tool.Get("name").String())

		if desc := tool.Get("description"); desc.Exists() {
			openAITool, _ = sjson.Set(openAITool, "function.description", desc.String())
		}

		if schema := tool.Get("input_schema"); schema.Exists() {
			openAITool, _ = sjson.SetRaw(openAITool, "function.parameters", schema.Raw)
		} else {
			openAITool, _ = sjson.SetRaw(openAITool, "function.parameters", `{"type":"object","properties":{}}`)
		}

		result, _ = sjson.SetRawBytes(result, "-1", []byte(openAITool))
		return true
	})

	return result
}

// claudeToolsToGemini converts Claude tool format to Gemini format.
func (tc *ToolConverter) claudeToolsToGemini(tools []byte) []byte {
	parsed := gjson.ParseBytes(tools)
	if !parsed.IsArray() {
		return tools
	}

	declarations := []byte("[]")

	parsed.ForEach(func(_, tool gjson.Result) bool {
		decl := `{}`
		decl, _ = sjson.Set(decl, "name", tool.Get("name").String())

		if desc := tool.Get("description"); desc.Exists() {
			decl, _ = sjson.Set(decl, "description", desc.String())
		}

		if schema := tool.Get("input_schema"); schema.Exists() {
			decl, _ = sjson.SetRaw(decl, "parameters", schema.Raw)
		}

		declarations, _ = sjson.SetRawBytes(declarations, "-1", []byte(decl))
		return true
	})

	result := `[{"functionDeclarations":[]}]`
	result, _ = sjson.SetRaw(result, "0.functionDeclarations", string(declarations))

	return []byte(result)
}

// convertFromClaudeToolCalls converts Claude tool_use response to target format.
func (tc *ToolConverter) convertFromClaudeToolCalls(response []byte, to string) []byte {
	switch to {
	case ProviderOpenAI:
		return tc.claudeToolCallsToOpenAI(response)
	case ProviderGemini:
		return tc.claudeToolCallsToGemini(response)
	default:
		return response
	}
}

// claudeToolCallsToOpenAI converts Claude tool_use blocks to OpenAI tool_calls.
func (tc *ToolConverter) claudeToolCallsToOpenAI(response []byte) []byte {
	content := gjson.GetBytes(response, "content")
	if !content.Exists() || !content.IsArray() {
		return response
	}

	toolCalls := []byte("[]")
	idx := 0

	content.ForEach(func(_, block gjson.Result) bool {
		if block.Get("type").String() != "tool_use" {
			return true
		}

		tc := `{"type":"function","function":{}}`
		tc, _ = sjson.Set(tc, "id", block.Get("id").String())
		tc, _ = sjson.Set(tc, "index", idx)
		tc, _ = sjson.Set(tc, "function.name", block.Get("name").String())

		if input := block.Get("input"); input.Exists() {
			tc, _ = sjson.Set(tc, "function.arguments", input.Raw)
		} else {
			tc, _ = sjson.Set(tc, "function.arguments", "{}")
		}

		toolCalls, _ = sjson.SetRawBytes(toolCalls, "-1", []byte(tc))
		idx++
		return true
	})

	// Extract any text content for the message
	textContent := ""
	content.ForEach(func(_, block gjson.Result) bool {
		if block.Get("type").String() == "text" {
			textContent += block.Get("text").String()
		}
		return true
	})

	// Build OpenAI response
	result := `{"choices":[{"index":0,"message":{"role":"assistant"},"finish_reason":"tool_calls"}]}`

	if textContent != "" {
		result, _ = sjson.Set(result, "choices.0.message.content", textContent)
	} else {
		result, _ = sjson.Set(result, "choices.0.message.content", nil)
	}

	result, _ = sjson.SetRaw(result, "choices.0.message.tool_calls", string(toolCalls))

	// Copy usage if present
	if usage := gjson.GetBytes(response, "usage"); usage.Exists() {
		result, _ = sjson.SetRaw(result, "usage", usage.Raw)
	}

	return []byte(result)
}

// claudeToolCallsToGemini converts Claude tool_use blocks to Gemini functionCall format.
func (tc *ToolConverter) claudeToolCallsToGemini(response []byte) []byte {
	content := gjson.GetBytes(response, "content")
	if !content.Exists() || !content.IsArray() {
		return response
	}

	parts := []byte("[]")

	content.ForEach(func(_, block gjson.Result) bool {
		if block.Get("type").String() != "tool_use" {
			return true
		}

		part := `{"functionCall":{}}`
		part, _ = sjson.Set(part, "functionCall.name", block.Get("name").String())

		if input := block.Get("input"); input.Exists() {
			part, _ = sjson.SetRaw(part, "functionCall.args", input.Raw)
		} else {
			part, _ = sjson.SetRaw(part, "functionCall.args", "{}")
		}

		parts, _ = sjson.SetRawBytes(parts, "-1", []byte(part))
		return true
	})

	result := `{"candidates":[{"content":{"parts":[]}}]}`
	result, _ = sjson.SetRaw(result, "candidates.0.content.parts", string(parts))

	return []byte(result)
}

// extractClaudeToolCalls extracts tool calls from Claude response format.
func (tc *ToolConverter) extractClaudeToolCalls(response []byte) []ToolCall {
	content := gjson.GetBytes(response, "content")
	if !content.Exists() || !content.IsArray() {
		return nil
	}

	var calls []ToolCall
	idx := 0

	content.ForEach(func(_, block gjson.Result) bool {
		if block.Get("type").String() != "tool_use" {
			return true
		}

		call := ToolCall{
			ID:    block.Get("id").String(),
			Name:  block.Get("name").String(),
			Index: idx,
		}

		if input := block.Get("input"); input.Exists() {
			call.Arguments = input.Raw
		} else {
			call.Arguments = "{}"
		}

		calls = append(calls, call)
		idx++
		return true
	})

	return calls
}

// toolResultsToClaudeFormat converts tool results to Claude tool_result format.
func (tc *ToolConverter) toolResultsToClaudeFormat(results []ToolResult) []byte {
	content := []byte("[]")

	for _, result := range results {
		block := `{"type":"tool_result"}`
		block, _ = sjson.Set(block, "tool_use_id", result.ToolCallID)
		block, _ = sjson.Set(block, "content", result.Content)

		if result.IsError {
			block, _ = sjson.Set(block, "is_error", true)
		}

		content, _ = sjson.SetRawBytes(content, "-1", []byte(block))
	}

	// Return as user message with tool results
	msg := `{"role":"user","content":[]}`
	msg, _ = sjson.SetRaw(msg, "content", string(content))

	return []byte(msg)
}

// BuildClaudeToolUseMessage builds an assistant message with tool_use blocks for Claude format.
func (tc *ToolConverter) BuildClaudeToolUseMessage(toolCalls []ToolCall) []byte {
	content := []byte("[]")

	for _, call := range toolCalls {
		block := `{"type":"tool_use"}`
		block, _ = sjson.Set(block, "id", call.ID)
		block, _ = sjson.Set(block, "name", call.Name)

		if call.Arguments != "" {
			block, _ = sjson.SetRaw(block, "input", call.Arguments)
		} else {
			block, _ = sjson.SetRaw(block, "input", "{}")
		}

		content, _ = sjson.SetRawBytes(content, "-1", []byte(block))
	}

	msg := `{"role":"assistant","content":[]}`
	msg, _ = sjson.SetRaw(msg, "content", string(content))

	return []byte(msg)
}
