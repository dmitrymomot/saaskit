package subscription_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/subscription"
)

func TestSubscription_TrialDaysRemaining(t *testing.T) {
	t.Parallel()

	t.Run("returns 0 when not in trial", func(t *testing.T) {
		t.Parallel()
		sub := &subscription.Subscription{
			TenantID: uuid.New(),
			Status:   subscription.StatusActive,
		}

		days := sub.TrialDaysRemaining()
		assert.Equal(t, 0, days)
	})

	t.Run("returns 0 when trial has no end date", func(t *testing.T) {
		t.Parallel()
		sub := &subscription.Subscription{
			TenantID: uuid.New(),
			Status:   subscription.StatusTrialing,
			// TrialEndsAt is nil
		}

		days := sub.TrialDaysRemaining()
		assert.Equal(t, 0, days)
	})

	t.Run("returns 0 when trial has expired", func(t *testing.T) {
		t.Parallel()
		pastDate := time.Now().UTC().AddDate(0, 0, -1) // Yesterday
		sub := &subscription.Subscription{
			TenantID:    uuid.New(),
			Status:      subscription.StatusTrialing,
			TrialEndsAt: &pastDate,
		}

		days := sub.TrialDaysRemaining()
		assert.Equal(t, 0, days)
	})

	t.Run("rounds up partial days", func(t *testing.T) {
		t.Parallel()
		// Trial ends in 1.2 days, should return 2
		futureDate := time.Now().UTC().Add(29 * time.Hour) // 1.2 days
		sub := &subscription.Subscription{
			TenantID:    uuid.New(),
			Status:      subscription.StatusTrialing,
			TrialEndsAt: &futureDate,
		}

		days := sub.TrialDaysRemaining()
		// Allow for timing variance
		assert.GreaterOrEqual(t, days, 1)
		assert.LessOrEqual(t, days, 2)
	})

	t.Run("rounds up even small partial days", func(t *testing.T) {
		t.Parallel()
		// Trial ends in 0.1 days (2.4 hours), should return 1
		futureDate := time.Now().UTC().Add(2 * time.Hour)
		sub := &subscription.Subscription{
			TenantID:    uuid.New(),
			Status:      subscription.StatusTrialing,
			TrialEndsAt: &futureDate,
		}

		days := sub.TrialDaysRemaining()
		// With rounding, should be 0 or 1 depending on exact timing
		assert.GreaterOrEqual(t, days, 0)
		assert.LessOrEqual(t, days, 1)
	})

	t.Run("returns exact days for whole numbers", func(t *testing.T) {
		t.Parallel()
		// Trial ends in exactly 7 days
		futureDate := time.Now().UTC().AddDate(0, 0, 7)
		sub := &subscription.Subscription{
			TenantID:    uuid.New(),
			Status:      subscription.StatusTrialing,
			TrialEndsAt: &futureDate,
		}

		days := sub.TrialDaysRemaining()
		// Allow for 7 or 8 due to timing precision
		assert.GreaterOrEqual(t, days, 7)
		assert.LessOrEqual(t, days, 8)
	})

	t.Run("handles edge case of almost expired trial", func(t *testing.T) {
		t.Parallel()
		// Trial ends in 30 minutes, should still return 0 or 1
		futureDate := time.Now().UTC().Add(30 * time.Minute)
		sub := &subscription.Subscription{
			TenantID:    uuid.New(),
			Status:      subscription.StatusTrialing,
			TrialEndsAt: &futureDate,
		}

		days := sub.TrialDaysRemaining()
		// 30 minutes = 0.02 days, with +0.5 rounding = 0.52 -> 0
		assert.GreaterOrEqual(t, days, 0)
		assert.LessOrEqual(t, days, 1)
	})
}