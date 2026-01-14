// Package middleware provides HTTP middleware components for the API server.
// This file implements audit logging middleware that captures all API requests.
package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/audit"
	"github.com/tidwall/gjson"
)

// responseBodyWriter wraps gin.ResponseWriter to capture response body
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// AuditMiddleware creates a middleware that logs all API requests to the audit log.
// It captures request/response metadata including latency, tokens, and errors.
func AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip non-API paths
		path := c.Request.URL.Path
		if !shouldAudit(path) {
			c.Next()
			return
		}

		// Capture request body for model extraction
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Wrap response writer to capture response body
		rbw := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer(nil),
		}
		c.Writer = rbw

		// Record start time
		start := time.Now()

		// Extract request info before processing
		provider := detectProvider(path)
		model := extractModelFromBody(requestBody)
		streaming := isStreamingRequest(requestBody)

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Extract token counts from response
		inputTokens, outputTokens := extractTokensFromResponse(rbw.body.Bytes())

		// Check for context values set by handlers
		if ctxModel, exists := c.Get("audit_model"); exists {
			if m, ok := ctxModel.(string); ok && m != "" {
				model = m
			}
		}
		if ctxProvider, exists := c.Get("audit_provider"); exists {
			if p, ok := ctxProvider.(string); ok && p != "" {
				provider = p
			}
		}
		if ctxInputTokens, exists := c.Get("audit_input_tokens"); exists {
			if t, ok := ctxInputTokens.(int64); ok && t > 0 {
				inputTokens = t
			}
		}
		if ctxOutputTokens, exists := c.Get("audit_output_tokens"); exists {
			if t, ok := ctxOutputTokens.(int64); ok && t > 0 {
				outputTokens = t
			}
		}

		// Get auth info from context
		authID := getStringFromContext(c, "auth_id")
		authLabel := getStringFromContext(c, "auth_label")

		// Check if cached
		cached := false
		if ctxCached, exists := c.Get("audit_cached"); exists {
			if b, ok := ctxCached.(bool); ok {
				cached = b
			}
		}

		// Get error if any
		var reqError error
		if len(c.Errors) > 0 {
			reqError = c.Errors.Last().Err
		}

		// Log to audit
		audit.GetAuditLogger().LogResponse(
			provider,
			model,
			authID,
			authLabel,
			path,
			c.Request.Method,
			c.Writer.Status(),
			latency,
			inputTokens,
			outputTokens,
			streaming,
			cached,
			reqError,
		)
	}
}

// shouldAudit determines if a path should be audited
func shouldAudit(path string) bool {
	// Audit API endpoints
	if strings.HasPrefix(path, "/v1/") {
		return true
	}
	if strings.HasPrefix(path, "/v1beta/") {
		return true
	}
	// Skip management, dashboard, static files
	if strings.HasPrefix(path, "/v0/management") {
		return false
	}
	if strings.HasPrefix(path, "/dashboard") {
		return false
	}
	if strings.HasPrefix(path, "/static") {
		return false
	}
	if strings.HasPrefix(path, "/ws") {
		return false
	}
	return false
}

// detectProvider extracts the provider from the URL path
func detectProvider(path string) string {
	path = strings.ToLower(path)

	// Claude/Anthropic endpoints
	if strings.Contains(path, "/messages") && !strings.Contains(path, "/chat") {
		return "claude"
	}

	// Gemini endpoints
	if strings.Contains(path, "/gemini") || strings.Contains(path, "/v1beta/") {
		return "gemini"
	}
	if strings.Contains(path, ":generateContent") || strings.Contains(path, ":streamGenerateContent") {
		return "gemini"
	}

	// OpenAI is the default
	if strings.Contains(path, "/chat/completions") || strings.Contains(path, "/completions") {
		return "openai"
	}
	if strings.Contains(path, "/embeddings") {
		return "openai"
	}

	return "unknown"
}

// extractModelFromBody extracts the model name from request body
func extractModelFromBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	return gjson.GetBytes(body, "model").String()
}

// isStreamingRequest checks if the request is for streaming
func isStreamingRequest(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	return gjson.GetBytes(body, "stream").Bool()
}

// extractTokensFromResponse extracts token counts from response body
func extractTokensFromResponse(body []byte) (input, output int64) {
	if len(body) == 0 {
		return 0, 0
	}

	usage := gjson.GetBytes(body, "usage")
	if !usage.Exists() {
		return 0, 0
	}

	// OpenAI format
	input = usage.Get("prompt_tokens").Int()
	output = usage.Get("completion_tokens").Int()

	// Claude format fallback
	if input == 0 {
		input = usage.Get("input_tokens").Int()
	}
	if output == 0 {
		output = usage.Get("output_tokens").Int()
	}

	return input, output
}

// getStringFromContext safely extracts a string from gin context
func getStringFromContext(c *gin.Context, key string) string {
	if val, exists := c.Get(key); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}
