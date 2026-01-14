// Package management provides HTTP handlers for the management API.
// This file implements API playground endpoints for testing API calls.
package management

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/audit"
	log "github.com/sirupsen/logrus"
)

// PlaygroundRequest represents an API playground test request.
type PlaygroundRequest struct {
	Provider    string            `json:"provider"`
	Model       string            `json:"model"`
	Messages    []PlaygroundMsg   `json:"messages"`
	Stream      bool              `json:"stream"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// PlaygroundMsg represents a message in the playground request.
type PlaygroundMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// PlaygroundResponse represents an API playground test response.
type PlaygroundResponse struct {
	Success      bool              `json:"success"`
	StatusCode   int               `json:"status_code"`
	LatencyMs    int64             `json:"latency_ms"`
	Response     json.RawMessage   `json:"response,omitempty"`
	Error        string            `json:"error,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	InputTokens  int64             `json:"input_tokens,omitempty"`
	OutputTokens int64             `json:"output_tokens,omitempty"`
	Model        string            `json:"model,omitempty"`
}

// ExecutePlayground handles API playground requests.
func (h *Handler) ExecutePlayground(c *gin.Context) {
	var req PlaygroundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.Provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is required"})
		return
	}
	if req.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}
	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "messages are required"})
		return
	}

	// Build the internal API URL based on provider
	var apiURL string
	switch strings.ToLower(req.Provider) {
	case "openai", "codex":
		apiURL = "/v1/chat/completions"
	case "claude", "anthropic":
		apiURL = "/v1/messages"
	case "gemini", "google":
		apiURL = "/v1/chat/completions" // Use OpenAI-compatible endpoint
	default:
		apiURL = "/v1/chat/completions"
	}

	// Build the request body in OpenAI format
	requestBody := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   req.Stream,
	}
	if req.MaxTokens > 0 {
		requestBody["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		requestBody["temperature"] = req.Temperature
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode request"})
		return
	}

	// Create internal request
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	// Get server port from config
	port := 8080
	if h.cfg != nil && h.cfg.Port > 0 {
		port = h.cfg.Port
	}

	internalURL := "http://127.0.0.1:" + itoa(port) + apiURL
	internalReq, err := http.NewRequestWithContext(ctx, http.MethodPost, internalURL, bytes.NewReader(bodyBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	internalReq.Header.Set("Content-Type", "application/json")

	// Copy authorization header from original request if present
	if auth := c.GetHeader("Authorization"); auth != "" {
		internalReq.Header.Set("Authorization", auth)
	} else if h.cfg != nil && len(h.cfg.APIKeys) > 0 {
		// Use first configured API key for internal testing
		internalReq.Header.Set("Authorization", "Bearer "+h.cfg.APIKeys[0])
	}

	// Add custom headers
	for k, v := range req.Headers {
		internalReq.Header.Set(k, v)
	}

	// Execute request
	startTime := time.Now()
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(internalReq)
	latency := time.Since(startTime)

	if err != nil {
		// Log to audit
		audit.GetAuditLogger().LogResponse(
			req.Provider, req.Model, "", "", apiURL, "POST",
			0, latency, 0, 0, req.Stream, false, err,
		)

		c.JSON(http.StatusOK, PlaygroundResponse{
			Success:    false,
			StatusCode: 0,
			LatencyMs:  latency.Milliseconds(),
			Error:      "Request failed: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusOK, PlaygroundResponse{
			Success:    false,
			StatusCode: resp.StatusCode,
			LatencyMs:  latency.Milliseconds(),
			Error:      "Failed to read response: " + err.Error(),
		})
		return
	}

	// Extract response headers
	respHeaders := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	// Parse token counts if available
	var inputTokens, outputTokens int64
	var parsedResp map[string]interface{}
	if err := json.Unmarshal(respBody, &parsedResp); err == nil {
		if usage, ok := parsedResp["usage"].(map[string]interface{}); ok {
			if v, ok := usage["prompt_tokens"].(float64); ok {
				inputTokens = int64(v)
			}
			if v, ok := usage["completion_tokens"].(float64); ok {
				outputTokens = int64(v)
			}
		}
	}

	// Log to audit
	var auditErr error
	if resp.StatusCode >= 400 {
		auditErr = &playgroundError{msg: string(respBody)}
	}
	audit.GetAuditLogger().LogResponse(
		req.Provider, req.Model, "", "playground", apiURL, "POST",
		resp.StatusCode, latency, inputTokens, outputTokens, req.Stream, false, auditErr,
	)

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	var errorMsg string
	if !success {
		errorMsg = string(respBody)
	}

	c.JSON(http.StatusOK, PlaygroundResponse{
		Success:      success,
		StatusCode:   resp.StatusCode,
		LatencyMs:    latency.Milliseconds(),
		Response:     json.RawMessage(respBody),
		Error:        errorMsg,
		Headers:      respHeaders,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Model:        req.Model,
	})
}

// GetPlaygroundModels returns available models for the playground.
func (h *Handler) GetPlaygroundModels(c *gin.Context) {
	// Return a curated list of common models
	models := []map[string]interface{}{
		{"id": "gpt-4o", "provider": "openai", "name": "GPT-4o"},
		{"id": "gpt-4o-mini", "provider": "openai", "name": "GPT-4o Mini"},
		{"id": "gpt-4-turbo", "provider": "openai", "name": "GPT-4 Turbo"},
		{"id": "claude-sonnet-4-20250514", "provider": "claude", "name": "Claude Sonnet 4"},
		{"id": "claude-3-5-sonnet-20241022", "provider": "claude", "name": "Claude 3.5 Sonnet"},
		{"id": "claude-3-opus-20240229", "provider": "claude", "name": "Claude 3 Opus"},
		{"id": "gemini-2.0-flash", "provider": "gemini", "name": "Gemini 2.0 Flash"},
		{"id": "gemini-1.5-pro", "provider": "gemini", "name": "Gemini 1.5 Pro"},
		{"id": "gemini-1.5-flash", "provider": "gemini", "name": "Gemini 1.5 Flash"},
	}

	c.JSON(http.StatusOK, gin.H{
		"models": models,
	})
}

// GetPlaygroundTemplates returns template prompts for the playground.
func (h *Handler) GetPlaygroundTemplates(c *gin.Context) {
	templates := []map[string]interface{}{
		{
			"id":    "hello",
			"name":  "Hello World",
			"model": "gpt-4o-mini",
			"messages": []map[string]string{
				{"role": "user", "content": "Say hello!"},
			},
		},
		{
			"id":    "explain-code",
			"name":  "Explain Code",
			"model": "gpt-4o",
			"messages": []map[string]string{
				{"role": "system", "content": "You are a helpful coding assistant."},
				{"role": "user", "content": "Explain this code:\n\n```python\ndef fibonacci(n):\n    if n <= 1:\n        return n\n    return fibonacci(n-1) + fibonacci(n-2)\n```"},
			},
		},
		{
			"id":    "translate",
			"name":  "Translate Text",
			"model": "gpt-4o-mini",
			"messages": []map[string]string{
				{"role": "system", "content": "You are a professional translator."},
				{"role": "user", "content": "Translate this to Spanish: Hello, how are you?"},
			},
		},
		{
			"id":    "json-extract",
			"name":  "Extract JSON",
			"model": "gpt-4o",
			"messages": []map[string]string{
				{"role": "system", "content": "Extract structured data as JSON."},
				{"role": "user", "content": "Extract name, email, and phone from: John Doe, john@example.com, +1-555-123-4567"},
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
	})
}

type playgroundError struct {
	msg string
}

func (e *playgroundError) Error() string {
	return e.msg
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}

func init() {
	// Suppress unused import warning
	_ = log.Debug
}
