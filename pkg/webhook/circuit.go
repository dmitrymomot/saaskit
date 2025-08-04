package webhook

import (
	"sync"
	"time"
)

// CircuitState represents the current state of the circuit breaker
type CircuitState int

const (
	// CircuitClosed allows requests to pass through
	CircuitClosed CircuitState = iota
	// CircuitOpen blocks all requests
	CircuitOpen
	// CircuitHalfOpen allows one request to test if the service has recovered
	CircuitHalfOpen
)

// String returns the string representation of the circuit state
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

// CircuitBreaker implements a simple circuit breaker pattern to prevent
// hammering failed endpoints. Safe for concurrent use.
type CircuitBreaker struct {
	mu sync.RWMutex

	failureThreshold int
	recoveryTimeout  time.Duration
	successThreshold int

	// Internal state tracking
	state           CircuitState
	failures        int
	lastFailureTime time.Time
	successCount    int // Tracks consecutive successes in half-open state
}

// NewCircuitBreaker creates a circuit breaker with the given configuration.
// Default values provide reasonable protection for most webhook scenarios.
func NewCircuitBreaker(failureThreshold, successThreshold int, recoveryTimeout time.Duration) *CircuitBreaker {
	// Conservative defaults protect against flapping while allowing quick recovery
	if failureThreshold <= 0 {
		failureThreshold = 5 // Open after 5 consecutive failures
	}
	if successThreshold <= 0 {
		successThreshold = 2 // Need 2 successes to close from half-open
	}
	if recoveryTimeout <= 0 {
		recoveryTimeout = 30 * time.Second // Wait 30s before testing recovery
	}

	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		recoveryTimeout:  recoveryTimeout,
		successThreshold: successThreshold,
		state:            CircuitClosed,
	}
}

// Allow checks if a request should be allowed through the circuit breaker.
// Uses a write lock since it may transition from open to half-open state.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		// Automatically transition to half-open after recovery timeout
		if time.Since(cb.lastFailureTime) > cb.recoveryTimeout {
			cb.state = CircuitHalfOpen
			cb.successCount = 0
			return true
		}
		return false

	case CircuitHalfOpen:
		// Allow request to test if service has recovered
		return true

	default:
		return false
	}
}

// RecordSuccess records a successful request and may close the circuit
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		// Reset failure counter to prevent gradual degradation
		cb.failures = 0

	case CircuitHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			// Service appears healthy, fully close the circuit
			cb.state = CircuitClosed
			cb.failures = 0
			cb.successCount = 0
		}
	}
}

// RecordFailure records a failed request and may open the circuit
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitClosed:
		cb.failures++
		if cb.failures >= cb.failureThreshold {
			// Threshold reached, protect the failing service
			cb.state = CircuitOpen
		}

	case CircuitHalfOpen:
		// Service still failing, immediately reopen circuit
		cb.state = CircuitOpen
		cb.failures = cb.failureThreshold
		cb.successCount = 0
	}
}

// State returns the current state, accounting for automatic transitions
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	// Show what the state would be if Allow() were called
	if cb.state == CircuitOpen && time.Since(cb.lastFailureTime) > cb.recoveryTimeout {
		return CircuitHalfOpen
	}

	return cb.state
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitClosed
	cb.failures = 0
	cb.successCount = 0
	cb.lastFailureTime = time.Time{}
}

// CircuitStats provides visibility into circuit breaker state for monitoring
type CircuitStats struct {
	State           string
	Failures        int
	SuccessCount    int
	LastFailureTime time.Time
}

// Stats returns the current statistics of the circuit breaker
func (cb *CircuitBreaker) Stats() CircuitStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitStats{
		State:           cb.state.String(),
		Failures:        cb.failures,
		SuccessCount:    cb.successCount,
		LastFailureTime: cb.lastFailureTime,
	}
}
