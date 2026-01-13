package executor

import (
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond, 1)

	// Closed circuit should allow requests
	for i := 0; i < 10; i++ {
		if !cb.Allow() {
			t.Errorf("closed circuit should allow request %d", i)
		}
		cb.RecordSuccess()
	}

	if cb.State() != CircuitClosed {
		t.Errorf("expected CircuitClosed, got %v", cb.State())
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond, 1)

	// Record failures up to threshold
	for i := 0; i < 3; i++ {
		if !cb.Allow() {
			t.Errorf("circuit should allow request before threshold")
		}
		cb.RecordFailure()
	}

	// Circuit should now be open
	if cb.State() != CircuitOpen {
		t.Errorf("expected CircuitOpen after %d failures, got %v", 3, cb.State())
	}

	// Should not allow new requests
	if cb.Allow() {
		t.Error("open circuit should not allow requests")
	}
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond, 1)

	// Open the circuit
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Fatalf("expected CircuitOpen, got %v", cb.State())
	}

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open and allow one request
	if !cb.Allow() {
		t.Error("circuit should transition to half-open and allow request")
	}

	if cb.State() != CircuitHalfOpen {
		t.Errorf("expected CircuitHalfOpen, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenToClosedOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond, 1)

	// Open the circuit
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Transition to half-open
	cb.Allow()

	// Success should close the circuit
	cb.RecordSuccess()

	if cb.State() != CircuitClosed {
		t.Errorf("expected CircuitClosed after success in half-open, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenToOpenOnFailure(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond, 1)

	// Open the circuit
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Transition to half-open
	cb.Allow()

	// Failure should open the circuit again
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Errorf("expected CircuitOpen after failure in half-open, got %v", cb.State())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond, 1)

	// Open the circuit
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Fatalf("expected CircuitOpen, got %v", cb.State())
	}

	// Reset should close the circuit
	cb.Reset()

	if cb.State() != CircuitClosed {
		t.Errorf("expected CircuitClosed after reset, got %v", cb.State())
	}

	if !cb.Allow() {
		t.Error("reset circuit should allow requests")
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	cb := NewCircuitBreaker(5, 100*time.Millisecond, 1)

	cb.Allow()
	cb.RecordSuccess()
	cb.Allow()
	cb.RecordSuccess()
	cb.Allow()
	cb.RecordFailure()

	stats := cb.Stats()

	if stats.Successes != 2 {
		t.Errorf("expected 2 successes, got %d", stats.Successes)
	}
	if stats.Failures != 1 {
		t.Errorf("expected 1 failure, got %d", stats.Failures)
	}
	if stats.State != CircuitClosed {
		t.Errorf("expected CircuitClosed, got %v", stats.State)
	}
}

func TestEndpointCircuitBreakers_GetFirstAvailable(t *testing.T) {
	ecb := NewEndpointCircuitBreakers(CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     100 * time.Millisecond,
		HalfOpenMax:      1,
	})

	endpoints := []string{"endpoint1", "endpoint2", "endpoint3"}

	// Initially all should be available
	first := ecb.GetFirstAvailable(endpoints)
	if first != "endpoint1" {
		t.Errorf("expected endpoint1, got %s", first)
	}

	// Open circuit for endpoint1
	cb1 := ecb.Get("endpoint1")
	cb1.Allow()
	cb1.RecordFailure()
	cb1.Allow()
	cb1.RecordFailure()

	// Should now return endpoint2
	first = ecb.GetFirstAvailable(endpoints)
	if first != "endpoint2" {
		t.Errorf("expected endpoint2 after endpoint1 circuit open, got %s", first)
	}

	// Open circuit for endpoint2
	cb2 := ecb.Get("endpoint2")
	cb2.Allow()
	cb2.RecordFailure()
	cb2.Allow()
	cb2.RecordFailure()

	// Should now return endpoint3
	first = ecb.GetFirstAvailable(endpoints)
	if first != "endpoint3" {
		t.Errorf("expected endpoint3 after endpoint1 and endpoint2 circuits open, got %s", first)
	}
}

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{CircuitClosed, "closed"},
		{CircuitOpen, "open"},
		{CircuitHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		result := tt.state.String()
		if result != tt.expected {
			t.Errorf("CircuitState(%d).String() = %s, expected %s", tt.state, result, tt.expected)
		}
	}
}
