// Package errors provides provider-specific error parsing and classification.
// It enables smart retry logic based on error type and provider.
package errors

import (
	"time"

	"github.com/tidwall/gjson"
)

// ProviderError represents a parsed error from an AI provider.
type ProviderError struct {
	// Provider is the source provider (openai, anthropic, google, etc.)
	Provider string `json:"provider"`

	// StatusCode is the HTTP status code
	StatusCode int `json:"status_code"`

	// Code is the provider-specific error code
	Code string `json:"code"`

	// Type is the error type category
	Type string `json:"type"`

	// Message is the human-readable error message
	Message string `json:"message"`

	// Retryable indicates if the request can be retried
	Retryable bool `json:"retryable"`

	// RetryAfter suggests how long to wait before retrying
	RetryAfter time.Duration `json:"retry_after"`

	// ShouldFailover indicates if request should try another provider
	ShouldFailover bool `json:"should_failover"`
}

// RetryConfig holds retry behavior configuration.
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int `yaml:"max-attempts" json:"max_attempts"`

	// InitialDelayMs is the initial delay between retries in milliseconds
	InitialDelayMs int `yaml:"initial-delay-ms" json:"initial_delay_ms"`

	// MaxDelayMs is the maximum delay between retries in milliseconds
	MaxDelayMs int `yaml:"max-delay-ms" json:"max_delay_ms"`

	// Multiplier is the backoff multiplier
	Multiplier float64 `yaml:"multiplier" json:"multiplier"`

	// Jitter adds randomness to delay (0.0 to 1.0)
	Jitter float64 `yaml:"jitter" json:"jitter"`

	// RetryableStatusCodes lists HTTP status codes to retry
	RetryableStatusCodes []int `yaml:"retryable-status-codes" json:"retryable_status_codes"`
}

// DefaultRetryConfig returns sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:          3,
		InitialDelayMs:       1000,
		MaxDelayMs:           60000,
		Multiplier:           2.0,
		Jitter:               0.1,
		RetryableStatusCodes: []int{429, 500, 502, 503, 504},
	}
}

// ParseProviderError parses an error response from any provider.
func ParseProviderError(provider string, statusCode int, body []byte) *ProviderError {
	switch provider {
	case "anthropic", "claude":
		return parseAnthropicError(statusCode, body)
	case "openai":
		return parseOpenAIError(statusCode, body)
	case "google", "gemini":
		return parseGeminiError(statusCode, body)
	default:
		return parseGenericError(provider, statusCode, body)
	}
}

// parseAnthropicError parses Anthropic/Claude API errors.
func parseAnthropicError(statusCode int, body []byte) *ProviderError {
	err := &ProviderError{
		Provider:   "anthropic",
		StatusCode: statusCode,
		Retryable:  false,
	}

	// Parse error fields
	errObj := gjson.GetBytes(body, "error")
	if errObj.Exists() {
		err.Type = errObj.Get("type").String()
		err.Message = errObj.Get("message").String()
	} else {
		err.Message = string(body)
	}

	// Classify error
	switch statusCode {
	case 400:
		err.Code = "invalid_request"
		err.Retryable = false
	case 401:
		err.Code = "authentication_error"
		err.Retryable = false
	case 403:
		err.Code = "permission_denied"
		err.Retryable = false
	case 404:
		err.Code = "not_found"
		err.Retryable = false
	case 429:
		err.Code = "rate_limit_exceeded"
		err.Retryable = true
		err.RetryAfter = 60 * time.Second
		err.ShouldFailover = true
	case 500:
		err.Code = "api_error"
		err.Retryable = true
		err.RetryAfter = 5 * time.Second
	case 529:
		err.Code = "overloaded_error"
		err.Retryable = true
		err.RetryAfter = 30 * time.Second
		err.ShouldFailover = true
	default:
		if statusCode >= 500 {
			err.Code = "server_error"
			err.Retryable = true
			err.RetryAfter = 5 * time.Second
		}
	}

	// Check for specific error types
	if err.Type == "overloaded_error" {
		err.Retryable = true
		err.RetryAfter = 30 * time.Second
		err.ShouldFailover = true
	}

	return err
}

// parseOpenAIError parses OpenAI API errors.
func parseOpenAIError(statusCode int, body []byte) *ProviderError {
	err := &ProviderError{
		Provider:   "openai",
		StatusCode: statusCode,
		Retryable:  false,
	}

	// Parse error fields
	errObj := gjson.GetBytes(body, "error")
	if errObj.Exists() {
		err.Type = errObj.Get("type").String()
		err.Code = errObj.Get("code").String()
		err.Message = errObj.Get("message").String()
	} else {
		err.Message = string(body)
	}

	// Classify error
	switch statusCode {
	case 400:
		err.Retryable = false
		// Check for context length error
		if containsAny(err.Message, "context_length_exceeded", "maximum context length") {
			err.Code = "context_length_exceeded"
		}
	case 401:
		err.Code = "invalid_api_key"
		err.Retryable = false
	case 403:
		err.Code = "insufficient_quota"
		err.Retryable = false
		err.ShouldFailover = true
	case 429:
		err.Retryable = true
		err.ShouldFailover = true
		// Check for rate limit type
		if containsAny(err.Code, "rate_limit") {
			err.RetryAfter = 60 * time.Second
		} else {
			err.RetryAfter = 20 * time.Second
		}
	case 500, 502, 503:
		err.Code = "server_error"
		err.Retryable = true
		err.RetryAfter = 5 * time.Second
	case 504:
		err.Code = "timeout"
		err.Retryable = true
		err.RetryAfter = 10 * time.Second
	}

	return err
}

