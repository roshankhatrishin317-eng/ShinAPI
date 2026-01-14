// Package circuitbreaker implements the circuit breaker pattern for handling failures gracefully.
// It provides protection against cascading failures by tracking endpoint health and temporarily
// blocking requests to failing services.
package circuitbreaker

import (
	"sync"
	"time"
)

// State represents the state of a circuit breaker.
type State int

const (
	// Closed indicates the circuit is functioning normally.
	Closed State = iota
	// Open indicates the circuit has tripped due to failures.
	Open
	// HalfOpen indicates the circuit is testing if it can close again.
	HalfOpen
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
	state         State
	failures      int
	successes     int
	lastFailure   time.Time
	halfOpenCount int
}

// New creates a new circuit breaker with the specified configuration.
func New(failureThreshold int, resetTimeout time.Duration, halfOpenMax int) *CircuitBreaker {
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
		state:            Closed,
	}
}

// Default returns a circuit breaker with sensible defaults.
func Default() *CircuitBreaker {
	return New(5, 30*time.Second, 1)
}

// Allow checks if a request should be allowed through the circuit breaker.
// Returns true if the request can proceed, false if the circuit is open.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case Closed:
		return true

	case Open:
		// Check if reset timeout has elapsed
		if time.Since(cb.lastFailure) >= cb.ResetTimeout {
			cb.state = HalfOpen
			cb.halfOpenCount = 0
			return true
		}
		return false

	case HalfOpen:
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
	if cb.state == HalfOpen {
		// Success in half-open state closes the circuit
		cb.state = Closed
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
	case Closed:
		if cb.failures >= cb.FailureThreshold {
			cb.state = Open
		}

	case HalfOpen:
		// Any failure in half-open state opens the circuit again
		cb.state = Open
		cb.halfOpenCount = 0
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = Closed
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenCount = 0
}

// Stats returns circuit breaker statistics.
func (cb *CircuitBreaker) Stats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return Stats{
		State:       cb.state,
		Failures:    cb.failures,
		Successes:   cb.successes,
		LastFailure: cb.lastFailure,
	}
}

// Stats holds statistics about a circuit breaker.
type Stats struct {
	State       State
	Failures    int
	Successes   int
	LastFailure time.Time
}

// String returns a human-readable state name.
func (s State) String() string {
	switch s {
	case Closed:
		return "closed"
	case Open:
		return "open"
	case HalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config holds configuration for circuit breakers.
type Config struct {
	FailureThreshold int
	ResetTimeout     time.Duration
	HalfOpenMax      int
}

// DefaultConfig returns default configuration.
func DefaultConfig() Config {
	return Config{
		FailureThreshold: 5,
		ResetTimeout:     30 * time.Second,
		HalfOpenMax:      1,
	}
}

// EndpointBreakers manages circuit breakers for multiple endpoints.
type EndpointBreakers struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	config   Config
}

// NewEndpointBreakers creates a new endpoint circuit breaker manager.
func NewEndpointBreakers(config Config) *EndpointBreakers {
	return &EndpointBreakers{
		breakers: make(map[string]*CircuitBreaker),
		config:   config,
	}
}

// Get returns the circuit breaker for the specified endpoint, creating one if needed.
func (e *EndpointBreakers) Get(endpoint string) *CircuitBreaker {
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

	cb = New(e.config.FailureThreshold, e.config.ResetTimeout, e.config.HalfOpenMax)
	e.breakers[endpoint] = cb
	return cb
}

// GetFirstAvailable returns the first endpoint whose circuit breaker allows requests.
// Returns empty string if all circuits are open.
func (e *EndpointBreakers) GetFirstAvailable(endpoints []string) string {
	for _, endpoint := range endpoints {
		if e.Get(endpoint).Allow() {
			return endpoint
		}
	}
	return ""
}

// ResetAll resets all circuit breakers.
func (e *EndpointBreakers) ResetAll() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, cb := range e.breakers {
		cb.Reset()
	}
}
