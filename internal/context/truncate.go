// Package context provides context window management for AI models.
package context

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// TruncateOptions controls truncation behavior.
type TruncateOptions struct {
	// KeepSystemPrompt always keeps the system prompt
	KeepSystemPrompt bool

	// KeepRecentMessages keeps the N most recent messages
	KeepRecentMessages int

	// KeepToolCalls keeps tool call/result pairs
	KeepToolCalls bool

	// PreferUserMessages prioritizes user messages over assistant
	PreferUserMessages bool
}

// DefaultTruncateOptions returns sensible defaults.
func DefaultTruncateOptions() TruncateOptions {
	return TruncateOptions{
		KeepSystemPrompt:   true,
		KeepRecentMessages: 10,
		KeepToolCalls:      true,
		PreferUserMessages: true,
	}
}

// TruncateMessages truncates messages to fit within token limit.
func TruncateMessages(messages []byte, maxTokens int64, estimator TokenEstimator, opts TruncateOptions) []byte {
	parsed := gjson.ParseBytes(messages)
	if !parsed.IsArray() {
		return messages
	}

	msgArray := parsed.Array()
	if len(msgArray) == 0 {
		return messages
	}

	// Estimate current token usage
	currentTokens := int64(0)
	if estimator != nil {
		currentTokens = estimator.EstimateTokens(messages)
	} else {
		currentTokens = estimateTokensRough(messages)
	}

	if currentTokens <= maxTokens {
		return messages // No truncation needed
	}

	// Build priority list
	type msgWithPriority struct {
		index    int
		msg      gjson.Result
		priority int // higher = keep
	}

	var prioritized []msgWithPriority
	for i, msg := range msgArray {
		priority := calculateMessagePriority(msg, i, len(msgArray), opts)
		prioritized = append(prioritized, msgWithPriority{
			index:    i,
			msg:      msg,
			priority: priority,
		})
	}

	// Sort by priority (keep original order within same priority)
	// Simple insertion sort to maintain stability
	for i := 1; i < len(prioritized); i++ {
		for j := i; j > 0 && prioritized[j-1].priority < prioritized[j].priority; j-- {
			prioritized[j-1], prioritized[j] = prioritized[j], prioritized[j-1]
		}
	}

	// Keep messages until we exceed token limit
	result := []byte("[]")
	runningTokens := int64(0)
	keptIndices := make(map[int]bool)

	for _, pm := range prioritized {
		msgTokens := int64(0)
		if estimator != nil {
			msgTokens = estimator.EstimateTokens([]byte(pm.msg.Raw))
		} else {
			msgTokens = estimateTokensRough([]byte(pm.msg.Raw))
		}

		if runningTokens+msgTokens <= maxTokens {
			keptIndices[pm.index] = true
			runningTokens += msgTokens
		}
	}

	// Rebuild in original order
	for i, msg := range msgArray {
		if keptIndices[i] {
			result, _ = sjson.SetRawBytes(result, "-1", []byte(msg.Raw))
		}
	}

	return result
}

// calculateMessagePriority assigns priority to a message.
func calculateMessagePriority(msg gjson.Result, index, total int, opts TruncateOptions) int {
	role := msg.Get("role").String()
	priority := 0

	// System prompt gets highest priority
	if role == "system" && opts.KeepSystemPrompt {
		return 1000
	}

	// Recent messages get high priority
	recentThreshold := total - opts.KeepRecentMessages
	if index >= recentThreshold {
		priority += 500
	}

	// Tool calls and results
	if opts.KeepToolCalls {
		if msg.Get("tool_calls").Exists() {
			priority += 200
		}
		if role == "tool" {
			priority += 200
		}
		// Claude tool_use blocks
		content := msg.Get("content")
		if content.IsArray() {
			content.ForEach(func(_, block gjson.Result) bool {
				if block.Get("type").String() == "tool_use" || block.Get("type").String() == "tool_result" {
					priority += 200
					return false
				}
				return true
			})
		}
	}

	// User messages get slight priority boost
	if role == "user" && opts.PreferUserMessages {
		priority += 50
	}

	// Older messages get lower priority
	priority += index // Newer messages get slightly higher index

	return priority
}

