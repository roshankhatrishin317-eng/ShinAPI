// Package reasoning provides extended thinking/reasoning support for various AI models.
package reasoning

import (
	"regexp"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Claude thinking response format:
// {
//   "content": [
//     {"type": "thinking", "thinking": "...", "signature": "..."},
//     {"type": "text", "text": "..."}
//   ],
//   "usage": {"input_tokens": N, "output_tokens": N, "thinking_tokens": N}
// }

// Regex patterns for Claude thinking extraction
var (
	claudeThinkingBlockPattern = regexp.MustCompile(`"type"\s*:\s*"thinking"`)
)

// extractClaudeThinking extracts thinking content from Claude response.
func (rp *ReasoningParser) extractClaudeThinking(response []byte) ThinkingResult {
	result := ThinkingResult{}

	content := gjson.GetBytes(response, "content")
	if !content.Exists() || !content.IsArray() {
		return result
	}

	var thinkingParts []string
	var textParts []string

	content.ForEach(func(_, block gjson.Result) bool {
		blockType := block.Get("type").String()

		switch blockType {
		case "thinking":
			thinking := block.Get("thinking").String()
			if thinking != "" {
				thinkingParts = append(thinkingParts, thinking)
			}
			if sig := block.Get("signature"); sig.Exists() {
				result.Signature = sig.String()
			}

		case "redacted_thinking":
			result.Redacted = true

		case "text":
			text := block.Get("text").String()
			if text != "" {
				textParts = append(textParts, text)
			}
		}
		return true
	})

	// Combine thinking parts
	for i, t := range thinkingParts {
		if i > 0 {
			result.Thinking += "\n\n"
		}
		result.Thinking += t
	}

	// Combine text parts as the answer
	for i, t := range textParts {
		if i > 0 {
			result.Answer += "\n"
		}
		result.Answer += t
	}

	// Extract thinking tokens from usage
	if usage := gjson.GetBytes(response, "usage"); usage.Exists() {
		result.ThinkingTokens = usage.Get("thinking_tokens").Int()
	}

	return result
}

// enhanceClaudeRequest adds thinking parameters to Claude request.
func (rp *ReasoningParser) enhanceClaudeRequest(request []byte) []byte {
	if !rp.config.Claude.EnableThinking {
		return request
	}

	// Add thinking configuration
	request, _ = sjson.SetBytes(request, "thinking.type", "enabled")

	if rp.config.Claude.BudgetTokens > 0 {
		request, _ = sjson.SetBytes(request, "thinking.budget_tokens", rp.config.Claude.BudgetTokens)
	}

	// Add effort parameter if specified
	if rp.config.Claude.DefaultEffort != "" {
		request, _ = sjson.SetBytes(request, "effort", rp.config.Claude.DefaultEffort)
	}

	return request
}

// EnhanceClaudeRequestWithParams adds specific thinking parameters to Claude request.
func EnhanceClaudeRequestWithParams(request []byte, enableThinking bool, budgetTokens int, effort string) []byte {
	if !enableThinking {
		request, _ = sjson.SetBytes(request, "thinking.type", "disabled")
		return request
	}

	request, _ = sjson.SetBytes(request, "thinking.type", "enabled")

	if budgetTokens > 0 {
		request, _ = sjson.SetBytes(request, "thinking.budget_tokens", budgetTokens)
	}

	if effort != "" {
		request, _ = sjson.SetBytes(request, "effort", effort)
	}

	return request
}

// HasClaudeThinking checks if a Claude response contains thinking blocks.
func HasClaudeThinking(response []byte) bool {
	content := gjson.GetBytes(response, "content")
	if !content.Exists() || !content.IsArray() {
		return false
	}

	hasThinking := false
	content.ForEach(func(_, block gjson.Result) bool {
		blockType := block.Get("type").String()
		if blockType == "thinking" || blockType == "redacted_thinking" {
			hasThinking = true
			return false // Stop iteration
		}
		return true
	})

	return hasThinking
}

// GetClaudeThinkingTokens extracts thinking token count from Claude response.
func GetClaudeThinkingTokens(response []byte) int64 {
	return gjson.GetBytes(response, "usage.thinking_tokens").Int()
}

// ExtractClaudeTextContent extracts only text content from Claude response.
func ExtractClaudeTextContent(response []byte) string {
	content := gjson.GetBytes(response, "content")
	if !content.Exists() || !content.IsArray() {
		return ""
	}

	var text string
	content.ForEach(func(_, block gjson.Result) bool {
		if block.Get("type").String() == "text" {
			if text != "" {
				text += "\n"
			}
			text += block.Get("text").String()
		}
		return true
	})

	return text
}

// BuildClaudeThinkingResponse builds a Claude response with thinking blocks.
func BuildClaudeThinkingResponse(thinking, answer string, thinkingTokens int64) []byte {
	response := `{"content":[],"stop_reason":"end_turn"}`

	if thinking != "" {
		thinkingBlock := `{"type":"thinking"}`
		thinkingBlock, _ = sjson.Set(thinkingBlock, "thinking", thinking)
		response, _ = sjson.SetRaw(response, "content.-1", thinkingBlock)
	}

	if answer != "" {
		textBlock := `{"type":"text"}`
		textBlock, _ = sjson.Set(textBlock, "text", answer)
		response, _ = sjson.SetRaw(response, "content.-1", textBlock)
	}

	if thinkingTokens > 0 {
		response, _ = sjson.Set(response, "usage.thinking_tokens", thinkingTokens)
	}

	return []byte(response)
}
