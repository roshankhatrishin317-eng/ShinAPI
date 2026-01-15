// Package tools provides tool calling format conversion between different AI providers.
package tools

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// convertFromOpenAITools converts OpenAI tool definitions to target format.
// OpenAI format: tools[].{type:"function", function:{name, description, parameters}}
func (tc *ToolConverter) convertFromOpenAITools(tools []byte, to string) []byte {
	switch to {
	case ProviderClaude:
		return tc.openAIToolsToClaude(tools)
	case ProviderGemini:
		return tc.openAIToolsToGemini(tools)
	default:
		return tools
	}
}

// openAIToolsToClaude converts OpenAI tool format to Claude format.
// Claude format: tools[].{name, description, input_schema}
func (tc *ToolConverter) openAIToolsToClaude(tools []byte) []byte {
	parsed := gjson.ParseBytes(tools)
	if !parsed.IsArray() {
		return tools
	}

	result := []byte("[]")
	idx := 0

	parsed.ForEach(func(_, tool gjson.Result) bool {
		if tool.Get("type").String() != "function" {
			return true
		}

		fn := tool.Get("function")
		if !fn.Exists() {
			return true
		}

		claudeTool := `{}`
		claudeTool, _ = sjson.Set(claudeTool, "name", fn.Get("name").String())

		if desc := fn.Get("description"); desc.Exists() {
			claudeTool, _ = sjson.Set(claudeTool, "description", desc.String())
		}

		if params := fn.Get("parameters"); params.Exists() {
			claudeTool, _ = sjson.SetRaw(claudeTool, "input_schema", params.Raw)
		} else {
			claudeTool, _ = sjson.SetRaw(claudeTool, "input_schema", `{"type":"object","properties":{}}`)
		}

		result, _ = sjson.SetRawBytes(result, "-1", []byte(claudeTool))
		idx++
		return true
	})

	return result
}

// openAIToolsToGemini converts OpenAI tool format to Gemini format.
// Gemini format: tools[].functionDeclarations[].{name, description, parameters}
func (tc *ToolConverter) openAIToolsToGemini(tools []byte) []byte {
	parsed := gjson.ParseBytes(tools)
	if !parsed.IsArray() {
		return tools
	}

	declarations := []byte("[]")

	parsed.ForEach(func(_, tool gjson.Result) bool {
		if tool.Get("type").String() != "function" {
			return true
		}

		fn := tool.Get("function")
		if !fn.Exists() {
			return true
		}

		decl := `{}`
		decl, _ = sjson.Set(decl, "name", fn.Get("name").String())

		if desc := fn.Get("description"); desc.Exists() {
			decl, _ = sjson.Set(decl, "description", desc.String())
		}

		if params := fn.Get("parameters"); params.Exists() {
			decl, _ = sjson.SetRaw(decl, "parameters", params.Raw)
		}

		declarations, _ = sjson.SetRawBytes(declarations, "-1", []byte(decl))
		return true
	})

	// Wrap in Gemini tools format
	result := `[{"functionDeclarations":[]}]`
	result, _ = sjson.SetRaw(result, "0.functionDeclarations", string(declarations))

	return []byte(result)
}

// convertFromOpenAIToolCalls converts OpenAI tool call response to target format.
func (tc *ToolConverter) convertFromOpenAIToolCalls(response []byte, to string) []byte {
	switch to {
	case ProviderClaude:
		return tc.openAIToolCallsToClaude(response)
	case ProviderGemini:
		return tc.openAIToolCallsToGemini(response)
	default:
		return response
	}
}

