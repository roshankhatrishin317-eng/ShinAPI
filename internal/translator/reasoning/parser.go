// Package reasoning provides extended thinking/reasoning support for various AI models.
package reasoning

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ReasoningParser provides methods to extract and handle reasoning content.
type ReasoningParser struct {
	config ReasoningConfig
}

// NewReasoningParser creates a new reasoning parser with the given config.
func NewReasoningParser(cfg ReasoningConfig) *ReasoningParser {
	return &ReasoningParser{config: cfg}
}

// ExtractThinking extracts thinking content from a response based on provider.
func (rp *ReasoningParser) ExtractThinking(response []byte, provider ReasoningProvider) ThinkingResult {
	switch provider {
	case ProviderClaude:
		return rp.extractClaudeThinking(response)
	case ProviderGemini:
		return rp.extractGeminiThinking(response)
	case ProviderDeepSeek:
		return rp.extractDeepSeekThinking(response)
	case ProviderOpenAI:
		return rp.extractOpenAIThinking(response)
	default:
		return ThinkingResult{}
	}
}

// EnhanceRequest adds reasoning parameters to a request based on provider.
func (rp *ReasoningParser) EnhanceRequest(request []byte, provider ReasoningProvider) []byte {
	if !rp.config.Enabled {
		return request
	}

	switch provider {
	case ProviderClaude:
		return rp.enhanceClaudeRequest(request)
	case ProviderGemini:
		return rp.enhanceGeminiRequest(request)
	default:
		return request
	}
}

// StripThinking removes thinking content from response if configured.
func (rp *ReasoningParser) StripThinking(response []byte, provider ReasoningProvider) []byte {
	if rp.config.ShowThinkingToClient {
		return response
	}

	switch provider {
	case ProviderClaude:
		return rp.stripClaudeThinking(response)
	case ProviderDeepSeek:
		return rp.stripDeepSeekThinking(response)
	default:
		return response
	}
}

// ExtractTokenUsage extracts token usage including thinking tokens.
func (rp *ReasoningParser) ExtractTokenUsage(response []byte, provider ReasoningProvider) TokenUsage {
	switch provider {
	case ProviderClaude:
		return rp.extractClaudeTokenUsage(response)
	case ProviderGemini:
		return rp.extractGeminiTokenUsage(response)
	default:
		return TokenUsage{}
	}
}

// GetRequiredHeaders returns any required headers for reasoning features.
func (rp *ReasoningParser) GetRequiredHeaders(provider ReasoningProvider) map[string]string {
	headers := make(map[string]string)

	if provider == ProviderClaude && rp.config.Claude.InterleavedTools {
		headers["anthropic-beta"] = "interleaved-thinking-2025-05-14"
	}

	return headers
}

// extractClaudeTokenUsage extracts token usage from Claude response.
func (rp *ReasoningParser) extractClaudeTokenUsage(response []byte) TokenUsage {
	usage := gjson.GetBytes(response, "usage")
	if !usage.Exists() {
		return TokenUsage{}
	}

	return TokenUsage{
		PromptTokens:     usage.Get("input_tokens").Int(),
		CompletionTokens: usage.Get("output_tokens").Int(),
		ThinkingTokens:   usage.Get("thinking_tokens").Int(),
		TotalTokens:      usage.Get("input_tokens").Int() + usage.Get("output_tokens").Int(),
	}
}

// extractGeminiTokenUsage extracts token usage from Gemini response.
func (rp *ReasoningParser) extractGeminiTokenUsage(response []byte) TokenUsage {
	usage := gjson.GetBytes(response, "usageMetadata")
	if !usage.Exists() {
		return TokenUsage{}
	}

	return TokenUsage{
		PromptTokens:     usage.Get("promptTokenCount").Int(),
		CompletionTokens: usage.Get("candidatesTokenCount").Int(),
		ThinkingTokens:   usage.Get("thinkingTokenCount").Int(),
		TotalTokens:      usage.Get("totalTokenCount").Int(),
	}
}

// stripClaudeThinking removes thinking blocks from Claude response.
func (rp *ReasoningParser) stripClaudeThinking(response []byte) []byte {
	content := gjson.GetBytes(response, "content")
	if !content.Exists() || !content.IsArray() {
		return response
	}

	// Filter out thinking blocks
	newContent := []byte("[]")
	content.ForEach(func(_, block gjson.Result) bool {
		blockType := block.Get("type").String()
		if blockType != "thinking" && blockType != "redacted_thinking" {
			newContent, _ = sjson.SetRawBytes(newContent, "-1", []byte(block.Raw))
		}
		return true
	})

	result, _ := sjson.SetRawBytes(response, "content", newContent)
	return result
}

// stripDeepSeekThinking removes <think> tags from DeepSeek response.
func (rp *ReasoningParser) stripDeepSeekThinking(response []byte) []byte {
	// DeepSeek thinking is in the content as <think>...</think>
	content := gjson.GetBytes(response, "choices.0.message.content").String()
	if content == "" {
		return response
	}

	// Use the DeepSeek extractor to get answer without thinking
	result := ExtractDeepSeekThinking([]byte(content))
	if result.Answer != "" {
		response, _ = sjson.SetBytes(response, "choices.0.message.content", result.Answer)
	}

	return response
}

// extractOpenAIThinking extracts reasoning from OpenAI o1/o3 models.
func (rp *ReasoningParser) extractOpenAIThinking(response []byte) ThinkingResult {
	// OpenAI o-series returns reasoning_content in the response
	reasoningContent := gjson.GetBytes(response, "choices.0.message.reasoning_content")
	if !reasoningContent.Exists() {
		return ThinkingResult{}
	}

	return ThinkingResult{
		Thinking: reasoningContent.String(),
		Answer:   gjson.GetBytes(response, "choices.0.message.content").String(),
	}
}

// extractDeepSeekThinking extracts thinking from DeepSeek R1 response.
func (rp *ReasoningParser) extractDeepSeekThinking(response []byte) ThinkingResult {
	// DeepSeek uses OpenAI-compatible format, thinking is in content as <think> tags
	content := gjson.GetBytes(response, "choices.0.message.content").String()
	if content == "" {
		return ThinkingResult{}
	}
	return ExtractDeepSeekThinking([]byte(content))
}
