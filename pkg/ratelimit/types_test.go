package ratelimit_test

import (
	"testing"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/ratelimit"

	"github.com/stretchr/testify/assert"
)

func TestResult_RetryAfter(t *testing.T) {
	t.Parallel()

	t.Run("allowed request returns zero", func(t *testing.T) {
		t.Parallel()
		result := ratelimit.Result{
			Allowed: true,
			ResetAt: time.Now().Add(10 * time.Second),
		}
		duration := result.RetryAfter()
		assert.Equal(t, time.Duration(0), duration)
	})

	t.Run("denied request with future reset", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		result := ratelimit.Result{
			Allowed: false,
			ResetAt: now.Add(5 * time.Second),
		}
		duration := result.RetryAfter()
		// Check that duration is approximately 5 seconds
		assert.InDelta(t, 5.0, duration.Seconds(), 0.5)
	})

	t.Run("denied request with past reset", func(t *testing.T) {
		t.Parallel()
		result := ratelimit.Result{
			Allowed: false,
			ResetAt: time.Now().Add(-1 * time.Second),
		}
		duration := result.RetryAfter()
		// Should return a negative or zero duration
		assert.LessOrEqual(t, duration, time.Duration(0))
	})
}

func TestResult_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("zero values", func(t *testing.T) {
		t.Parallel()
		var r ratelimit.Result
		assert.Equal(t, 0, r.Limit)
		assert.Equal(t, 0, r.Remaining)
		assert.False(t, r.Allowed)
		assert.True(t, r.ResetAt.IsZero())
		// Zero ResetAt with denied request returns negative duration (time until zero time)
		// This is expected behavior - just verify it doesn't panic
		_ = r.RetryAfter()
	})

	t.Run("negative remaining", func(t *testing.T) {
		t.Parallel()
		r := ratelimit.Result{
			Allowed:   false,
			Limit:     10,
			Remaining: -5,
			ResetAt:   time.Now().Add(time.Second),
		}
		assert.Less(t, r.Remaining, 0)
		assert.Greater(t, r.RetryAfter(), time.Duration(0))
	})
}