// TokenEstimator estimates token count for content.
type TokenEstimator interface {
	EstimateTokens(content []byte) int64
}

// estimateTokensRough provides a rough token estimate without a tokenizer.
// Approximately 4 characters per token for English text.
func estimateTokensRough(content []byte) int64 {
	return int64(len(content) / 4)
}

// TruncateToMessageCount truncates to keep at most N messages.
func TruncateToMessageCount(messages []byte, maxMessages int, keepSystem bool) []byte {
	parsed := gjson.ParseBytes(messages)
	if !parsed.IsArray() {
		return messages
	}

	msgArray := parsed.Array()
	if len(msgArray) <= maxMessages {
		return messages
	}

	result := []byte("[]")
	startIdx := 0

	// Keep system message if present and configured
	if len(msgArray) > 0 && msgArray[0].Get("role").String() == "system" && keepSystem {
		result, _ = sjson.SetRawBytes(result, "-1", []byte(msgArray[0].Raw))
		startIdx = 1
		maxMessages-- // Account for system message
	}

	// Keep the last N messages
	keepStart := len(msgArray) - maxMessages
	if keepStart < startIdx {
		keepStart = startIdx
	}

	for i := keepStart; i < len(msgArray); i++ {
		result, _ = sjson.SetRawBytes(result, "-1", []byte(msgArray[i].Raw))
	}

	return result
}

// ExtractSystemPrompt extracts the system prompt from messages.
func ExtractSystemPrompt(messages []byte) string {
	parsed := gjson.ParseBytes(messages)
	if !parsed.IsArray() {
		return ""
	}

	msgArray := parsed.Array()
	if len(msgArray) == 0 {
		return ""
	}

	first := msgArray[0]
	if first.Get("role").String() == "system" {
		content := first.Get("content")
		if content.Type == gjson.String {
			return content.String()
		}
		// Handle array content (Claude format)
		if content.IsArray() {
			var text string
			content.ForEach(func(_, block gjson.Result) bool {
				if block.Get("type").String() == "text" {
					text += block.Get("text").String()
				}
				return true
			})
			return text
		}
	}

	return ""
}

// CountMessages returns the number of messages in the array.
func CountMessages(messages []byte) int {
	parsed := gjson.ParseBytes(messages)
	if !parsed.IsArray() {
		return 0
	}
	return len(parsed.Array())
}

// SplitConversationTurns splits messages into user/assistant turn pairs.
func SplitConversationTurns(messages []byte) [][]byte {
	parsed := gjson.ParseBytes(messages)
	if !parsed.IsArray() {
		return nil
	}

	var turns [][]byte
	currentTurn := []byte("[]")
	inTurn := false

	parsed.ForEach(func(_, msg gjson.Result) bool {
		role := msg.Get("role").String()

		if role == "system" {
			// System message starts or is its own turn
			if inTurn {
				turns = append(turns, currentTurn)
				currentTurn = []byte("[]")
			}
			currentTurn, _ = sjson.SetRawBytes(currentTurn, "-1", []byte(msg.Raw))
			turns = append(turns, currentTurn)
			currentTurn = []byte("[]")
			inTurn = false
			return true
		}

		if role == "user" {
			if inTurn {
				// End previous turn
				turns = append(turns, currentTurn)
				currentTurn = []byte("[]")
			}
			inTurn = true
		}

		currentTurn, _ = sjson.SetRawBytes(currentTurn, "-1", []byte(msg.Raw))
		return true
	})

	if len(gjson.ParseBytes(currentTurn).Array()) > 0 {
		turns = append(turns, currentTurn)
	}

	return turns
}
