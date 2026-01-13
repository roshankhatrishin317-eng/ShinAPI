package chat_completions

import (
	"context"
	"testing"

	"github.com/tidwall/gjson"
)

// Helper to simulate streaming chunks through the translator
func processChunks(t *testing.T, chunks [][]byte) []string {
	t.Helper()
	var param any
	var results []string
	for _, chunk := range chunks {
		out := ConvertAntigravityResponseToOpenAI(context.Background(), "", nil, nil, chunk, &param)
		results = append(results, out...)
	}
	return results
}

func getFinishReason(jsonStr string) string {
	return gjson.Get(jsonStr, "choices.0.finish_reason").String()
}

func getNativeFinishReason(jsonStr string) string {
	return gjson.Get(jsonStr, "choices.0.native_finish_reason").String()
}

// TestFinishReasonToolCallsNotOverwritten verifies that when a tool call is seen
// in chunk 1 and STOP comes in chunk 2, the final finish_reason is "tool_calls"
func TestFinishReasonToolCallsNotOverwritten(t *testing.T) {
	// Chunk 1: Contains functionCall
	chunk1 := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{
						"functionCall": {
							"name": "list_files",
							"args": {"path": "/tmp"}
						}
					}]
				}
			}],
			"modelVersion": "gemini-3-pro"
		}
	}`)

	// Chunk 2: Contains finishReason STOP and usageMetadata
	chunk2 := []byte(`{
		"response": {
			"candidates": [{
				"finishReason": "STOP"
			}],
			"usageMetadata": {
				"promptTokenCount": 100,
				"candidatesTokenCount": 50,
				"totalTokenCount": 150
			},
			"modelVersion": "gemini-3-pro"
		}
	}`)

	results := processChunks(t, [][]byte{chunk1, chunk2})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Final chunk should have finish_reason = "tool_calls"
	finalResult := results[len(results)-1]
	finishReason := getFinishReason(finalResult)
	if finishReason != "tool_calls" {
		t.Errorf("expected finish_reason 'tool_calls', got '%s'", finishReason)
	}
}

// TestFinishReasonStopForNormalText verifies normal text responses get "stop"
func TestFinishReasonStopForNormalText(t *testing.T) {
	// Chunk 1: Text content
	chunk1 := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{
						"text": "Hello, world!"
					}]
				}
			}],
			"modelVersion": "gemini-3-pro"
		}
	}`)

	// Chunk 2: finishReason STOP with usage
	chunk2 := []byte(`{
		"response": {
			"candidates": [{
				"finishReason": "STOP"
			}],
			"usageMetadata": {
				"promptTokenCount": 10,
				"candidatesTokenCount": 5,
				"totalTokenCount": 15
			}
		}
	}`)

	results := processChunks(t, [][]byte{chunk1, chunk2})
	finalResult := results[len(results)-1]
	finishReason := getFinishReason(finalResult)
	if finishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got '%s'", finishReason)
	}
}

// TestFinishReasonMaxTokens verifies MAX_TOKENS maps to "max_tokens"
func TestFinishReasonMaxTokens(t *testing.T) {
	chunk := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{
						"text": "This response was cut off..."
					}]
				},
				"finishReason": "MAX_TOKENS"
			}],
			"usageMetadata": {
				"promptTokenCount": 100,
				"candidatesTokenCount": 4096,
				"totalTokenCount": 4196
			}
		}
	}`)

	results := processChunks(t, [][]byte{chunk})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	finishReason := getFinishReason(results[0])
	if finishReason != "max_tokens" {
		t.Errorf("expected finish_reason 'max_tokens', got '%s'", finishReason)
	}

	nativeReason := getNativeFinishReason(results[0])
	if nativeReason != "max_tokens" {
		t.Errorf("expected native_finish_reason 'max_tokens', got '%s'", nativeReason)
	}
}

// TestToolCallTakesPriorityOverMaxTokens verifies tool_calls beats MAX_TOKENS
func TestToolCallTakesPriorityOverMaxTokens(t *testing.T) {
	// Chunk with both tool call and MAX_TOKENS
	chunk := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{
						"functionCall": {
							"name": "search",
							"args": {"query": "test"}
						}
					}]
				},
				"finishReason": "MAX_TOKENS"
			}],
			"usageMetadata": {
				"promptTokenCount": 100,
				"candidatesTokenCount": 4096,
				"totalTokenCount": 4196
			}
		}
	}`)

	results := processChunks(t, [][]byte{chunk})
	finishReason := getFinishReason(results[0])
	if finishReason != "tool_calls" {
		t.Errorf("expected finish_reason 'tool_calls' (priority over max_tokens), got '%s'", finishReason)
	}
}

