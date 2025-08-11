package ratelimiter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/ratelimiter"
)

func TestResult_RetryAfter(t *testing.T) {
	t.Parallel()

	t.Run("returns zero when allowed", func(t *testing.T) {
		result := ratelimiter.Result{
			Limit:     100,
			Remaining: 10,
			ResetAt:   time.Now().Add(time.Minute),
		}

		assert.Equal(t, time.Duration(0), result.RetryAfter())
	})

	t.Run("returns zero when exactly at limit", func(t *testing.T) {
		result := ratelimiter.Result{
			Limit:     100,
			Remaining: 0,
			ResetAt:   time.Now().Add(time.Minute),
		}

		assert.Equal(t, time.Duration(0), result.RetryAfter())
	})

	t.Run("returns positive duration when not allowed", func(t *testing.T) {
		future := time.Now().Add(30 * time.Second)
		result := ratelimiter.Result{
			Limit:     100,
			Remaining: -1,
			ResetAt:   future,
		}

		retryAfter := result.RetryAfter()
		assert.Greater(t, retryAfter, time.Duration(0))
		assert.LessOrEqual(t, retryAfter, 30*time.Second)
	})

	t.Run("returns zero for past reset time when not allowed", func(t *testing.T) {
		past := time.Now().Add(-time.Minute)
		result := ratelimiter.Result{
			Limit:     100,
			Remaining: -5,
			ResetAt:   past,
		}

		retryAfter := result.RetryAfter()
		assert.LessOrEqual(t, retryAfter, time.Duration(0))
	})

	t.Run("handles far future reset times", func(t *testing.T) {
		farFuture := time.Now().Add(24 * time.Hour)
		result := ratelimiter.Result{
			Limit:     100,
			Remaining: -10,
			ResetAt:   farFuture,
		}

		retryAfter := result.RetryAfter()
		assert.Greater(t, retryAfter, 23*time.Hour)
		assert.LessOrEqual(t, retryAfter, 24*time.Hour)
	})
}

func TestResult_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("zero limit with positive remaining", func(t *testing.T) {
		result := ratelimiter.Result{
			Limit:     0,
			Remaining: 5,
			ResetAt:   time.Now().Add(time.Minute),
		}

		assert.True(t, result.Allowed())
		assert.Equal(t, time.Duration(0), result.RetryAfter())
	})

	t.Run("zero limit with negative remaining", func(t *testing.T) {
		result := ratelimiter.Result{
			Limit:     0,
			Remaining: -1,
			ResetAt:   time.Now().Add(time.Minute),
		}

		assert.False(t, result.Allowed())
		assert.Greater(t, result.RetryAfter(), time.Duration(0))
	})

	t.Run("zero time reset", func(t *testing.T) {
		result := ratelimiter.Result{
			Limit:     100,
			Remaining: -1,
			ResetAt:   time.Time{},
		}

		assert.False(t, result.Allowed())
		retryAfter := result.RetryAfter()
		assert.Less(t, retryAfter, time.Duration(0))
	})
}
