package subscription_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/svc/subscription"
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

	t.Run("rounds properly for values greater than 0.5", func(t *testing.T) {
		t.Parallel()
		// Fixed time for deterministic testing
		now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		// Trial ends in 1.6 days (38.4 hours)
		trialEndsAt := now.Add(38*time.Hour + 24*time.Minute)
		sub := &subscription.Subscription{
			TenantID:    uuid.New(),
			Status:      subscription.StatusTrialing,
			TrialEndsAt: &trialEndsAt,
		}

		days := sub.TrialDaysRemainingAt(now)
		// 38.4 hours = 1.6 days, + 0.5 = 2.1, int() = 2
		assert.Equal(t, 2, days)
	})

	t.Run("rounds down for values less than 0.5", func(t *testing.T) {
		t.Parallel()
		// Fixed time for deterministic testing
		now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		// Trial ends in 0.4 days (9.6 hours)
		trialEndsAt := now.Add(9*time.Hour + 36*time.Minute)
		sub := &subscription.Subscription{
			TenantID:    uuid.New(),
			Status:      subscription.StatusTrialing,
			TrialEndsAt: &trialEndsAt,
		}

		days := sub.TrialDaysRemainingAt(now)
		// 9.6 hours = 0.4 days, + 0.5 = 0.9, int() = 0
		assert.Equal(t, 0, days)
	})

	t.Run("returns exact days for whole numbers", func(t *testing.T) {
		t.Parallel()
		// Fixed time for deterministic testing
		now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		// Trial ends in exactly 7 days
		trialEndsAt := now.AddDate(0, 0, 7)
		sub := &subscription.Subscription{
			TenantID:    uuid.New(),
			Status:      subscription.StatusTrialing,
			TrialEndsAt: &trialEndsAt,
		}

		days := sub.TrialDaysRemainingAt(now)
		// Exactly 7 days
		assert.Equal(t, 7, days)
	})

	t.Run("handles edge case of almost expired trial", func(t *testing.T) {
		t.Parallel()
		// Fixed time for deterministic testing
		now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		// Trial ends in 30 minutes
		trialEndsAt := now.Add(30 * time.Minute)
		sub := &subscription.Subscription{
			TenantID:    uuid.New(),
			Status:      subscription.StatusTrialing,
			TrialEndsAt: &trialEndsAt,
		}

		days := sub.TrialDaysRemainingAt(now)
		// 30 minutes = 0.0208 days, + 0.5 = 0.5208, int() = 0
		assert.Equal(t, 0, days)
	})

	t.Run("handles rounding boundary at exactly 0.5 days", func(t *testing.T) {
		t.Parallel()
		// Fixed time for deterministic testing
		now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		// Trial ends in exactly 0.5 days (12 hours)
		trialEndsAt := now.Add(12 * time.Hour)
		sub := &subscription.Subscription{
			TenantID:    uuid.New(),
			Status:      subscription.StatusTrialing,
			TrialEndsAt: &trialEndsAt,
		}

		days := sub.TrialDaysRemainingAt(now)
		// 0.5 days + 0.5 rounding = 1.0 -> 1
		assert.Equal(t, 1, days)
	})

	t.Run("handles rounding for just under half day", func(t *testing.T) {
		t.Parallel()
		// Fixed time for deterministic testing
		now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		// Trial ends in 11 hours 59 minutes
		trialEndsAt := now.Add(11*time.Hour + 59*time.Minute)
		sub := &subscription.Subscription{
			TenantID:    uuid.New(),
			Status:      subscription.StatusTrialing,
			TrialEndsAt: &trialEndsAt,
		}

		days := sub.TrialDaysRemainingAt(now)
		// 11.983 hours = 0.499 days, + 0.5 = 0.999, int() = 0
		assert.Equal(t, 0, days)
	})
}
