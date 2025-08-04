package webhook_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/webhook"
)

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	t.Parallel()

	t.Run("Closed to Open", func(t *testing.T) {
		t.Parallel()

		cb := webhook.NewCircuitBreaker(2, 1, 100*time.Millisecond)

		// Initially closed
		assert.Equal(t, webhook.CircuitClosed, cb.State())
		assert.True(t, cb.Allow())

		// First failure - still closed
		cb.RecordFailure()
		assert.Equal(t, webhook.CircuitClosed, cb.State())
		assert.True(t, cb.Allow())

		// Second failure - should open
		cb.RecordFailure()
		assert.Equal(t, webhook.CircuitOpen, cb.State())
		assert.False(t, cb.Allow())
	})

	t.Run("Open to HalfOpen", func(t *testing.T) {
		t.Parallel()

		cb := webhook.NewCircuitBreaker(1, 1, 50*time.Millisecond)

		// Open the circuit
		cb.RecordFailure()
		assert.Equal(t, webhook.CircuitOpen, cb.State())
		assert.False(t, cb.Allow())

		// Wait for recovery timeout
		time.Sleep(60 * time.Millisecond)

		// Should transition to half-open on next Allow() call
		assert.True(t, cb.Allow())
		assert.Equal(t, webhook.CircuitHalfOpen, cb.State())
	})

	t.Run("HalfOpen to Closed", func(t *testing.T) {
		t.Parallel()

		cb := webhook.NewCircuitBreaker(1, 2, 50*time.Millisecond)

		// Open the circuit
		cb.RecordFailure()
		time.Sleep(60 * time.Millisecond)

		// Transition to half-open
		assert.True(t, cb.Allow())
		assert.Equal(t, webhook.CircuitHalfOpen, cb.State())

		// First success - still half-open
		cb.RecordSuccess()
		assert.Equal(t, webhook.CircuitHalfOpen, cb.State())

		// Second success - should close
		cb.RecordSuccess()
		assert.Equal(t, webhook.CircuitClosed, cb.State())
	})

	t.Run("HalfOpen to Open", func(t *testing.T) {
		t.Parallel()

		cb := webhook.NewCircuitBreaker(1, 2, 50*time.Millisecond)

		// Open the circuit
		cb.RecordFailure()
		time.Sleep(60 * time.Millisecond)

		// Transition to half-open
		assert.True(t, cb.Allow())
		assert.Equal(t, webhook.CircuitHalfOpen, cb.State())

		// Failure in half-open - should reopen
		cb.RecordFailure()
		assert.Equal(t, webhook.CircuitOpen, cb.State())
		assert.False(t, cb.Allow())
	})
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	cb := webhook.NewCircuitBreaker(10, 2, 100*time.Millisecond)

	const numGoroutines = 100
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Run concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Mix of operations
				switch j % 4 {
				case 0:
					cb.Allow()
				case 1:
					cb.RecordSuccess()
				case 2:
					cb.RecordFailure()
				case 3:
					cb.State()
				}
			}
		}(i)
	}

	wg.Wait()

	// Circuit should be in a valid state after concurrent access
	state := cb.State()
	assert.Contains(t, []webhook.CircuitState{
		webhook.CircuitClosed,
		webhook.CircuitOpen,
		webhook.CircuitHalfOpen,
	}, state)

	// Stats should be accessible
	stats := cb.Stats()
	assert.Contains(t, []string{"closed", "open", "half-open"}, stats.State)
	assert.GreaterOrEqual(t, stats.Failures, 0)
	assert.GreaterOrEqual(t, stats.SuccessCount, 0)
}

