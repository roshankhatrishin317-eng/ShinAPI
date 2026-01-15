// Package tools provides tool calling format conversion between different AI providers.
package tools

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// convertFromGeminiTools converts Gemini tool definitions to target format.
// Gemini format: tools[].functionDeclarations[].{name, description, parameters}
func (tc *ToolConverter) convertFromGeminiTools(tools []byte, to string) []byte {
	switch to {
	case ProviderOpenAI:
		return tc.geminiToolsToOpenAI(tools)
	case ProviderClaude:
		return tc.geminiToolsToClaude(tools)
	default:
		return tools
	}
}

// geminiToolsToOpenAI converts Gemini function declarations to OpenAI format.
func (tc *ToolConverter) geminiToolsToOpenAI(tools []byte) []byte {
	parsed := gjson.ParseBytes(tools)
	if !parsed.IsArray() {
		return tools
	}

	result := []byte("[]")

	// Gemini tools are nested: tools[].functionDeclarations[]
	parsed.ForEach(func(_, toolGroup gjson.Result) bool {
		declarations := toolGroup.Get("functionDeclarations")
		if !declarations.Exists() || !declarations.IsArray() {
			return true
		}

		declarations.ForEach(func(_, fn gjson.Result) bool {
			openAITool := `{"type":"function","function":{}}`
			openAITool, _ = sjson.Set(openAITool, "function.name", fn.Get("name").String())

			if desc := fn.Get("description"); desc.Exists() {
				openAITool, _ = sjson.Set(openAITool, "function.description", desc.String())
			}

			if params := fn.Get("parameters"); params.Exists() {
				openAITool, _ = sjson.SetRaw(openAITool, "function.parameters", params.Raw)
			} else {
				openAITool, _ = sjson.SetRaw(openAITool, "function.parameters", `{"type":"object","properties":{}}`)
			}

			result, _ = sjson.SetRawBytes(result, "-1", []byte(openAITool))
			return true
		})
		return true
	})

	return result
}

// geminiToolsToClaude converts Gemini function declarations to Claude format.
func (tc *ToolConverter) geminiToolsToClaude(tools []byte) []byte {
	parsed := gjson.ParseBytes(tools)
	if !parsed.IsArray() {
		return tools
	}

	result := []byte("[]")

	parsed.ForEach(func(_, toolGroup gjson.Result) bool {
		declarations := toolGroup.Get("functionDeclarations")
		if !declarations.Exists() || !declarations.IsArray() {
			return true
		}

		declarations.ForEach(func(_, fn gjson.Result) bool {
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
			return true
		})
		return true
	})

	return result
}

// convertFromGeminiToolCalls converts Gemini functionCall response to target format.
func (tc *ToolConverter) convertFromGeminiToolCalls(response []byte, to string) []byte {
	switch to {
	case ProviderOpenAI:
		return tc.geminiToolCallsToOpenAI(response)
	case ProviderClaude:
		return tc.geminiToolCallsToClaude(response)
	default:
		return response
	}
}

// geminiToolCallsToOpenAI converts Gemini functionCall to OpenAI tool_calls format.
func (tc *ToolConverter) geminiToolCallsToOpenAI(response []byte) []byte {
	parts := gjson.GetBytes(response, "candidates.0.content.parts")
	if !parts.Exists() || !parts.IsArray() {
		return response
	}

	toolCalls := []byte("[]")
	idx := 0

	parts.ForEach(func(_, part gjson.Result) bool {
		fc := part.Get("functionCall")
		if !fc.Exists() {
			return true
		}

		tc := `{"type":"function","function":{}}`
		tc, _ = sjson.Set(tc, "id", generateToolCallID(fc.Get("name").String(), idx))
		tc, _ = sjson.Set(tc, "index", idx)
		tc, _ = sjson.Set(tc, "function.name", fc.Get("name").String())

		if args := fc.Get("args"); args.Exists() {
			tc, _ = sjson.Set(tc, "function.arguments", args.Raw)
		} else {
			tc, _ = sjson.Set(tc, "function.arguments", "{}")
		}

		toolCalls, _ = sjson.SetRawBytes(toolCalls, "-1", []byte(tc))
		idx++
		return true
	})

	// Build OpenAI response
	result := `{"choices":[{"index":0,"message":{"role":"assistant","content":null},"finish_reason":"tool_calls"}]}`
	result, _ = sjson.SetRaw(result, "choices.0.message.tool_calls", string(toolCalls))

	// Copy usage metadata if present
	if usage := gjson.GetBytes(response, "usageMetadata"); usage.Exists() {
		// Convert Gemini usage format to OpenAI format
		openAIUsage := `{}`
		if pt := usage.Get("promptTokenCount"); pt.Exists() {
			openAIUsage, _ = sjson.Set(openAIUsage, "prompt_tokens", pt.Int())
		}
		if ct := usage.Get("candidatesTokenCount"); ct.Exists() {
			openAIUsage, _ = sjson.Set(openAIUsage, "completion_tokens", ct.Int())
		}
		if tt := usage.Get("totalTokenCount"); tt.Exists() {
			openAIUsage, _ = sjson.Set(openAIUsage, "total_tokens", tt.Int())
		}
		result, _ = sjson.SetRaw(result, "usage", openAIUsage)
	}

	return []byte(result)
}

