package subscription

import (
	"time"

	"github.com/google/uuid"
)

// Subscription represents a tenant's subscription to a plan.
// Each tenant has exactly one active subscription at a time.
type Subscription struct {
	TenantID      uuid.UUID // Primary key - one subscription per tenant
	PlanID        string
	Status        SubscriptionStatus
	ProviderSubID string // Provider's subscription ID (empty for free plans)
	CreatedAt     time.Time
	TrialEndsAt   *time.Time // Set only for plans with trials
	UpdatedAt     time.Time
	CancelledAt   *time.Time // Set when subscription is cancelled
}

// IsTrialing returns true if the subscription is in trial status.
func (s *Subscription) IsTrialing() bool {
	return s.Status == StatusTrialing
}

// IsActive returns true if the subscription is active (paid).
func (s *Subscription) IsActive() bool {
	return s.Status == StatusActive
}

// IsCancelled returns true if the subscription is cancelled.
func (s *Subscription) IsCancelled() bool {
	return s.Status == StatusCancelled
}

// IsTrialExpired returns true if the trial period has ended.
func (s *Subscription) IsTrialExpired() bool {
	if s.TrialEndsAt == nil {
		return false
	}
	return time.Now().UTC().After(*s.TrialEndsAt)
}

// TrialDaysRemainingAt returns the number of days remaining in the trial at a given time.
// Returns 0 if not in trial or trial has expired.
// This method is useful for testing with fixed time values.
func (s *Subscription) TrialDaysRemainingAt(now time.Time) int {
	if !s.IsTrialing() || s.TrialEndsAt == nil {
		return 0
	}

	remaining := s.TrialEndsAt.Sub(now)
	if remaining <= 0 {
		return 0
	}

	// Round up partial days to be user-friendly
	days := remaining.Hours() / 24
	return int(days + 0.5)
}

// TrialDaysRemaining returns the number of days remaining in the trial.
// Returns 0 if not in trial or trial has expired.
func (s *Subscription) TrialDaysRemaining() int {
	return s.TrialDaysRemainingAt(time.Now().UTC())
}
