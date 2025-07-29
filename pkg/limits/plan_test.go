package limits_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/limits"
)

func TestPlan_TrialEndsAt(t *testing.T) {
	t.Parallel()

	t.Run("with trial days", func(t *testing.T) {
		t.Parallel()

		plan := limits.Plan{
			ID:        "pro",
			TrialDays: 14,
		}
		startedAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		expected := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

		result := plan.TrialEndsAt(startedAt)

		assert.Equal(t, expected, result)
	})

	t.Run("without trial days", func(t *testing.T) {
		t.Parallel()

		plan := limits.Plan{
			ID:        "free",
			TrialDays: 0,
		}
		startedAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		result := plan.TrialEndsAt(startedAt)

		assert.Equal(t, startedAt, result)
	})

	t.Run("with negative trial days", func(t *testing.T) {
		t.Parallel()

		plan := limits.Plan{
			ID:        "legacy",
			TrialDays: -5,
		}
		startedAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		result := plan.TrialEndsAt(startedAt)

		assert.Equal(t, startedAt, result)
	})
}

func TestPlan_IsTrialActive(t *testing.T) {
	t.Parallel()

	t.Run("active trial", func(t *testing.T) {
		t.Parallel()

		plan := limits.Plan{
			ID:        "pro",
			TrialDays: 14,
		}
		// Started yesterday
		startedAt := time.Now().UTC().AddDate(0, 0, -1)

		result := plan.IsTrialActive(startedAt)

		assert.True(t, result)
	})

	t.Run("expired trial", func(t *testing.T) {
		t.Parallel()

		plan := limits.Plan{
			ID:        "pro",
			TrialDays: 14,
		}
		// Started 15 days ago
		startedAt := time.Now().UTC().UTC().AddDate(0, 0, -15)

		result := plan.IsTrialActive(startedAt)

		assert.False(t, result)
	})

	t.Run("no trial", func(t *testing.T) {
		t.Parallel()

		plan := limits.Plan{
			ID:        "free",
			TrialDays: 0,
		}
		startedAt := time.Now().UTC()

		result := plan.IsTrialActive(startedAt)

		assert.False(t, result)
	})

	t.Run("negative trial days", func(t *testing.T) {
		t.Parallel()

		plan := limits.Plan{
			ID:        "legacy",
			TrialDays: -5,
		}
		startedAt := time.Now().UTC()

		result := plan.IsTrialActive(startedAt)

		assert.False(t, result)
	})

	t.Run("UTC time zone handling", func(t *testing.T) {
		t.Parallel()

		plan := limits.Plan{
			ID:        "pro",
			TrialDays: 1,
		}

		// Create a time in a different timezone
		loc, err := time.LoadLocation("America/New_York")
		require.NoError(t, err)

		// Start time 23 hours ago in New York time
		startedAt := time.Now().In(loc).Add(-23 * time.Hour)

		// Trial should still be active regardless of timezone
		result := plan.IsTrialActive(startedAt)
		assert.True(t, result)

		// Verify TrialEndsAt returns UTC
		trialEnd := plan.TrialEndsAt(startedAt)
		assert.Equal(t, time.UTC, trialEnd.Location())
	})
}