func TestCircuitBreaker_RecoveryTimeout(t *testing.T) {
	t.Parallel()

	t.Run("Respect Recovery Timeout", func(t *testing.T) {
		t.Parallel()

		cb := webhook.NewCircuitBreaker(1, 1, 100*time.Millisecond)

		// Open the circuit
		cb.RecordFailure()
		assert.Equal(t, webhook.CircuitOpen, cb.State())
		assert.False(t, cb.Allow())

		// Too early - still open
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, webhook.CircuitOpen, cb.State())
		assert.False(t, cb.Allow())

		// After timeout - should allow and be half-open
		time.Sleep(60 * time.Millisecond)
		assert.Equal(t, webhook.CircuitHalfOpen, cb.State()) // State() checks timeout
		assert.True(t, cb.Allow())
	})

	t.Run("Multiple Failures Reset Timeout", func(t *testing.T) {
		t.Parallel()

		cb := webhook.NewCircuitBreaker(1, 1, 100*time.Millisecond)

		// Open the circuit
		cb.RecordFailure()
		originalTime := time.Now()

		// Wait almost until timeout
		time.Sleep(80 * time.Millisecond)

		// Another failure should reset the timeout
		cb.RecordFailure()

		// Original timeout should not apply
		time.Sleep(30 * time.Millisecond) // Total 110ms from original
		assert.Equal(t, webhook.CircuitOpen, cb.State())
		assert.False(t, cb.Allow())

		// Need to wait full timeout from last failure
		time.Sleep(80 * time.Millisecond) // 110ms from last failure
		assert.Equal(t, webhook.CircuitHalfOpen, cb.State())
		assert.True(t, cb.Allow())

		_ = originalTime // Use variable to avoid lint error
	})
}

func TestCircuitBreaker_HalfOpenSuccess(t *testing.T) {
	t.Parallel()

	t.Run("Single Success Required", func(t *testing.T) {
		t.Parallel()

		cb := webhook.NewCircuitBreaker(1, 1, 50*time.Millisecond)

		// Open and transition to half-open
		cb.RecordFailure()
		time.Sleep(60 * time.Millisecond)
		cb.Allow()

		assert.Equal(t, webhook.CircuitHalfOpen, cb.State())

		// One success should close it
		cb.RecordSuccess()
		assert.Equal(t, webhook.CircuitClosed, cb.State())
	})

	t.Run("Multiple Successes Required", func(t *testing.T) {
		t.Parallel()

		cb := webhook.NewCircuitBreaker(1, 3, 50*time.Millisecond)

		// Open and transition to half-open
		cb.RecordFailure()
		time.Sleep(60 * time.Millisecond)
		cb.Allow()

		assert.Equal(t, webhook.CircuitHalfOpen, cb.State())

		// First two successes - still half-open
		cb.RecordSuccess()
		assert.Equal(t, webhook.CircuitHalfOpen, cb.State())

		cb.RecordSuccess()
		assert.Equal(t, webhook.CircuitHalfOpen, cb.State())

		// Third success should close it
		cb.RecordSuccess()
		assert.Equal(t, webhook.CircuitClosed, cb.State())
	})

	t.Run("Allow Multiple Requests in HalfOpen", func(t *testing.T) {
		t.Parallel()

		cb := webhook.NewCircuitBreaker(1, 2, 50*time.Millisecond)

		// Open and transition to half-open
		cb.RecordFailure()
		time.Sleep(60 * time.Millisecond)
		cb.Allow()

		// Multiple Allow() calls should return true in half-open
		assert.True(t, cb.Allow())
		assert.True(t, cb.Allow())
		assert.True(t, cb.Allow())
	})
}

