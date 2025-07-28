package limits_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/limits"
)

func TestNewInMemSource(t *testing.T) {
	t.Parallel()

	t.Run("creates independent copy", func(t *testing.T) {
		t.Parallel()

		originalPlans := map[string]limits.Plan{
			"free": {
				ID:   "free",
				Name: "Free Plan",
				Limits: map[limits.Resource]int64{
					limits.ResourceUsers: 5,
				},
				Features: []limits.Feature{limits.FeatureAI},
			},
		}

		source := limits.NewInMemSource(originalPlans)

		// Modify original plans
		freePlan := originalPlans["free"]
		freePlan.Name = "Modified Name"
		freePlan.Limits[limits.ResourceUsers] = 100
		freePlan.Features[0] = limits.FeatureSSO
		originalPlans["free"] = freePlan

		// Load plans from source
		loadedPlans, err := source.Load(context.Background())
		require.NoError(t, err)

		// Verify source has independent copy
		assert.Equal(t, "Free Plan", loadedPlans["free"].Name)
		assert.Equal(t, int64(5), loadedPlans["free"].Limits[limits.ResourceUsers])
		assert.Equal(t, limits.FeatureAI, loadedPlans["free"].Features[0])
	})

	t.Run("handles empty plans", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(map[string]limits.Plan{})

		loadedPlans, err := source.Load(context.Background())

		require.NoError(t, err)
		assert.Empty(t, loadedPlans)
	})

	t.Run("handles nil plans", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(nil)

		loadedPlans, err := source.Load(context.Background())

		require.NoError(t, err)
		assert.Empty(t, loadedPlans)
	})

	t.Run("deep copies all plan fields", func(t *testing.T) {
		t.Parallel()

		plans := map[string]limits.Plan{
			"pro": {
				ID:          "pro",
				Name:        "Pro Plan",
				Description: "Professional features",
				Limits: map[limits.Resource]int64{
					limits.ResourceUsers:    50,
					limits.ResourceProjects: 20,
				},
				Features:  []limits.Feature{limits.FeatureAI, limits.FeatureSSO},
				Public:    true,
				TrialDays: 14,
			},
		}

		source := limits.NewInMemSource(plans)
		loadedPlans, err := source.Load(context.Background())
		require.NoError(t, err)

		proPlan := loadedPlans["pro"]
		assert.Equal(t, "pro", proPlan.ID)
		assert.Equal(t, "Pro Plan", proPlan.Name)
		assert.Equal(t, "Professional features", proPlan.Description)
		assert.Equal(t, int64(50), proPlan.Limits[limits.ResourceUsers])
		assert.Equal(t, int64(20), proPlan.Limits[limits.ResourceProjects])
		assert.ElementsMatch(t, []limits.Feature{limits.FeatureAI, limits.FeatureSSO}, proPlan.Features)
		assert.True(t, proPlan.Public)
		assert.Equal(t, 14, proPlan.TrialDays)
	})
}

func TestInMemSource_Load(t *testing.T) {
	t.Parallel()

	t.Run("returns independent copy on each load", func(t *testing.T) {
		t.Parallel()

		plans := map[string]limits.Plan{
			"basic": {
				ID:   "basic",
				Name: "Basic Plan",
				Limits: map[limits.Resource]int64{
					limits.ResourceUsers: 10,
				},
				Features: []limits.Feature{limits.FeatureAI},
			},
		}

		source := limits.NewInMemSource(plans)

		// Load first copy
		plans1, err := source.Load(context.Background())
		require.NoError(t, err)

		// Modify first copy
		basicPlan := plans1["basic"]
		basicPlan.Name = "Modified Basic"
		basicPlan.Limits[limits.ResourceUsers] = 999
		basicPlan.Features[0] = limits.FeatureSSO
		plans1["basic"] = basicPlan

		// Load second copy
		plans2, err := source.Load(context.Background())
		require.NoError(t, err)

		// Verify second copy is unmodified
		assert.Equal(t, "Basic Plan", plans2["basic"].Name)
		assert.Equal(t, int64(10), plans2["basic"].Limits[limits.ResourceUsers])
		assert.Equal(t, limits.FeatureAI, plans2["basic"].Features[0])
	})

	t.Run("multiple plans", func(t *testing.T) {
		t.Parallel()

		plans := map[string]limits.Plan{
			"free": {
				ID:   "free",
				Name: "Free Plan",
				Limits: map[limits.Resource]int64{
					limits.ResourceUsers: 5,
				},
			},
			"pro": {
				ID:   "pro",
				Name: "Pro Plan",
				Limits: map[limits.Resource]int64{
					limits.ResourceUsers: limits.Unlimited,
				},
			},
			"enterprise": {
				ID:   "enterprise",
				Name: "Enterprise Plan",
				Limits: map[limits.Resource]int64{
					limits.ResourceUsers:    limits.Unlimited,
					limits.ResourceProjects: limits.Unlimited,
				},
			},
		}

		source := limits.NewInMemSource(plans)
		loadedPlans, err := source.Load(context.Background())

		require.NoError(t, err)
		assert.Len(t, loadedPlans, 3)
		assert.Contains(t, loadedPlans, "free")
		assert.Contains(t, loadedPlans, "pro")
		assert.Contains(t, loadedPlans, "enterprise")
	})

	t.Run("concurrent loads", func(t *testing.T) {
		t.Parallel()

		plans := map[string]limits.Plan{
			"concurrent": {
				ID:   "concurrent",
				Name: "Concurrent Plan",
				Limits: map[limits.Resource]int64{
					limits.ResourceUsers: 100,
				},
				Features: []limits.Feature{limits.FeatureAI},
			},
		}

		source := limits.NewInMemSource(plans)

		// Run multiple concurrent loads
		var wg sync.WaitGroup
		iterations := 100
		wg.Add(iterations)

		errors := make([]error, iterations)
		results := make([]map[string]limits.Plan, iterations)

		for i := range iterations {
			go func(index int) {
				defer wg.Done()
				loaded, err := source.Load(context.Background())
				errors[index] = err
				results[index] = loaded
			}(i)
		}

		wg.Wait()

		// Verify all loads succeeded
		for i := range iterations {
			require.NoError(t, errors[i])
			require.NotNil(t, results[i])
			assert.Equal(t, "Concurrent Plan", results[i]["concurrent"].Name)
			assert.Equal(t, int64(100), results[i]["concurrent"].Limits[limits.ResourceUsers])
		}
	})

	t.Run("handles context cancellation gracefully", func(t *testing.T) {
		t.Parallel()

		plans := map[string]limits.Plan{
			"test": {
				ID:   "test",
				Name: "Test Plan",
			},
		}

		source := limits.NewInMemSource(plans)

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Load should still work with cancelled context
		loadedPlans, err := source.Load(ctx)

		require.NoError(t, err)
		assert.Len(t, loadedPlans, 1)
		assert.Equal(t, "Test Plan", loadedPlans["test"].Name)
	})

	t.Run("empty features and limits", func(t *testing.T) {
		t.Parallel()

		plans := map[string]limits.Plan{
			"minimal": {
				ID:       "minimal",
				Name:     "Minimal Plan",
				Limits:   nil,
				Features: nil,
			},
		}

		source := limits.NewInMemSource(plans)
		loadedPlans, err := source.Load(context.Background())

		require.NoError(t, err)
		minimalPlan := loadedPlans["minimal"]
		// maps.Clone and slices.Clone return nil for nil inputs
		assert.Nil(t, minimalPlan.Limits)
		assert.Nil(t, minimalPlan.Features)
	})
}
