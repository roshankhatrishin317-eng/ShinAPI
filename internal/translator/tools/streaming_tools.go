// Package tools provides tool calling format conversion between different AI providers.
package tools

import (
	"sync"

	"github.com/tidwall/gjson"
)

// StreamingToolAccumulator accumulates partial tool calls from SSE streaming responses.
// Different providers stream tool calls differently:
// - OpenAI: Sends delta chunks with tool_calls[].function.arguments building up
// - Claude: Sends content_block_start, content_block_delta, content_block_stop events
// - Gemini: Sends complete functionCall objects (not streamed incrementally)
type StreamingToolAccumulator struct {
	mu           sync.Mutex
	provider     string
	partialCalls map[int]*PartialToolCall
	completeCalls []ToolCall
}

// PartialToolCall represents an incomplete tool call being accumulated.
type PartialToolCall struct {
	ID            string
	Index         int
	Name          string
	ArgumentsJSON string
	Complete      bool
}

// NewStreamingToolAccumulator creates a new accumulator for the given provider.
func NewStreamingToolAccumulator(provider string) *StreamingToolAccumulator {
	return &StreamingToolAccumulator{
		provider:     provider,
		partialCalls: make(map[int]*PartialToolCall),
	}
}

// ProcessChunk processes a streaming chunk and returns any newly completed tool calls.
func (sta *StreamingToolAccumulator) ProcessChunk(chunk []byte) []ToolCall {
	sta.mu.Lock()
	defer sta.mu.Unlock()

	switch sta.provider {
	case ProviderOpenAI:
		return sta.processOpenAIChunk(chunk)
	case ProviderClaude:
		return sta.processClaudeChunk(chunk)
	case ProviderGemini:
		return sta.processGeminiChunk(chunk)
	default:
		return nil
	}
}

// GetCompleteCalls returns all accumulated complete tool calls.
func (sta *StreamingToolAccumulator) GetCompleteCalls() []ToolCall {
	sta.mu.Lock()
	defer sta.mu.Unlock()
	return sta.completeCalls
}

// GetPartialCalls returns all partial (incomplete) tool calls.
func (sta *StreamingToolAccumulator) GetPartialCalls() []*PartialToolCall {
	sta.mu.Lock()
	defer sta.mu.Unlock()

	var partials []*PartialToolCall
	for _, pc := range sta.partialCalls {
		partials = append(partials, pc)
	}
	return partials
}

// Reset clears all accumulated state.
func (sta *StreamingToolAccumulator) Reset() {
	sta.mu.Lock()
	defer sta.mu.Unlock()
	sta.partialCalls = make(map[int]*PartialToolCall)
	sta.completeCalls = nil
}

// Finalize marks all partial calls as complete and returns them.
func (sta *StreamingToolAccumulator) Finalize() []ToolCall {
	sta.mu.Lock()
	defer sta.mu.Unlock()

	for _, pc := range sta.partialCalls {
		if !pc.Complete {
			pc.Complete = true
			sta.completeCalls = append(sta.completeCalls, ToolCall{
				ID:        pc.ID,
				Name:      pc.Name,
				Arguments: pc.ArgumentsJSON,
				Index:     pc.Index,
			})
		}
	}

	return sta.completeCalls
}

// processOpenAIChunk processes OpenAI streaming delta format.
// OpenAI sends: choices[].delta.tool_calls[].{index, id, function.name, function.arguments}
// Arguments are sent character by character across multiple chunks.
func (sta *StreamingToolAccumulator) processOpenAIChunk(chunk []byte) []ToolCall {
	delta := gjson.GetBytes(chunk, "choices.0.delta")
	if !delta.Exists() {
		return nil
	}

	toolCalls := delta.Get("tool_calls")
	if !toolCalls.Exists() || !toolCalls.IsArray() {
		return nil
	}

	var completed []ToolCall

	toolCalls.ForEach(func(_, tc gjson.Result) bool {
		index := int(tc.Get("index").Int())

		// Get or create partial call
		partial, exists := sta.partialCalls[index]
		if !exists {
			partial = &PartialToolCall{Index: index}
			sta.partialCalls[index] = partial
		}

		// Update ID if present (usually in first chunk)
		if id := tc.Get("id"); id.Exists() && id.String() != "" {
			partial.ID = id.String()
		}

		// Update function name if present (usually in first chunk)
		if name := tc.Get("function.name"); name.Exists() && name.String() != "" {
			partial.Name = name.String()
		}

		// Append to arguments (incremental)
		if args := tc.Get("function.arguments"); args.Exists() {
			partial.ArgumentsJSON += args.String()
		}

		return true
	})

	// Check finish_reason to determine if tool calls are complete
	finishReason := gjson.GetBytes(chunk, "choices.0.finish_reason")
	if finishReason.Exists() && finishReason.String() == "tool_calls" {
		for _, pc := range sta.partialCalls {
			if !pc.Complete {
				pc.Complete = true
				call := ToolCall{
					ID:        pc.ID,
					Name:      pc.Name,
					Arguments: pc.ArgumentsJSON,
					Index:     pc.Index,
				}
				completed = append(completed, call)
				sta.completeCalls = append(sta.completeCalls, call)
			}
		}
	}

	return completed
}

