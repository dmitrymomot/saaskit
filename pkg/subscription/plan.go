package subscription

import (
	"slices"
	"time"
)

// Plan describes a subscription plan and its resource/feature constraints.
// The ID field should be set to the payment provider's price ID for paid plans
// to enable direct mapping during checkout and webhook processing.
type Plan struct {
	ID          string // provider's price ID (e.g., price_starter_monthly)
	Name        string
	Description string
	Limits      map[Resource]int64 // -1 represents unlimited
	Features    []Feature
	Public      bool // available for self-service signup
	TrialDays   int
	Price       Money
	Interval    BillingInterval
}

// TrialEndsAt calculates when the trial period ends.
// Returns startedAt unchanged if no trial is available.
func (p Plan) TrialEndsAt(startedAt time.Time) time.Time {
	if p.TrialDays <= 0 {
		return startedAt
	}
	return startedAt.AddDate(0, 0, p.TrialDays).UTC()
}

// IsTrialActive reports whether the tenant is still in its trial window.
func (p Plan) IsTrialActive(startedAt time.Time) bool {
	if p.TrialDays <= 0 {
		return false
	}
	return time.Now().UTC().Before(p.TrialEndsAt(startedAt))
}

// PlanComparison contains the differences between two plans.
// Used to validate downgrades and communicate changes to users.
type PlanComparison struct {
	NewFeatures      []Feature
	LostFeatures     []Feature
	IncreasedLimits  map[Resource]ResourceChange
	DecreasedLimits  map[Resource]ResourceChange
	NewResources     map[Resource]int64
	RemovedResources map[Resource]int64
}

// ResourceChange represents a change in resource limit.
type ResourceChange struct {
	From int64
	To   int64
}

// HasResourceDecreases returns true if any resources have decreased limits.
func (c *PlanComparison) HasResourceDecreases() bool {
	return len(c.DecreasedLimits) > 0 || len(c.RemovedResources) > 0
}

// ComparePlans returns the differences between current and target plans.
func ComparePlans(current, target *Plan) *PlanComparison {
	if current == nil || target == nil {
		return nil
	}

	comparison := &PlanComparison{
		NewFeatures:      make([]Feature, 0),
		LostFeatures:     make([]Feature, 0),
		IncreasedLimits:  make(map[Resource]ResourceChange),
		DecreasedLimits:  make(map[Resource]ResourceChange),
		NewResources:     make(map[Resource]int64),
		RemovedResources: make(map[Resource]int64),
	}

	// Compare features
	for _, feature := range target.Features {
		if !slices.Contains(current.Features, feature) {
			comparison.NewFeatures = append(comparison.NewFeatures, feature)
		}
	}

	for _, feature := range current.Features {
		if !slices.Contains(target.Features, feature) {
			comparison.LostFeatures = append(comparison.LostFeatures, feature)
		}
	}

	// Compare resource limits
	for resource, targetLimit := range target.Limits {
		currentLimit, exists := current.Limits[resource]
		if !exists {
			comparison.NewResources[resource] = targetLimit
			continue
		}

		if targetLimit != currentLimit {
			change := ResourceChange{From: currentLimit, To: targetLimit}

			// Treat unlimited-to-limited as decrease to prevent accidental loss of unlimited access
			if currentLimit == Unlimited {
				comparison.DecreasedLimits[resource] = change
			} else if targetLimit == Unlimited {
				comparison.IncreasedLimits[resource] = change
			} else if targetLimit > currentLimit {
				comparison.IncreasedLimits[resource] = change
			} else {
				comparison.DecreasedLimits[resource] = change
			}
		}
	}

	for resource, currentLimit := range current.Limits {
		if _, exists := target.Limits[resource]; !exists {
			comparison.RemovedResources[resource] = currentLimit
		}
	}

	return comparison
}
