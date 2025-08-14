package subscription_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/svc/subscription"
)

func TestComparePlans(t *testing.T) {
	t.Parallel()

	t.Run("identifies new features", func(t *testing.T) {
		t.Parallel()
		current := &subscription.Plan{
			ID: "basic",
			Features: []subscription.Feature{
				subscription.FeatureAPI,
			},
		}
		target := &subscription.Plan{
			ID: "pro",
			Features: []subscription.Feature{
				subscription.FeatureAPI,
				subscription.FeatureSSO,
				subscription.FeatureWebhooks,
			},
		}

		comparison := subscription.ComparePlans(current, target)
		assert.Len(t, comparison.NewFeatures, 2)
		assert.Contains(t, comparison.NewFeatures, subscription.FeatureSSO)
		assert.Contains(t, comparison.NewFeatures, subscription.FeatureWebhooks)
		assert.Empty(t, comparison.LostFeatures)
	})

	t.Run("identifies lost features", func(t *testing.T) {
		t.Parallel()
		current := &subscription.Plan{
			ID: "pro",
			Features: []subscription.Feature{
				subscription.FeatureAPI,
				subscription.FeatureSSO,
				subscription.FeatureWebhooks,
			},
		}
		target := &subscription.Plan{
			ID: "basic",
			Features: []subscription.Feature{
				subscription.FeatureAPI,
			},
		}

		comparison := subscription.ComparePlans(current, target)
		assert.Empty(t, comparison.NewFeatures)
		assert.Len(t, comparison.LostFeatures, 2)
		assert.Contains(t, comparison.LostFeatures, subscription.FeatureSSO)
		assert.Contains(t, comparison.LostFeatures, subscription.FeatureWebhooks)
	})

	t.Run("identifies increased limits", func(t *testing.T) {
		t.Parallel()
		current := &subscription.Plan{
			ID: "basic",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects:    10,
				subscription.ResourceTeamMembers: 5,
			},
		}
		target := &subscription.Plan{
			ID: "pro",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects:    50,
				subscription.ResourceTeamMembers: 20,
			},
		}

		comparison := subscription.ComparePlans(current, target)
		assert.Len(t, comparison.IncreasedLimits, 2)
		assert.Equal(t, int64(10), comparison.IncreasedLimits[subscription.ResourceProjects].From)
		assert.Equal(t, int64(50), comparison.IncreasedLimits[subscription.ResourceProjects].To)
		assert.Empty(t, comparison.DecreasedLimits)
	})

	t.Run("identifies decreased limits", func(t *testing.T) {
		t.Parallel()
		current := &subscription.Plan{
			ID: "pro",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects:    50,
				subscription.ResourceTeamMembers: 20,
			},
		}
		target := &subscription.Plan{
			ID: "basic",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects:    10,
				subscription.ResourceTeamMembers: 5,
			},
		}

		comparison := subscription.ComparePlans(current, target)
		assert.Empty(t, comparison.IncreasedLimits)
		assert.Len(t, comparison.DecreasedLimits, 2)
		assert.Equal(t, int64(50), comparison.DecreasedLimits[subscription.ResourceProjects].From)
		assert.Equal(t, int64(10), comparison.DecreasedLimits[subscription.ResourceProjects].To)
	})

	t.Run("treats unlimited to limited as decrease", func(t *testing.T) {
		t.Parallel()
		current := &subscription.Plan{
			ID: "pro",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceTeamMembers: subscription.Unlimited,
			},
		}
		target := &subscription.Plan{
			ID: "basic",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceTeamMembers: 5,
			},
		}

		comparison := subscription.ComparePlans(current, target)
		assert.Empty(t, comparison.IncreasedLimits)
		assert.Len(t, comparison.DecreasedLimits, 1)
		assert.Equal(t, subscription.Unlimited, comparison.DecreasedLimits[subscription.ResourceTeamMembers].From)
		assert.Equal(t, int64(5), comparison.DecreasedLimits[subscription.ResourceTeamMembers].To)
	})

	t.Run("treats limited to unlimited as increase", func(t *testing.T) {
		t.Parallel()
		current := &subscription.Plan{
			ID: "basic",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceTeamMembers: 5,
			},
		}
		target := &subscription.Plan{
			ID: "pro",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceTeamMembers: subscription.Unlimited,
			},
		}

		comparison := subscription.ComparePlans(current, target)
		assert.Len(t, comparison.IncreasedLimits, 1)
		assert.Equal(t, int64(5), comparison.IncreasedLimits[subscription.ResourceTeamMembers].From)
		assert.Equal(t, subscription.Unlimited, comparison.IncreasedLimits[subscription.ResourceTeamMembers].To)
		assert.Empty(t, comparison.DecreasedLimits)
	})

	t.Run("identifies new resources", func(t *testing.T) {
		t.Parallel()
		current := &subscription.Plan{
			ID: "basic",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects: 10,
			},
		}
		target := &subscription.Plan{
			ID: "pro",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects: 50,
				subscription.ResourceWebhooks: 5,
			},
		}

		comparison := subscription.ComparePlans(current, target)
		assert.Len(t, comparison.NewResources, 1)
		assert.Equal(t, int64(5), comparison.NewResources[subscription.ResourceWebhooks])
		assert.Empty(t, comparison.RemovedResources)
	})

	t.Run("identifies removed resources", func(t *testing.T) {
		t.Parallel()
		current := &subscription.Plan{
			ID: "pro",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects: 50,
				subscription.ResourceWebhooks: 5,
			},
		}
		target := &subscription.Plan{
			ID: "basic",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects: 10,
			},
		}

		comparison := subscription.ComparePlans(current, target)
		assert.Empty(t, comparison.NewResources)
		assert.Len(t, comparison.RemovedResources, 1)
		assert.Equal(t, int64(5), comparison.RemovedResources[subscription.ResourceWebhooks])
	})

	t.Run("HasResourceDecreases returns true when limits decreased", func(t *testing.T) {
		t.Parallel()
		current := &subscription.Plan{
			ID: "pro",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects: 50,
			},
		}
		target := &subscription.Plan{
			ID: "basic",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects: 10,
			},
		}

		comparison := subscription.ComparePlans(current, target)
		assert.True(t, comparison.HasResourceDecreases())
	})

	t.Run("HasResourceDecreases returns true when resources removed", func(t *testing.T) {
		t.Parallel()
		current := &subscription.Plan{
			ID: "pro",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects: 50,
				subscription.ResourceWebhooks: 5,
			},
		}
		target := &subscription.Plan{
			ID: "basic",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects: 50, // Same limit
			},
		}

		comparison := subscription.ComparePlans(current, target)
		assert.True(t, comparison.HasResourceDecreases())
	})

	t.Run("HasResourceDecreases returns false when only increases", func(t *testing.T) {
		t.Parallel()
		current := &subscription.Plan{
			ID: "basic",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects: 10,
			},
		}
		target := &subscription.Plan{
			ID: "pro",
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects: 50,
				subscription.ResourceWebhooks: 5,
			},
		}

		comparison := subscription.ComparePlans(current, target)
		assert.False(t, comparison.HasResourceDecreases())
	})

	t.Run("returns nil when current plan is nil", func(t *testing.T) {
		t.Parallel()
		target := &subscription.Plan{ID: "basic"}
		comparison := subscription.ComparePlans(nil, target)
		assert.Nil(t, comparison)
	})

	t.Run("returns nil when target plan is nil", func(t *testing.T) {
		t.Parallel()
		current := &subscription.Plan{ID: "basic"}
		comparison := subscription.ComparePlans(current, nil)
		assert.Nil(t, comparison)
	})
}
