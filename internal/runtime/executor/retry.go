// Package executor provides runtime execution capabilities for various AI service providers.
// This file implements retry utilities with exponential backoff and jitter.
package executor

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"
)

// RetryConfig configures exponential backoff retry behavior.
type RetryConfig struct {
	// InitialDelay is the base delay for the first retry (default: 1s).
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries (default: 30s).
	MaxDelay time.Duration
	// Multiplier is the exponential factor for each retry (default: 2.0).
	Multiplier float64
	// JitterFactor is the random jitter factor (0.0-1.0, default: 0.2 = 20%).
	JitterFactor float64
	// MaxRetries is the maximum number of retry attempts (default: 3).
	MaxRetries int
}

// DefaultRetryConfig returns sensible defaults for API retry behavior.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		InitialDelay: 2 * time.Second,
		MaxDelay:     60 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.25,
		MaxRetries:   5,
	}
}

var (
	retryRand      = rand.New(rand.NewSource(time.Now().UnixNano()))
	retryRandMutex sync.Mutex
)

// CalculateBackoff computes the delay for a given retry attempt using exponential backoff with jitter.
// If serverDelay is provided (from Retry-After header), it takes precedence with jitter applied.
func CalculateBackoff(cfg RetryConfig, attempt int, serverDelay *time.Duration) time.Duration {
	var baseDelay time.Duration

	if serverDelay != nil && *serverDelay > 0 {
		// Use server-provided delay as base
		baseDelay = *serverDelay
	} else {
		// Calculate exponential delay: initialDelay * (multiplier ^ attempt)
		delaySeconds := float64(cfg.InitialDelay.Seconds()) * math.Pow(cfg.Multiplier, float64(attempt))
		baseDelay = time.Duration(delaySeconds * float64(time.Second))

		// Cap at max delay
		if baseDelay > cfg.MaxDelay {
			baseDelay = cfg.MaxDelay
		}
	}

	// Apply jitter: delay * (1 ± jitterFactor)
	if cfg.JitterFactor > 0 {
		retryRandMutex.Lock()
		jitter := (retryRand.Float64()*2 - 1) * cfg.JitterFactor // Range: [-jitterFactor, +jitterFactor]
		retryRandMutex.Unlock()
		baseDelay = time.Duration(float64(baseDelay) * (1 + jitter))
	}

	// Ensure minimum delay of 100ms
	if baseDelay < 100*time.Millisecond {
		baseDelay = 100 * time.Millisecond
	}

	return baseDelay
}

// SleepWithContext sleeps for the specified duration, returning early if the context is cancelled.
// Returns true if the sleep completed, false if interrupted by context cancellation.
func SleepWithContext(ctx context.Context, duration time.Duration) bool {
	if ctx == nil {
		time.Sleep(duration)
		return true
	}

	select {
	case <-time.After(duration):
		return true
	case <-ctx.Done():
		return false
	}
}

// IsRetryableError determines if an HTTP status code is retryable.
// Retryable codes: 429 (Too Many Requests), 500, 502, 503, 504.
func IsRetryableError(statusCode int) bool {
	switch statusCode {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

// RetryableFunc is a function that can be retried.
// It should return an error and optionally a status code for retry decisions.
type RetryableFunc func() (statusCode int, retryAfter *time.Duration, err error)

// ExecuteWithRetry executes a function with exponential backoff retry.
// It retries the function if it returns a retryable error status code.
func ExecuteWithRetry(ctx context.Context, cfg RetryConfig, fn RetryableFunc) error {
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		statusCode, retryAfter, err := fn()

		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry if not a retryable error
		if !IsRetryableError(statusCode) {
			return err
		}

		// Don't retry if we've exhausted attempts
		if attempt >= cfg.MaxRetries {
			return err
		}

		// Calculate backoff delay
		delay := CalculateBackoff(cfg, attempt, retryAfter)

		// Wait before retrying
		if !SleepWithContext(ctx, delay) {
			// Context was cancelled
			return ctx.Err()
		}
	}

	return lastErr
}