func TestComparePlans(t *testing.T) {
	t.Parallel()

	t.Run("nil plans", func(t *testing.T) {
		t.Parallel()

		result := limits.ComparePlans(nil, nil)
		assert.Nil(t, result)

		current := &limits.Plan{ID: "free"}
		result = limits.ComparePlans(current, nil)
		assert.Nil(t, result)

		result = limits.ComparePlans(nil, current)
		assert.Nil(t, result)
	})

	t.Run("feature changes", func(t *testing.T) {
		t.Parallel()

		current := &limits.Plan{
			ID:       "basic",
			Features: []limits.Feature{"feature1", "feature2"},
		}
		target := &limits.Plan{
			ID:       "pro",
			Features: []limits.Feature{"feature2", "feature3", "feature4"},
		}

		result := limits.ComparePlans(current, target)

		require.NotNil(t, result)
		assert.ElementsMatch(t, []limits.Feature{"feature3", "feature4"}, result.NewFeatures)
		assert.ElementsMatch(t, []limits.Feature{"feature1"}, result.LostFeatures)
	})

	t.Run("resource limit increases", func(t *testing.T) {
		t.Parallel()

		current := &limits.Plan{
			ID: "basic",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    10,
				limits.ResourceProjects: 5,
			},
		}
		target := &limits.Plan{
			ID: "pro",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    50,
				limits.ResourceProjects: 20,
			},
		}

		result := limits.ComparePlans(current, target)

		require.NotNil(t, result)
		assert.Len(t, result.IncreasedLimits, 2)
		assert.Equal(t, limits.ResourceChange{From: 10, To: 50}, result.IncreasedLimits[limits.ResourceUsers])
		assert.Equal(t, limits.ResourceChange{From: 5, To: 20}, result.IncreasedLimits[limits.ResourceProjects])
		assert.Empty(t, result.DecreasedLimits)
	})

	t.Run("resource limit decreases", func(t *testing.T) {
		t.Parallel()

		current := &limits.Plan{
			ID: "pro",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    50,
				limits.ResourceProjects: 20,
			},
		}
		target := &limits.Plan{
			ID: "basic",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    10,
				limits.ResourceProjects: 5,
			},
		}

		result := limits.ComparePlans(current, target)

		require.NotNil(t, result)
		assert.Empty(t, result.IncreasedLimits)
		assert.Len(t, result.DecreasedLimits, 2)
		assert.Equal(t, limits.ResourceChange{From: 50, To: 10}, result.DecreasedLimits[limits.ResourceUsers])
		assert.Equal(t, limits.ResourceChange{From: 20, To: 5}, result.DecreasedLimits[limits.ResourceProjects])
		assert.True(t, result.HasResourceDecreases())
	})

	t.Run("unlimited to limited", func(t *testing.T) {
		t.Parallel()

		current := &limits.Plan{
			ID: "enterprise",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    limits.Unlimited,
				limits.ResourceProjects: limits.Unlimited,
			},
		}
		target := &limits.Plan{
			ID: "pro",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    100,
				limits.ResourceProjects: 50,
			},
		}

		result := limits.ComparePlans(current, target)

		require.NotNil(t, result)
		assert.Empty(t, result.IncreasedLimits)
		assert.Len(t, result.DecreasedLimits, 2)
		assert.Equal(t, limits.ResourceChange{From: limits.Unlimited, To: 100}, result.DecreasedLimits[limits.ResourceUsers])
		assert.True(t, result.HasResourceDecreases())
	})

	t.Run("limited to unlimited", func(t *testing.T) {
		t.Parallel()

		current := &limits.Plan{
			ID: "pro",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    100,
				limits.ResourceProjects: 50,
			},
		}
		target := &limits.Plan{
			ID: "enterprise",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    limits.Unlimited,
				limits.ResourceProjects: limits.Unlimited,
			},
		}

		result := limits.ComparePlans(current, target)

		require.NotNil(t, result)
		assert.Len(t, result.IncreasedLimits, 2)
		assert.Equal(t, limits.ResourceChange{From: 100, To: limits.Unlimited}, result.IncreasedLimits[limits.ResourceUsers])
		assert.Empty(t, result.DecreasedLimits)
		assert.False(t, result.HasResourceDecreases())
	})

	t.Run("new and removed resources", func(t *testing.T) {
		t.Parallel()

		current := &limits.Plan{
			ID: "basic",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers: 10,
				"old_resource":       5,
			},
		}
		target := &limits.Plan{
			ID: "pro",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    10, // Same limit
				limits.ResourceProjects: 20,
			},
		}

		result := limits.ComparePlans(current, target)

		require.NotNil(t, result)
		assert.Equal(t, int64(20), result.NewResources[limits.ResourceProjects])
		assert.Equal(t, int64(5), result.RemovedResources["old_resource"])
		assert.Empty(t, result.IncreasedLimits)
		assert.Empty(t, result.DecreasedLimits)
		assert.True(t, result.HasResourceDecreases()) // Because of removed resource
	})

	t.Run("mixed changes", func(t *testing.T) {
		t.Parallel()

		current := &limits.Plan{
			ID: "custom1",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    50,
				limits.ResourceProjects: 10,
				"storage":               100,
				"api_calls":             1000,
			},
			Features: []limits.Feature{"feature1", "feature2"},
		}
		target := &limits.Plan{
			ID: "custom2",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    100, // Increased
				limits.ResourceProjects: 5,   // Decreased
				"storage":               100, // Same
				"new_resource":          50,  // New
				// api_calls removed
			},
			Features: []limits.Feature{"feature2", "feature3"}, // feature1 lost, feature3 gained
		}

		result := limits.ComparePlans(current, target)

		require.NotNil(t, result)

		// Check features
		assert.ElementsMatch(t, []limits.Feature{"feature3"}, result.NewFeatures)
		assert.ElementsMatch(t, []limits.Feature{"feature1"}, result.LostFeatures)

		// Check resources
		assert.Len(t, result.IncreasedLimits, 1)
		assert.Equal(t, limits.ResourceChange{From: 50, To: 100}, result.IncreasedLimits[limits.ResourceUsers])

		assert.Len(t, result.DecreasedLimits, 1)
		assert.Equal(t, limits.ResourceChange{From: 10, To: 5}, result.DecreasedLimits[limits.ResourceProjects])

		assert.Equal(t, int64(50), result.NewResources["new_resource"])
		assert.Equal(t, int64(1000), result.RemovedResources["api_calls"])

		assert.True(t, result.HasResourceDecreases())
	})

	t.Run("no changes", func(t *testing.T) {
		t.Parallel()

		plan := &limits.Plan{
			ID: "basic",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    10,
				limits.ResourceProjects: 5,
			},
			Features: []limits.Feature{"feature1", "feature2"},
		}

		result := limits.ComparePlans(plan, plan)

		require.NotNil(t, result)
		assert.Empty(t, result.NewFeatures)
		assert.Empty(t, result.LostFeatures)
		assert.Empty(t, result.IncreasedLimits)
		assert.Empty(t, result.DecreasedLimits)
		assert.Empty(t, result.NewResources)
		assert.Empty(t, result.RemovedResources)
		assert.False(t, result.HasResourceDecreases())
	})
}
