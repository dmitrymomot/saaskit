package limits

import (
	"slices"
	"time"
)

// Plan describes a subscription plan and its resource/feature constraints.
type Plan struct {
	ID          string
	Name        string
	Description string
	Limits      map[Resource]int64 // Resource limits
	Features    []Feature          // Feature flags enabled for this plan
	Public      bool               // If true, plan is available for self-registration
	TrialDays   int                // Number of trial days (0 disables trial)
}

// TrialEndsAt returns the timestamp when a trial period ends for this plan.
// If no trial is available, returns startedAt.
func (p Plan) TrialEndsAt(startedAt time.Time) time.Time {
	if p.TrialDays <= 0 {
		return startedAt
	}
	return startedAt.AddDate(0, 0, p.TrialDays).UTC()
}

// IsTrialActive reports whether the tenant is still in its trial window for this plan.
// Uses TrialEndsAt to avoid daylight-saving time issues.
func (p Plan) IsTrialActive(startedAt time.Time) bool {
	if p.TrialDays <= 0 {
		return false
	}
	return time.Now().UTC().Before(p.TrialEndsAt(startedAt))
}

// PlanComparison contains the differences between two plans.
type PlanComparison struct {
	// Features gained in the target plan
	NewFeatures []Feature
	// Features lost from the current plan
	LostFeatures []Feature
	// Resources with increased limits (old limit -> new limit)
	IncreasedLimits map[Resource]ResourceChange
	// Resources with decreased limits (old limit -> new limit)
	DecreasedLimits map[Resource]ResourceChange
	// Resources that exist only in the target plan
	NewResources map[Resource]int64
	// Resources that exist only in the current plan
	RemovedResources map[Resource]int64
}

// ResourceChange represents a change in resource limit.
type ResourceChange struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
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
			// New resource in target plan
			comparison.NewResources[resource] = targetLimit
			continue
		}

		// Both plans have this resource
		if targetLimit != currentLimit {
			change := ResourceChange{From: currentLimit, To: targetLimit}

			// Check if it's an increase or decrease
			// Treat unlimited (-1) specially
			if currentLimit == Unlimited {
				// Going from unlimited to limited is a decrease
				comparison.DecreasedLimits[resource] = change
			} else if targetLimit == Unlimited {
				// Going from limited to unlimited is an increase
				comparison.IncreasedLimits[resource] = change
			} else if targetLimit > currentLimit {
				// Normal increase
				comparison.IncreasedLimits[resource] = change
			} else {
				// Normal decrease
				comparison.DecreasedLimits[resource] = change
			}
		}
	}

	// Check for resources that exist in current but not in target
	for resource, currentLimit := range current.Limits {
		if _, exists := target.Limits[resource]; !exists {
			comparison.RemovedResources[resource] = currentLimit
		}
	}

	return comparison
}
