package executor

import (
	"context"
	"testing"
	"time"
)

func TestCalculateBackoff_ExponentialGrowth(t *testing.T) {
	cfg := RetryConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0, // No jitter for predictable testing
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 30 * time.Second}, // Capped at MaxDelay
		{6, 30 * time.Second}, // Still capped
	}

	for _, tt := range tests {
		result := CalculateBackoff(cfg, tt.attempt, nil)
		if result != tt.expected {
			t.Errorf("attempt %d: expected %v, got %v", tt.attempt, tt.expected, result)
		}
	}
}

func TestCalculateBackoff_ServerDelayTakesPrecedence(t *testing.T) {
	cfg := RetryConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0,
	}

	serverDelay := 5 * time.Second
	result := CalculateBackoff(cfg, 0, &serverDelay)

	if result != serverDelay {
		t.Errorf("expected server delay %v, got %v", serverDelay, result)
	}
}

func TestCalculateBackoff_WithJitter(t *testing.T) {
	cfg := RetryConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.2, // 20% jitter
	}

	// Run multiple times to verify jitter produces variation
	results := make(map[time.Duration]bool)
	for i := 0; i < 10; i++ {
		result := CalculateBackoff(cfg, 0, nil)
		results[result] = true

		// Should be within 20% of 1 second
		if result < 800*time.Millisecond || result > 1200*time.Millisecond {
			t.Errorf("result %v outside expected range [800ms, 1200ms]", result)
		}
	}

	// With 20% jitter over 10 runs, we should see some variation
	if len(results) < 2 {
		t.Log("Warning: jitter may not be producing sufficient variation")
	}
}

func TestCalculateBackoff_MinimumDelay(t *testing.T) {
	cfg := RetryConfig{
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0,
	}

	result := CalculateBackoff(cfg, 0, nil)

	// Should be at least 100ms (minimum enforced)
	if result < 100*time.Millisecond {
		t.Errorf("expected at least 100ms, got %v", result)
	}
}

func TestSleepWithContext_Completion(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	completed := SleepWithContext(ctx, 50*time.Millisecond)
	elapsed := time.Since(start)

	if !completed {
		t.Error("expected sleep to complete")
	}
	if elapsed < 50*time.Millisecond {
		t.Errorf("expected at least 50ms elapsed, got %v", elapsed)
	}
}

func TestSleepWithContext_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 10ms
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	completed := SleepWithContext(ctx, 1*time.Second)
	elapsed := time.Since(start)

	if completed {
		t.Error("expected sleep to be interrupted")
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("expected cancellation to happen quickly, took %v", elapsed)
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		statusCode int
		retryable  bool
	}{
		{200, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{429, true}, // Too Many Requests
		{500, true}, // Internal Server Error
		{502, true}, // Bad Gateway
		{503, true}, // Service Unavailable
		{504, true}, // Gateway Timeout
	}

	for _, tt := range tests {
		result := IsRetryableError(tt.statusCode)
		if result != tt.retryable {
			t.Errorf("status %d: expected retryable=%v, got %v", tt.statusCode, tt.retryable, result)
		}
	}
}