// processClaudeChunk processes Claude streaming event format.
// Claude sends:
// - content_block_start: {type: "content_block_start", index, content_block: {type: "tool_use", id, name}}
// - content_block_delta: {type: "content_block_delta", index, delta: {type: "input_json_delta", partial_json}}
// - content_block_stop: {type: "content_block_stop", index}
func (sta *StreamingToolAccumulator) processClaudeChunk(chunk []byte) []ToolCall {
	eventType := gjson.GetBytes(chunk, "type").String()

	switch eventType {
	case "content_block_start":
		return sta.processClaudeBlockStart(chunk)
	case "content_block_delta":
		return sta.processClaudeBlockDelta(chunk)
	case "content_block_stop":
		return sta.processClaudeBlockStop(chunk)
	default:
		return nil
	}
}

func (sta *StreamingToolAccumulator) processClaudeBlockStart(chunk []byte) []ToolCall {
	index := int(gjson.GetBytes(chunk, "index").Int())
	block := gjson.GetBytes(chunk, "content_block")

	if block.Get("type").String() != "tool_use" {
		return nil
	}

	sta.partialCalls[index] = &PartialToolCall{
		ID:            block.Get("id").String(),
		Index:         index,
		Name:          block.Get("name").String(),
		ArgumentsJSON: "",
		Complete:      false,
	}

	return nil
}

func (sta *StreamingToolAccumulator) processClaudeBlockDelta(chunk []byte) []ToolCall {
	index := int(gjson.GetBytes(chunk, "index").Int())
	delta := gjson.GetBytes(chunk, "delta")

	if delta.Get("type").String() != "input_json_delta" {
		return nil
	}

	partial, exists := sta.partialCalls[index]
	if !exists {
		return nil
	}

	partial.ArgumentsJSON += delta.Get("partial_json").String()
	return nil
}

func (sta *StreamingToolAccumulator) processClaudeBlockStop(chunk []byte) []ToolCall {
	index := int(gjson.GetBytes(chunk, "index").Int())

	partial, exists := sta.partialCalls[index]
	if !exists || partial.Complete {
		return nil
	}

	// Only complete if this was a tool_use block
	if partial.Name == "" {
		return nil
	}

	partial.Complete = true
	call := ToolCall{
		ID:        partial.ID,
		Name:      partial.Name,
		Arguments: partial.ArgumentsJSON,
		Index:     partial.Index,
	}
	sta.completeCalls = append(sta.completeCalls, call)

	return []ToolCall{call}
}

// processGeminiChunk processes Gemini streaming format.
// Gemini sends complete functionCall objects, not streamed incrementally.
func (sta *StreamingToolAccumulator) processGeminiChunk(chunk []byte) []ToolCall {
	parts := gjson.GetBytes(chunk, "candidates.0.content.parts")
	if !parts.Exists() || !parts.IsArray() {
		return nil
	}

	var completed []ToolCall
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

		completed = append(completed, call)
		sta.completeCalls = append(sta.completeCalls, call)
		idx++
		return true
	})

	return completed
}

// IsStreamingToolCallChunk checks if an SSE chunk contains tool call data.
func IsStreamingToolCallChunk(chunk []byte, provider string) bool {
	switch provider {
	case ProviderOpenAI:
		return gjson.GetBytes(chunk, "choices.0.delta.tool_calls").Exists()
	case ProviderClaude:
		eventType := gjson.GetBytes(chunk, "type").String()
		if eventType == "content_block_start" {
			return gjson.GetBytes(chunk, "content_block.type").String() == "tool_use"
		}
		if eventType == "content_block_delta" {
			return gjson.GetBytes(chunk, "delta.type").String() == "input_json_delta"
		}
		return false
	case ProviderGemini:
		return gjson.GetBytes(chunk, "candidates.0.content.parts.0.functionCall").Exists()
	default:
		return false
	}
}

// GetStreamingFinishReason extracts finish reason from streaming chunk.
func GetStreamingFinishReason(chunk []byte, provider string) string {
	switch provider {
	case ProviderOpenAI:
		return gjson.GetBytes(chunk, "choices.0.finish_reason").String()
	case ProviderClaude:
		if gjson.GetBytes(chunk, "type").String() == "message_delta" {
			return gjson.GetBytes(chunk, "delta.stop_reason").String()
		}
		return ""
	case ProviderGemini:
		return gjson.GetBytes(chunk, "candidates.0.finishReason").String()
	default:
		return ""
	}
}