func TestCircuitBreaker_ResetAndStats(t *testing.T) {
	t.Parallel()

	t.Run("Reset Functionality", func(t *testing.T) {
		t.Parallel()

		cb := webhook.NewCircuitBreaker(2, 1, 100*time.Millisecond)

		// Create some state
		cb.RecordFailure()
		cb.RecordFailure() // Opens circuit
		assert.Equal(t, webhook.CircuitOpen, cb.State())

		// Reset should restore to initial state
		cb.Reset()
		assert.Equal(t, webhook.CircuitClosed, cb.State())
		assert.True(t, cb.Allow())

		stats := cb.Stats()
		assert.Equal(t, "closed", stats.State)
		assert.Equal(t, 0, stats.Failures)
		assert.Equal(t, 0, stats.SuccessCount)
		assert.True(t, stats.LastFailureTime.IsZero())
	})

	t.Run("Stats Accuracy", func(t *testing.T) {
		t.Parallel()

		cb := webhook.NewCircuitBreaker(3, 2, 100*time.Millisecond)

		// Initial stats
		stats := cb.Stats()
		assert.Equal(t, "closed", stats.State)
		assert.Equal(t, 0, stats.Failures)
		assert.Equal(t, 0, stats.SuccessCount)

		// Record some failures
		beforeFailure := time.Now()
		cb.RecordFailure()
		cb.RecordFailure()

		stats = cb.Stats()
		assert.Equal(t, "closed", stats.State)
		assert.Equal(t, 2, stats.Failures)
		assert.Equal(t, 0, stats.SuccessCount)
		assert.True(t, stats.LastFailureTime.After(beforeFailure))

		// Open circuit
		cb.RecordFailure()
		stats = cb.Stats()
		assert.Equal(t, "open", stats.State)
		assert.Equal(t, 3, stats.Failures)

		// Transition to half-open and record success
		time.Sleep(110 * time.Millisecond)
		cb.Allow()
		cb.RecordSuccess()

		stats = cb.Stats()
		assert.Equal(t, "half-open", stats.State)
		assert.Equal(t, 3, stats.Failures)
		assert.Equal(t, 1, stats.SuccessCount)
	})

	t.Run("Success Resets Failures in Closed State", func(t *testing.T) {
		t.Parallel()

		cb := webhook.NewCircuitBreaker(5, 1, 100*time.Millisecond)

		// Accumulate some failures
		cb.RecordFailure()
		cb.RecordFailure()

		stats := cb.Stats()
		assert.Equal(t, 2, stats.Failures)

		// Success should reset failure count
		cb.RecordSuccess()
		stats = cb.Stats()
		assert.Equal(t, 0, stats.Failures)
		assert.Equal(t, "closed", stats.State)
	})
}

func TestCircuitBreaker_DefaultValues(t *testing.T) {
	t.Parallel()

	t.Run("Zero Values Get Defaults", func(t *testing.T) {
		t.Parallel()

		cb := webhook.NewCircuitBreaker(0, 0, 0)

		// Should use defaults and be functional
		assert.Equal(t, webhook.CircuitClosed, cb.State())
		assert.True(t, cb.Allow())

		// Test default failure threshold (5)
		for i := 0; i < 4; i++ {
			cb.RecordFailure()
			assert.Equal(t, webhook.CircuitClosed, cb.State())
		}

		// 5th failure should open
		cb.RecordFailure()
		assert.Equal(t, webhook.CircuitOpen, cb.State())
	})

	t.Run("Negative Values Get Defaults", func(t *testing.T) {
		t.Parallel()

		cb := webhook.NewCircuitBreaker(-1, -1, -1*time.Second)

		// Should still be functional with defaults
		assert.Equal(t, webhook.CircuitClosed, cb.State())
		assert.True(t, cb.Allow())
	})

	t.Run("Partial Zero Values", func(t *testing.T) {
		t.Parallel()

		// Only failure threshold specified
		cb1 := webhook.NewCircuitBreaker(3, 0, 0)

		// Test that it works with custom failure threshold
		for i := 0; i < 2; i++ {
			cb1.RecordFailure()
			assert.Equal(t, webhook.CircuitClosed, cb1.State())
		}

		// 3rd failure should open (custom threshold)
		cb1.RecordFailure()
		assert.Equal(t, webhook.CircuitOpen, cb1.State())

		// Wait for default recovery timeout (30s is too long for test)
		// Just verify the state transition logic
		time.Sleep(time.Millisecond) // Minimal wait

		// Only timeout specified
		cb2 := webhook.NewCircuitBreaker(0, 0, 50*time.Millisecond)
		cb2.RecordFailure() // Use default threshold of 5

		// Should be closed until reaching default threshold
		assert.Equal(t, webhook.CircuitClosed, cb2.State())
	})

	t.Run("String Representation", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "closed", webhook.CircuitClosed.String())
		assert.Equal(t, "open", webhook.CircuitOpen.String())
		assert.Equal(t, "half-open", webhook.CircuitHalfOpen.String())

		// Test invalid state
		invalidState := webhook.CircuitState(999)
		assert.Equal(t, "unknown", invalidState.String())
	})
}