// parseGeminiError parses Google Gemini API errors.
func parseGeminiError(statusCode int, body []byte) *ProviderError {
	err := &ProviderError{
		Provider:   "gemini",
		StatusCode: statusCode,
		Retryable:  false,
	}

	// Parse error fields (Google uses different structure)
	errObj := gjson.GetBytes(body, "error")
	if errObj.Exists() {
		err.Code = errObj.Get("code").String()
		err.Message = errObj.Get("message").String()
		err.Type = errObj.Get("status").String()
	} else {
		err.Message = string(body)
	}

	// Classify error
	switch statusCode {
	case 400:
		err.Retryable = false
		if containsAny(err.Message, "context", "token", "length") {
			err.Code = "context_length_exceeded"
		}
	case 401:
		err.Code = "unauthenticated"
		err.Retryable = false
	case 403:
		err.Code = "permission_denied"
		err.Retryable = false
		err.ShouldFailover = true
	case 429:
		err.Code = "resource_exhausted"
		err.Retryable = true
		err.RetryAfter = 60 * time.Second
		err.ShouldFailover = true
	case 500:
		err.Code = "internal"
		err.Retryable = true
		err.RetryAfter = 5 * time.Second
	case 503:
		err.Code = "unavailable"
		err.Retryable = true
		err.RetryAfter = 30 * time.Second
		err.ShouldFailover = true
	default:
		if statusCode >= 500 {
			err.Code = "server_error"
			err.Retryable = true
			err.RetryAfter = 5 * time.Second
		}
	}

	return err
}

// parseGenericError creates a generic error for unknown providers.
func parseGenericError(provider string, statusCode int, body []byte) *ProviderError {
	err := &ProviderError{
		Provider:   provider,
		StatusCode: statusCode,
		Message:    string(body),
		Retryable:  false,
	}

	// Use status code to determine retryability
	switch {
	case statusCode == 429:
		err.Code = "rate_limit"
		err.Retryable = true
		err.RetryAfter = 60 * time.Second
		err.ShouldFailover = true
	case statusCode >= 500 && statusCode < 600:
		err.Code = "server_error"
		err.Retryable = true
		err.RetryAfter = 5 * time.Second
	case statusCode >= 400 && statusCode < 500:
		err.Code = "client_error"
		err.Retryable = false
	}

	return err
}

// IsRetryable checks if a status code is retryable based on config.
func IsRetryable(statusCode int, cfg RetryConfig) bool {
	for _, code := range cfg.RetryableStatusCodes {
		if code == statusCode {
			return true
		}
	}
	return false
}

// CalculateBackoff calculates the delay for a retry attempt.
func CalculateBackoff(attempt int, cfg RetryConfig) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}

	// Calculate base delay with exponential backoff
	delay := float64(cfg.InitialDelayMs)
	for i := 1; i < attempt; i++ {
		delay *= cfg.Multiplier
	}

	// Cap at max delay
	if delay > float64(cfg.MaxDelayMs) {
		delay = float64(cfg.MaxDelayMs)
	}

	// Add jitter
	if cfg.Jitter > 0 {
		jitter := delay * cfg.Jitter
		// Simple deterministic jitter based on attempt
		delay += jitter * float64(attempt%3) / 2
	}

	return time.Duration(delay) * time.Millisecond
}

// ShouldRetry determines if a request should be retried.
func ShouldRetry(err *ProviderError, attempt int, cfg RetryConfig) bool {
	if err == nil {
		return false
	}

	if attempt >= cfg.MaxAttempts {
		return false
	}

	if !err.Retryable && !IsRetryable(err.StatusCode, cfg) {
		return false
	}

	return true
}

// Helper function to check for substrings
func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// ErrorHandler provides error handling with retry logic.
type ErrorHandler struct {
	config RetryConfig
}

// NewErrorHandler creates a new error handler with the given config.
func NewErrorHandler(cfg RetryConfig) *ErrorHandler {
	if cfg.MaxAttempts == 0 {
		cfg = DefaultRetryConfig()
	}
	return &ErrorHandler{config: cfg}
}

// Config returns the retry configuration.
func (h *ErrorHandler) Config() RetryConfig {
	return h.config
}

// ParseError parses a provider error from status code and body.
func (h *ErrorHandler) ParseError(provider string, statusCode int, body []byte) *ProviderError {
	return ParseProviderError(provider, statusCode, body)
}

// ShouldRetry determines if a request should be retried.
func (h *ErrorHandler) ShouldRetry(err *ProviderError, attempt int) bool {
	return ShouldRetry(err, attempt, h.config)
}

// GetBackoff calculates the backoff duration for the given attempt.
func (h *ErrorHandler) GetBackoff(attempt int) time.Duration {
	return CalculateBackoff(attempt, h.config)
}
