// Package reasoning provides extended thinking/reasoning support for various AI models.
package reasoning

import (
	"regexp"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Gemini 3 Pro thinking response format:
// {
//   "candidates": [{
//     "content": {
//       "parts": [{"text": "..."}],
//       "thought_summary": "...",
//       "thought_signature": "..."
//     }
//   }],
//   "usageMetadata": {"thinkingTokenCount": N, ...}
// }

// Regex patterns for Gemini thinking extraction
var (
	geminiThoughtSummaryPattern   = regexp.MustCompile(`"thought_summary"\s*:\s*"([^"]*)"`)
	geminiThoughtSignaturePattern = regexp.MustCompile(`"thought_signature"\s*:\s*"([^"]*)"`)
)

// extractGeminiThinking extracts thinking content from Gemini response.
func (rp *ReasoningParser) extractGeminiThinking(response []byte) ThinkingResult {
	result := ThinkingResult{}

	// Extract thought summary
	content := gjson.GetBytes(response, "candidates.0.content")
	if !content.Exists() {
		return result
	}

	if thoughtSummary := content.Get("thought_summary"); thoughtSummary.Exists() {
		result.ThinkingSummary = thoughtSummary.String()
		result.Thinking = thoughtSummary.String()
	}

	// Extract thought signature (CRITICAL for multi-turn)
	if thoughtSig := content.Get("thought_signature"); thoughtSig.Exists() {
		result.Signature = thoughtSig.String()
	}

	// Extract text answer
	parts := content.Get("parts")
	if parts.Exists() && parts.IsArray() {
		parts.ForEach(func(_, part gjson.Result) bool {
			if text := part.Get("text"); text.Exists() {
				if result.Answer != "" {
					result.Answer += "\n"
				}
				result.Answer += text.String()
			}
			return true
		})
	}

	// Extract thinking tokens from usage metadata
	if usage := gjson.GetBytes(response, "usageMetadata"); usage.Exists() {
		result.ThinkingTokens = usage.Get("thinkingTokenCount").Int()
	}

	return result
}

// enhanceGeminiRequest adds thinking parameters to Gemini request.
func (rp *ReasoningParser) enhanceGeminiRequest(request []byte) []byte {
	cfg := rp.config.Gemini

	// Add thinkingConfig
	if cfg.DefaultThinkingLevel != "" {
		thinkingLevel := cfg.DefaultThinkingLevel
		// Gemini uses uppercase: "LOW" or "HIGH"
		if thinkingLevel == "low" {
			thinkingLevel = "LOW"
		} else if thinkingLevel == "high" {
			thinkingLevel = "HIGH"
		}
		request, _ = sjson.SetBytes(request, "generationConfig.thinkingConfig.thinkingLevel", thinkingLevel)
	}

	if cfg.IncludeThoughts {
		request, _ = sjson.SetBytes(request, "generationConfig.thinkingConfig.includeThoughts", true)
	}

	// Force temperature to 1.0 for Gemini 3 (recommended)
	if cfg.ForceTemperature1 {
		request, _ = sjson.SetBytes(request, "generationConfig.temperature", 1.0)
	}

	return request
}

// EnhanceGeminiRequestWithParams adds specific thinking parameters to Gemini request.
func EnhanceGeminiRequestWithParams(request []byte, thinkingLevel string, includeThoughts bool, forceTemp1 bool) []byte {
	if thinkingLevel != "" {
		// Normalize to uppercase
		level := thinkingLevel
		if level == "low" {
			level = "LOW"
		} else if level == "high" {
			level = "HIGH"
		}
		request, _ = sjson.SetBytes(request, "generationConfig.thinkingConfig.thinkingLevel", level)
	}

	request, _ = sjson.SetBytes(request, "generationConfig.thinkingConfig.includeThoughts", includeThoughts)

	if forceTemp1 {
		request, _ = sjson.SetBytes(request, "generationConfig.temperature", 1.0)
	}

	return request
}

// HasGeminiThinking checks if a Gemini response contains thinking content.
func HasGeminiThinking(response []byte) bool {
	content := gjson.GetBytes(response, "candidates.0.content")
	if !content.Exists() {
		return false
	}

	return content.Get("thought_summary").Exists() || content.Get("thought_signature").Exists()
}

// GetGeminiThinkingTokens extracts thinking token count from Gemini response.
func GetGeminiThinkingTokens(response []byte) int64 {
	return gjson.GetBytes(response, "usageMetadata.thinkingTokenCount").Int()
}

// ExtractGeminiSignature extracts thought signature from Gemini response.
// This is CRITICAL for multi-turn conversations with thinking.
func ExtractGeminiSignature(response []byte) string {
	return gjson.GetBytes(response, "candidates.0.content.thought_signature").String()
}

// ExtractGeminiSignatures extracts all thought signatures from a conversation history.
func ExtractGeminiSignatures(responses [][]byte) []string {
	var signatures []string
	for _, resp := range responses {
		if sig := ExtractGeminiSignature(resp); sig != "" {
			signatures = append(signatures, sig)
		}
	}
	return signatures
}

// AddGeminiSignaturesToRequest adds preserved thought signatures to a Gemini request.
// This is required for multi-turn conversations with function calling.
func AddGeminiSignaturesToRequest(request []byte, signatures []string) []byte {
	if len(signatures) == 0 {
		return request
	}

	// Signatures are added to the request for context preservation
	for i, sig := range signatures {
		request, _ = sjson.SetBytes(request, "thoughtSignatures."+string(rune('0'+i)), sig)
	}

	return request
}

// ExtractGeminiTextContent extracts only text content from Gemini response.
func ExtractGeminiTextContent(response []byte) string {
	parts := gjson.GetBytes(response, "candidates.0.content.parts")
	if !parts.Exists() || !parts.IsArray() {
		return ""
	}

	var text string
	parts.ForEach(func(_, part gjson.Result) bool {
		if t := part.Get("text"); t.Exists() {
			if text != "" {
				text += "\n"
			}
			text += t.String()
		}
		return true
	})

	return text
}

// BuildGeminiThinkingResponse builds a Gemini response with thinking content.
func BuildGeminiThinkingResponse(answer, thoughtSummary, signature string, thinkingTokens int64) []byte {
	response := `{"candidates":[{"content":{"parts":[]}}]}`

	// Add text part
	if answer != "" {
		textPart := `{"text":""}`
		textPart, _ = sjson.Set(textPart, "text", answer)
		response, _ = sjson.SetRaw(response, "candidates.0.content.parts.-1", textPart)
	}

	// Add thought summary
	if thoughtSummary != "" {
		response, _ = sjson.Set(response, "candidates.0.content.thought_summary", thoughtSummary)
	}

	// Add thought signature
	if signature != "" {
		response, _ = sjson.Set(response, "candidates.0.content.thought_signature", signature)
	}

	// Add usage metadata
	if thinkingTokens > 0 {
		response, _ = sjson.Set(response, "usageMetadata.thinkingTokenCount", thinkingTokens)
	}

	return []byte(response)
}
