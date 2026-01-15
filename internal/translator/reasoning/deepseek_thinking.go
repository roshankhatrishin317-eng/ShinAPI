// Package reasoning provides extended thinking/reasoning support for various AI models.
package reasoning

import (
	"regexp"
	"strings"
)

// DeepSeek R1 thinking format:
// Response content contains <think>...</think> tags with reasoning,
// followed by the actual answer.
//
// Example:
// <think>
// Let me analyze this step by step...
// First, I need to consider...
// </think>
// The answer is...

// Regex patterns for DeepSeek thinking extraction
// Using (?s) flag to make . match newlines
var (
	deepSeekThinkPattern       = regexp.MustCompile(`(?s)<think>(.*?)</think>`)
	deepSeekThinkOpenPattern   = regexp.MustCompile(`<think>`)
	deepSeekThinkClosePattern  = regexp.MustCompile(`</think>`)
	deepSeekReasoningPattern   = regexp.MustCompile(`(?s)<reasoning>(.*?)</reasoning>`)
	deepSeekThoughtPattern     = regexp.MustCompile(`(?s)<thought>(.*?)</thought>`)
)

// ExtractDeepSeekThinking extracts thinking content from DeepSeek R1 response.
// Returns separated thinking and answer content.
func ExtractDeepSeekThinking(content []byte) ThinkingResult {
	result := ThinkingResult{}
	text := string(content)

	// Try multiple tag patterns (DeepSeek R1 may use different tags)
	patterns := []*regexp.Regexp{
		deepSeekThinkPattern,
		deepSeekReasoningPattern,
		deepSeekThoughtPattern,
	}

	var thinkingParts []string
	remainingText := text

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(remainingText, -1)
		for _, match := range matches {
			if len(match) > 1 {
				thinking := strings.TrimSpace(match[1])
				if thinking != "" {
					thinkingParts = append(thinkingParts, thinking)
				}
			}
		}
		// Remove matched tags from text
		remainingText = pattern.ReplaceAllString(remainingText, "")
	}

	// Combine all thinking parts
	result.Thinking = strings.Join(thinkingParts, "\n\n")

	// The remaining text is the answer
	result.Answer = strings.TrimSpace(remainingText)

	return result
}

// HasDeepSeekThinking checks if content contains DeepSeek thinking tags.
func HasDeepSeekThinking(content []byte) bool {
	text := string(content)
	return deepSeekThinkPattern.MatchString(text) ||
		deepSeekReasoningPattern.MatchString(text) ||
		deepSeekThoughtPattern.MatchString(text)
}

// StripDeepSeekThinking removes thinking tags from content, returning only the answer.
func StripDeepSeekThinking(content []byte) string {
	result := ExtractDeepSeekThinking(content)
	return result.Answer
}

// ExtractDeepSeekThinkingFromResponse extracts thinking from a full OpenAI-format response.
func ExtractDeepSeekThinkingFromResponse(response []byte) ThinkingResult {
	// DeepSeek uses OpenAI-compatible format
	// The thinking is embedded in the content
	contentStr := string(response)
	
	// Find content field in JSON
	// Simple extraction - look for "content": "..."
	contentStart := strings.Index(contentStr, `"content":"`)
	if contentStart == -1 {
		contentStart = strings.Index(contentStr, `"content": "`)
	}
	if contentStart == -1 {
		return ThinkingResult{}
	}

	// Find the content value
	contentStart = strings.Index(contentStr[contentStart:], `"`) + contentStart + 1
	contentStart = strings.Index(contentStr[contentStart:], `"`) + contentStart + 1
	
	contentEnd := contentStart
	escaped := false
	for i := contentStart; i < len(contentStr); i++ {
		if escaped {
			escaped = false
			continue
		}
		if contentStr[i] == '\\' {
			escaped = true
			continue
		}
		if contentStr[i] == '"' {
			contentEnd = i
			break
		}
	}

	if contentEnd <= contentStart {
		return ThinkingResult{}
	}

	content := contentStr[contentStart:contentEnd]
	// Unescape the content
	content = strings.ReplaceAll(content, `\n`, "\n")
	content = strings.ReplaceAll(content, `\"`, `"`)
	content = strings.ReplaceAll(content, `\\`, `\`)

	return ExtractDeepSeekThinking([]byte(content))
}

// IsDeepSeekReasoningModel checks if a model is a DeepSeek reasoning model.
func IsDeepSeekReasoningModel(model string) bool {
	lowerModel := strings.ToLower(model)
	return strings.Contains(lowerModel, "deepseek-r1") ||
		strings.Contains(lowerModel, "deepseek-reasoner") ||
		strings.Contains(lowerModel, "r1-")
}

// FormatDeepSeekThinkingForClient formats thinking content for client display.
func FormatDeepSeekThinkingForClient(thinking, answer string, showThinking bool) string {
	if !showThinking || thinking == "" {
		return answer
	}

	// Return thinking in collapsible format
	return "<details>\n<summary>Thinking</summary>\n\n" + thinking + "\n\n</details>\n\n" + answer
}

// EstimateDeepSeekThinkingTokens estimates thinking token count from content.
// This is a rough estimate since DeepSeek doesn't provide separate thinking token counts.
func EstimateDeepSeekThinkingTokens(thinking string) int64 {
	// Rough estimate: ~4 characters per token (for English)
	return int64(len(thinking) / 4)
}

// BuildDeepSeekResponse builds a response with embedded thinking tags.
func BuildDeepSeekResponse(thinking, answer string) string {
	if thinking == "" {
		return answer
	}
	return "<think>\n" + thinking + "\n</think>\n\n" + answer
}