// geminiToolCallsToClaude converts Gemini functionCall to Claude tool_use format.
func (tc *ToolConverter) geminiToolCallsToClaude(response []byte) []byte {
	parts := gjson.GetBytes(response, "candidates.0.content.parts")
	if !parts.Exists() || !parts.IsArray() {
		return response
	}

	content := []byte("[]")
	idx := 0

	parts.ForEach(func(_, part gjson.Result) bool {
		fc := part.Get("functionCall")
		if !fc.Exists() {
			return true
		}

		block := `{"type":"tool_use"}`
		block, _ = sjson.Set(block, "id", generateToolCallID(fc.Get("name").String(), idx))
		block, _ = sjson.Set(block, "name", fc.Get("name").String())

		if args := fc.Get("args"); args.Exists() {
			block, _ = sjson.SetRaw(block, "input", args.Raw)
		} else {
			block, _ = sjson.SetRaw(block, "input", "{}")
		}

		content, _ = sjson.SetRawBytes(content, "-1", []byte(block))
		idx++
		return true
	})

	result := `{"content":[],"stop_reason":"tool_use"}`
	result, _ = sjson.SetRaw(result, "content", string(content))

	return []byte(result)
}

// extractGeminiToolCalls extracts tool calls from Gemini response format.
func (tc *ToolConverter) extractGeminiToolCalls(response []byte) []ToolCall {
	parts := gjson.GetBytes(response, "candidates.0.content.parts")
	if !parts.Exists() || !parts.IsArray() {
		return nil
	}

	var calls []ToolCall
	idx := 0

	parts.ForEach(func(_, part gjson.Result) bool {
		fc := part.Get("functionCall")
		if !fc.Exists() {
			return true
		}

		call := ToolCall{
			ID:    generateToolCallID(fc.Get("name").String(), idx),
			Name:  fc.Get("name").String(),
			Index: idx,
		}

		if args := fc.Get("args"); args.Exists() {
			call.Arguments = args.Raw
		} else {
			call.Arguments = "{}"
		}

		calls = append(calls, call)
		idx++
		return true
	})

	return calls
}

// toolResultsToGemini converts tool results to Gemini functionResponse format.
func (tc *ToolConverter) toolResultsToGemini(results []ToolResult) []byte {
	parts := []byte("[]")

	for _, result := range results {
		part := `{"functionResponse":{}}`
		part, _ = sjson.Set(part, "functionResponse.name", result.ToolCallID) // Gemini uses name, not ID
		part, _ = sjson.SetRaw(part, "functionResponse.response", `{"result":`+result.Content+`}`)

		parts, _ = sjson.SetRawBytes(parts, "-1", []byte(part))
	}

	// Return as model content with function response
	msg := `{"role":"function","parts":[]}`
	msg, _ = sjson.SetRaw(msg, "parts", string(parts))

	return []byte(msg)
}

// BuildGeminiFunctionCallMessage builds a model message with functionCall for Gemini format.
func (tc *ToolConverter) BuildGeminiFunctionCallMessage(toolCalls []ToolCall) []byte {
	parts := []byte("[]")

	for _, call := range toolCalls {
		part := `{"functionCall":{}}`
		part, _ = sjson.Set(part, "functionCall.name", call.Name)

		if call.Arguments != "" {
			part, _ = sjson.SetRaw(part, "functionCall.args", call.Arguments)
		} else {
			part, _ = sjson.SetRaw(part, "functionCall.args", "{}")
		}

		parts, _ = sjson.SetRawBytes(parts, "-1", []byte(part))
	}

	msg := `{"role":"model","parts":[]}`
	msg, _ = sjson.SetRaw(msg, "parts", string(parts))

	return []byte(msg)
}

// generateToolCallID generates a unique tool call ID for providers that don't provide one.
func generateToolCallID(name string, index int) string {
	return "call_" + name + "_" + string(rune('0'+index))
}