// TestNoFinishReasonOnIntermediateChunks verifies intermediate chunks have null finish_reason
func TestNoFinishReasonOnIntermediateChunks(t *testing.T) {
	// Intermediate chunk with only text, no finishReason or usageMetadata
	chunk := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{
						"text": "Processing..."
					}]
				}
			}],
			"modelVersion": "gemini-3-pro"
		}
	}`)

	results := processChunks(t, [][]byte{chunk})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// finish_reason should be null (empty string when parsed)
	finishReason := getFinishReason(results[0])
	if finishReason != "" {
		t.Errorf("expected null/empty finish_reason on intermediate chunk, got '%s'", finishReason)
	}
}

// TestFinishReasonWithSplitChunks verifies handling when finishReason and usageMetadata come separately
func TestFinishReasonWithSplitChunks(t *testing.T) {
	// Chunk 1: Tool call
	chunk1 := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{
						"functionCall": {
							"name": "get_weather",
							"args": {"city": "NYC"}
						}
					}]
				}
			}]
		}
	}`)

	// Chunk 2: finishReason only (no usageMetadata)
	chunk2 := []byte(`{
		"response": {
			"candidates": [{
				"finishReason": "STOP"
			}]
		}
	}`)

	// Chunk 3: usageMetadata only (no finishReason in this chunk, but cached from chunk2)
	chunk3 := []byte(`{
		"response": {
			"usageMetadata": {
				"promptTokenCount": 50,
				"candidatesTokenCount": 25,
				"totalTokenCount": 75
			}
		}
	}`)

	results := processChunks(t, [][]byte{chunk1, chunk2, chunk3})
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Chunk 2 should have tool_calls since it has finishReason and we saw tool call
	chunk2Result := results[1]
	chunk2FinishReason := getFinishReason(chunk2Result)
	if chunk2FinishReason != "tool_calls" {
		t.Errorf("chunk2: expected finish_reason 'tool_calls', got '%s'", chunk2FinishReason)
	}

	// Chunk 3 (final) should also have tool_calls
	finalResult := results[2]
	finalFinishReason := getFinishReason(finalResult)
	if finalFinishReason != "tool_calls" {
		t.Errorf("final chunk: expected finish_reason 'tool_calls', got '%s'", finalFinishReason)
	}
}

// TestMultipleToolCallsAcrossChunks verifies multiple tool calls are tracked
func TestMultipleToolCallsAcrossChunks(t *testing.T) {
	// Chunk 1: First tool call
	chunk1 := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{
						"functionCall": {
							"name": "read_file",
							"args": {"path": "/etc/hosts"}
						}
					}]
				}
			}]
		}
	}`)

	// Chunk 2: Second tool call
	chunk2 := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{
						"functionCall": {
							"name": "write_file",
							"args": {"path": "/tmp/out.txt", "content": "test"}
						}
					}]
				}
			}]
		}
	}`)

	// Chunk 3: Final with STOP
	chunk3 := []byte(`{
		"response": {
			"candidates": [{
				"finishReason": "STOP"
			}],
			"usageMetadata": {
				"promptTokenCount": 200,
				"candidatesTokenCount": 100,
				"totalTokenCount": 300
			}
		}
	}`)

	results := processChunks(t, [][]byte{chunk1, chunk2, chunk3})
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Final chunk should have tool_calls
	finalResult := results[2]
	finishReason := getFinishReason(finalResult)
	if finishReason != "tool_calls" {
		t.Errorf("expected finish_reason 'tool_calls', got '%s'", finishReason)
	}

	// Verify tool_calls array exists in first two chunks
	for i := 0; i < 2; i++ {
		toolCalls := gjson.Get(results[i], "choices.0.delta.tool_calls")
		if !toolCalls.Exists() || !toolCalls.IsArray() {
			t.Errorf("chunk %d: expected tool_calls array", i+1)
		}
	}
}

