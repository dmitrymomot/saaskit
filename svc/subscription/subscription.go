package subscription

import (
	"time"

	"github.com/google/uuid"
)

// Subscription represents a tenant's subscription to a plan.
// Each tenant has exactly one active subscription at a time.
type Subscription struct {
	TenantID           uuid.UUID // primary key - one subscription per tenant
	PlanID             string
	Status             SubscriptionStatus
	ProviderSubID      string // provider's subscription ID (empty for free plans)
	ProviderCustomerID string // provider's customer ID (ctm_xxx, cus_xxx, etc)
	CreatedAt          time.Time
	TrialEndsAt        *time.Time // set only for plans with trials
	UpdatedAt          time.Time
	CancelledAt        *time.Time // set when subscription is cancelled
}

func (s *Subscription) IsTrialing() bool {
	return s.Status == StatusTrialing
}

func (s *Subscription) IsActive() bool {
	return s.Status == StatusActive
}

func (s *Subscription) IsCancelled() bool {
	return s.Status == StatusCancelled
}

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

	// Round up partial days for better UX
	days := remaining.Hours() / 24
	return int(days + 0.5)
}

// TrialDaysRemaining returns the number of days remaining in the trial.
// Returns 0 if not in trial or trial has expired.
func (s *Subscription) TrialDaysRemaining() int {
	return s.TrialDaysRemainingAt(time.Now().UTC())
}