// openAIToolCallsToClaude converts OpenAI tool_calls to Claude tool_use blocks.
func (tc *ToolConverter) openAIToolCallsToClaude(response []byte) []byte {
	toolCalls := gjson.GetBytes(response, "choices.0.message.tool_calls")
	if !toolCalls.Exists() || !toolCalls.IsArray() {
		return response
	}

	content := []byte("[]")

	toolCalls.ForEach(func(_, tc gjson.Result) bool {
		block := `{"type":"tool_use"}`
		block, _ = sjson.Set(block, "id", tc.Get("id").String())
		block, _ = sjson.Set(block, "name", tc.Get("function.name").String())

		args := tc.Get("function.arguments").String()
		if args != "" {
			block, _ = sjson.SetRaw(block, "input", args)
		} else {
			block, _ = sjson.SetRaw(block, "input", "{}")
		}

		content, _ = sjson.SetRawBytes(content, "-1", []byte(block))
		return true
	})

	// Build Claude response
	result := `{"content":[],"stop_reason":"tool_use"}`
	result, _ = sjson.SetRaw(result, "content", string(content))

	return []byte(result)
}

// openAIToolCallsToGemini converts OpenAI tool_calls to Gemini functionCall format.
func (tc *ToolConverter) openAIToolCallsToGemini(response []byte) []byte {
	toolCalls := gjson.GetBytes(response, "choices.0.message.tool_calls")
	if !toolCalls.Exists() || !toolCalls.IsArray() {
		return response
	}

	parts := []byte("[]")

	toolCalls.ForEach(func(_, tc gjson.Result) bool {
		part := `{"functionCall":{}}`
		part, _ = sjson.Set(part, "functionCall.name", tc.Get("function.name").String())

		args := tc.Get("function.arguments").String()
		if args != "" {
			part, _ = sjson.SetRaw(part, "functionCall.args", args)
		} else {
			part, _ = sjson.SetRaw(part, "functionCall.args", "{}")
		}

		parts, _ = sjson.SetRawBytes(parts, "-1", []byte(part))
		return true
	})

	// Build Gemini response structure
	result := `{"candidates":[{"content":{"parts":[]}}]}`
	result, _ = sjson.SetRaw(result, "candidates.0.content.parts", string(parts))

	return []byte(result)
}

// extractOpenAIToolCalls extracts tool calls from OpenAI response format.
func (tc *ToolConverter) extractOpenAIToolCalls(response []byte) []ToolCall {
	toolCalls := gjson.GetBytes(response, "choices.0.message.tool_calls")
	if !toolCalls.Exists() || !toolCalls.IsArray() {
		return nil
	}

	var calls []ToolCall
	toolCalls.ForEach(func(_, tc gjson.Result) bool {
		call := ToolCall{
			ID:        tc.Get("id").String(),
			Name:      tc.Get("function.name").String(),
			Arguments: tc.Get("function.arguments").String(),
			Index:     int(tc.Get("index").Int()),
		}
		calls = append(calls, call)
		return true
	})

	return calls
}

// toolResultsToOpenAI converts tool results to OpenAI message format.
func (tc *ToolConverter) toolResultsToOpenAI(results []ToolResult) []byte {
	messages := []byte("[]")

	for _, result := range results {
		msg := `{"role":"tool"}`
		msg, _ = sjson.Set(msg, "tool_call_id", result.ToolCallID)
		msg, _ = sjson.Set(msg, "content", result.Content)

		messages, _ = sjson.SetRawBytes(messages, "-1", []byte(msg))
	}

	return messages
}

// BuildOpenAIToolCallMessage builds an assistant message with tool calls for OpenAI format.
func (tc *ToolConverter) BuildOpenAIToolCallMessage(toolCalls []ToolCall) []byte {
	msg := `{"role":"assistant","content":null,"tool_calls":[]}`

	for i, call := range toolCalls {
		tc := `{"type":"function","function":{}}`
		tc, _ = sjson.Set(tc, "id", call.ID)
		tc, _ = sjson.Set(tc, "index", i)
		tc, _ = sjson.Set(tc, "function.name", call.Name)
		tc, _ = sjson.Set(tc, "function.arguments", call.Arguments)

		msg, _ = sjson.SetRaw(msg, "tool_calls.-1", tc)
	}

	return []byte(msg)
}