// TestDoneMarkerReturnsEmpty verifies [DONE] returns empty slice
func TestDoneMarkerReturnsEmpty(t *testing.T) {
	var param any
	results := ConvertAntigravityResponseToOpenAI(context.Background(), "", nil, nil, []byte("[DONE]"), &param)
	if len(results) != 0 {
		t.Errorf("expected empty results for [DONE], got %d", len(results))
	}
}

// TestReasoningContentHandling verifies thought content goes to reasoning_content
func TestReasoningContentHandling(t *testing.T) {
	chunk := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{
						"text": "Let me think about this...",
						"thought": true
					}]
				}
			}]
		}
	}`)

	results := processChunks(t, [][]byte{chunk})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	reasoningContent := gjson.Get(results[0], "choices.0.delta.reasoning_content").String()
	if reasoningContent != "Let me think about this..." {
		t.Errorf("expected reasoning_content, got '%s'", reasoningContent)
	}

	// Regular content should be null
	content := gjson.Get(results[0], "choices.0.delta.content")
	if content.String() != "" {
		t.Errorf("expected null content for thought, got '%s'", content.String())
	}
}

// TestFinishReasonSafety verifies SAFETY maps to content_filter
func TestFinishReasonSafety(t *testing.T) {
	chunk := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{
						"text": "I cannot help with that..."
					}]
				},
				"finishReason": "SAFETY"
			}],
			"usageMetadata": {
				"promptTokenCount": 100,
				"candidatesTokenCount": 10,
				"totalTokenCount": 110
			}
		}
	}`)

	results := processChunks(t, [][]byte{chunk})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	finishReason := getFinishReason(results[0])
	if finishReason != "content_filter" {
		t.Errorf("expected finish_reason 'content_filter' for SAFETY, got '%s'", finishReason)
	}

	nativeReason := getNativeFinishReason(results[0])
	if nativeReason != "safety" {
		t.Errorf("expected native_finish_reason 'safety', got '%s'", nativeReason)
	}
}

// TestFinishReasonRecitation verifies RECITATION maps to content_filter
func TestFinishReasonRecitation(t *testing.T) {
	chunk := []byte(`{
		"response": {
			"candidates": [{
				"content": {
					"parts": [{
						"text": "Content blocked..."
					}]
				},
				"finishReason": "RECITATION"
			}],
			"usageMetadata": {
				"promptTokenCount": 50,
				"candidatesTokenCount": 5,
				"totalTokenCount": 55
			}
		}
	}`)

	results := processChunks(t, [][]byte{chunk})
	finishReason := getFinishReason(results[0])
	if finishReason != "content_filter" {
		t.Errorf("expected finish_reason 'content_filter' for RECITATION, got '%s'", finishReason)
	}
}

// TestMapGeminiFinishReasonToOpenAI verifies all finish reason mappings
func TestMapGeminiFinishReasonToOpenAI(t *testing.T) {
	tests := []struct {
		gemini   string
		expected string
	}{
		{"STOP", "stop"},
		{"MAX_TOKENS", "max_tokens"},
		{"SAFETY", "content_filter"},
		{"RECITATION", "content_filter"},
		{"BLOCKLIST", "content_filter"},
		{"PROHIBITED_CONTENT", "content_filter"},
		{"SPII", "content_filter"},
		{"MALFORMED_FUNCTION_CALL", "stop"},
		{"UNKNOWN_REASON", "stop"},
		{"", "stop"},
	}

	for _, tt := range tests {
		t.Run(tt.gemini, func(t *testing.T) {
			result := mapGeminiFinishReasonToOpenAI(tt.gemini)
			if result != tt.expected {
				t.Errorf("mapGeminiFinishReasonToOpenAI(%q) = %q, want %q", tt.gemini, result, tt.expected)
			}
		})
	}
}
