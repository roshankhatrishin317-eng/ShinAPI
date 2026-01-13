// Package executor provides runtime execution capabilities for various AI service providers.
// This file implements a circuit breaker pattern for intelligent rate limiting and failover.
package executor

import (
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	// CircuitClosed indicates the circuit is functioning normally.
	CircuitClosed CircuitState = iota
	// CircuitOpen indicates the circuit has tripped due to failures.
	CircuitOpen
	// CircuitHalfOpen indicates the circuit is testing if it can close again.
	CircuitHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for handling failures gracefully.
// It tracks failures for a specific endpoint and opens the circuit when failures exceed
// the threshold, preventing further requests until the reset timeout has elapsed.
type CircuitBreaker struct {
	mu sync.RWMutex

	// Configuration
	FailureThreshold int           // Number of failures before opening circuit
	ResetTimeout     time.Duration // Time to wait before attempting to close circuit
	HalfOpenMax      int           // Max requests allowed in half-open state

	// State
	state          CircuitState
	failures       int
	successes      int
	lastFailure    time.Time
	halfOpenCount  int
}

// NewCircuitBreaker creates a new circuit breaker with the specified configuration.
func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration, halfOpenMax int) *CircuitBreaker {
	if failureThreshold <= 0 {
		failureThreshold = 5
	}
	if resetTimeout <= 0 {
		resetTimeout = 30 * time.Second
	}
	if halfOpenMax <= 0 {
		halfOpenMax = 1
	}
	return &CircuitBreaker{
		FailureThreshold: failureThreshold,
		ResetTimeout:     resetTimeout,
		HalfOpenMax:      halfOpenMax,
		state:            CircuitClosed,
	}
}

// DefaultCircuitBreaker returns a circuit breaker with sensible defaults.
func DefaultCircuitBreaker() *CircuitBreaker {
	return NewCircuitBreaker(5, 30*time.Second, 1)
}

// Allow checks if a request should be allowed through the circuit breaker.
// Returns true if the request can proceed, false if the circuit is open.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		// Check if reset timeout has elapsed
		if time.Since(cb.lastFailure) >= cb.ResetTimeout {
			cb.state = CircuitHalfOpen
			cb.halfOpenCount = 0
			return true
		}
		return false

	case CircuitHalfOpen:
		// Allow limited requests in half-open state
		if cb.halfOpenCount < cb.HalfOpenMax {
			cb.halfOpenCount++
			return true
		}
		return false

	default:
		return true
	}
}

// RecordSuccess records a successful request.
// In half-open state, success closes the circuit.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successes++
	if cb.state == CircuitHalfOpen {
		// Success in half-open state closes the circuit
		cb.state = CircuitClosed
		cb.failures = 0
		cb.halfOpenCount = 0
	}
}

// RecordFailure records a failed request.
// If failures exceed the threshold, the circuit opens.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	switch cb.state {
	case CircuitClosed:
		if cb.failures >= cb.FailureThreshold {
			cb.state = CircuitOpen
		}

	case CircuitHalfOpen:
		// Any failure in half-open state opens the circuit again
		cb.state = CircuitOpen
		cb.halfOpenCount = 0
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = CircuitClosed
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenCount = 0
}

// Stats returns circuit breaker statistics.
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return CircuitBreakerStats{
		State:       cb.state,
		Failures:    cb.failures,
		Successes:   cb.successes,
		LastFailure: cb.lastFailure,
	}
}

// CircuitBreakerStats holds statistics about a circuit breaker.
type CircuitBreakerStats struct {
	State       CircuitState
	Failures    int
	Successes   int
	LastFailure time.Time
}

// String returns a human-readable state name.
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// EndpointCircuitBreakers manages circuit breakers for multiple endpoints.
type EndpointCircuitBreakers struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	config   CircuitBreakerConfig
}

// CircuitBreakerConfig holds configuration for circuit breakers.
type CircuitBreakerConfig struct {
	FailureThreshold int
	ResetTimeout     time.Duration
	HalfOpenMax      int
}

// DefaultCircuitBreakerConfig returns default configuration.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		ResetTimeout:     30 * time.Second,
		HalfOpenMax:      1,
	}
}

// NewEndpointCircuitBreakers creates a new endpoint circuit breaker manager.
func NewEndpointCircuitBreakers(config CircuitBreakerConfig) *EndpointCircuitBreakers {
	return &EndpointCircuitBreakers{
		breakers: make(map[string]*CircuitBreaker),
		config:   config,
	}
}

// Get returns the circuit breaker for the specified endpoint, creating one if needed.
func (e *EndpointCircuitBreakers) Get(endpoint string) *CircuitBreaker {
	e.mu.RLock()
	cb, exists := e.breakers[endpoint]
	e.mu.RUnlock()

	if exists {
		return cb
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists = e.breakers[endpoint]; exists {
		return cb
	}

	cb = NewCircuitBreaker(e.config.FailureThreshold, e.config.ResetTimeout, e.config.HalfOpenMax)
	e.breakers[endpoint] = cb
	return cb
}

// GetFirstAvailable returns the first endpoint whose circuit breaker allows requests.
// Returns empty string if all circuits are open.
func (e *EndpointCircuitBreakers) GetFirstAvailable(endpoints []string) string {
	for _, endpoint := range endpoints {
		if e.Get(endpoint).Allow() {
			return endpoint
		}
	}
	return ""
}

// ResetAll resets all circuit breakers.
func (e *EndpointCircuitBreakers) ResetAll() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, cb := range e.breakers {
		cb.Reset()
	}
}
