package webhook_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/webhook"
)

func TestExponentialBackoff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		backoff  webhook.ExponentialBackoff
		attempts []int
		want     []time.Duration
	}{
		{
			name: "default values",
			backoff: webhook.ExponentialBackoff{
				JitterFactor: 0, // Disable jitter for predictable testing
			},
			attempts: []int{1, 2, 3, 4, 5},
			want: []time.Duration{
				time.Second,      // 1s * 2^0
				2 * time.Second,  // 1s * 2^1
				4 * time.Second,  // 1s * 2^2
				8 * time.Second,  // 1s * 2^3
				16 * time.Second, // 1s * 2^4
			},
		},
		{
			name: "custom values with max cap",
			backoff: webhook.ExponentialBackoff{
				InitialInterval: 500 * time.Millisecond,
				MaxInterval:     5 * time.Second,
				Multiplier:      3,
				JitterFactor:    0, // No jitter for predictable testing
			},
			attempts: []int{1, 2, 3, 4},
			want: []time.Duration{
				500 * time.Millisecond,  // 500ms * 3^0
				1500 * time.Millisecond, // 500ms * 3^1
				4500 * time.Millisecond, // 500ms * 3^2
				5 * time.Second,         // Capped at max
			},
		},
		{
			name:     "zero attempt returns zero",
			backoff:  webhook.ExponentialBackoff{},
			attempts: []int{0, -1},
			want:     []time.Duration{0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, len(tt.attempts), len(tt.want), "test setup error")

			for i, attempt := range tt.attempts {
				got := tt.backoff.NextInterval(attempt)
				assert.Equal(t, tt.want[i], got, "attempt %d", attempt)
			}
		})
	}
}

func TestExponentialBackoffJitter(t *testing.T) {
	t.Parallel()

	backoff := webhook.ExponentialBackoff{
		InitialInterval: time.Second,
		JitterFactor:    0.5, // 50% jitter
	}

	// Run multiple times to test jitter
	intervals := make([]time.Duration, 10)
	for i := range 10 {
		intervals[i] = backoff.NextInterval(3) // 3rd attempt = 4s base
	}

	// All intervals should be different due to jitter
	seen := make(map[time.Duration]bool)
	for _, interval := range intervals {
		// Should be within 4s ± 50% = 2s to 6s
		assert.GreaterOrEqual(t, interval, 2*time.Second)
		assert.LessOrEqual(t, interval, 6*time.Second)
		seen[interval] = true
	}

	// With 10 samples and high jitter, we should see variety
	assert.Greater(t, len(seen), 5, "expected more variety with jitter")
}

func TestLinearBackoff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		backoff  webhook.LinearBackoff
		attempts []int
		want     []time.Duration
	}{
		{
			name:     "default values",
			backoff:  webhook.LinearBackoff{},
			attempts: []int{1, 2, 3, 4, 5},
			want: []time.Duration{
				time.Second,     // 1s * 1
				2 * time.Second, // 1s * 2
				3 * time.Second, // 1s * 3
				4 * time.Second, // 1s * 4
				5 * time.Second, // 1s * 5
			},
		},
		{
			name: "custom values with max cap",
			backoff: webhook.LinearBackoff{
				Interval:    500 * time.Millisecond,
				MaxInterval: 2 * time.Second,
			},
			attempts: []int{1, 2, 3, 4, 5},
			want: []time.Duration{
				500 * time.Millisecond,  // 500ms * 1
				1000 * time.Millisecond, // 500ms * 2
				1500 * time.Millisecond, // 500ms * 3
				2 * time.Second,         // Capped at max
				2 * time.Second,         // Capped at max
			},
		},
		{
			name:     "zero attempt returns zero",
			backoff:  webhook.LinearBackoff{},
			attempts: []int{0, -1},
			want:     []time.Duration{0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, len(tt.attempts), len(tt.want), "test setup error")

			for i, attempt := range tt.attempts {
				got := tt.backoff.NextInterval(attempt)
				assert.Equal(t, tt.want[i], got, "attempt %d", attempt)
			}
		})
	}
}

func TestFixedBackoff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		backoff  webhook.FixedBackoff
		attempts []int
		want     time.Duration
	}{
		{
			name:     "custom interval",
			backoff:  webhook.FixedBackoff{Interval: 2 * time.Second},
			attempts: []int{1, 2, 3, 10, 100},
			want:     2 * time.Second,
		},
		{
			name:     "zero interval",
			backoff:  webhook.FixedBackoff{Interval: 0},
			attempts: []int{1, 2, 3},
			want:     0,
		},
		{
			name:     "zero attempt returns zero",
			backoff:  webhook.FixedBackoff{Interval: time.Second},
			attempts: []int{0, -1},
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for _, attempt := range tt.attempts {
				got := tt.backoff.NextInterval(attempt)
				if attempt <= 0 {
					assert.Equal(t, time.Duration(0), got, "attempt %d", attempt)
				} else {
					assert.Equal(t, tt.want, got, "attempt %d", attempt)
				}
			}
		})
	}
}

func TestDefaultBackoffStrategy(t *testing.T) {
	t.Parallel()

	strategy := webhook.DefaultBackoffStrategy()

	// Should be exponential backoff
	eb, ok := strategy.(webhook.ExponentialBackoff)
	require.True(t, ok, "default should be ExponentialBackoff")

	// Check default values
	assert.Equal(t, time.Second, eb.InitialInterval)
	assert.Equal(t, 30*time.Second, eb.MaxInterval)
	assert.Equal(t, float64(2), eb.Multiplier)
	assert.Equal(t, 0.1, eb.JitterFactor)
}

func BenchmarkExponentialBackoff(b *testing.B) {
	backoff := webhook.ExponentialBackoff{
		InitialInterval: time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2,
		JitterFactor:    0.1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = backoff.NextInterval(i%10 + 1)
	}
}

func BenchmarkLinearBackoff(b *testing.B) {
	backoff := webhook.LinearBackoff{
		Interval:    time.Second,
		MaxInterval: 30 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = backoff.NextInterval(i%10 + 1)
	}
}
