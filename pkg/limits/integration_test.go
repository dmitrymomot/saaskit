package limits_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/limits"
)

// Integration tests that demonstrate real-world usage patterns
func TestIntegration_BasicWorkflow(t *testing.T) {
	t.Parallel()

	// Setup: Define plans
	plans := map[string]limits.Plan{
		"starter": {
			ID:   "starter",
			Name: "Starter Plan",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    10,
				limits.ResourceProjects: 5,
				"api_calls":             1000,
			},
			Features:  []limits.Feature{},
			TrialDays: 7,
		},
		"professional": {
			ID:   "professional",
			Name: "Professional Plan",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    50,
				limits.ResourceProjects: 20,
				"api_calls":             10000,
			},
			Features:  []limits.Feature{limits.FeatureAI, limits.FeatureSSO},
			TrialDays: 14,
		},
	}

	// Create source
	source := limits.NewInMemSource(plans)

	// Create counters with simulated database
	type tenantData struct {
		users    int64
		projects int64
		apiCalls int64
	}
	database := map[uuid.UUID]tenantData{
		uuid.MustParse("11111111-1111-1111-1111-111111111111"): {users: 5, projects: 3, apiCalls: 500},
		uuid.MustParse("22222222-2222-2222-2222-222222222222"): {users: 45, projects: 18, apiCalls: 8000},
	}

	counters := limits.NewRegistry()
	counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		if data, ok := database[tenantID]; ok {
			return data.users, nil
		}
		return 0, nil
	})
	counters.Register(limits.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		if data, ok := database[tenantID]; ok {
			return data.projects, nil
		}
		return 0, nil
	})
	counters.Register("api_calls", func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		if data, ok := database[tenantID]; ok {
			return data.apiCalls, nil
		}
		return 0, nil
	})

	// Create service
	svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
	require.NoError(t, err)

	t.Run("tenant on starter plan", func(t *testing.T) {
		t.Parallel()

		tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		ctx := limits.SetPlanIDToContext(context.Background(), "starter")

		// Check if can create more users
		err := svc.CanCreate(ctx, tenantID, limits.ResourceUsers)
		assert.NoError(t, err, "should be able to create users (5/10)")

		// Check usage
		used, limit, err := svc.GetUsage(ctx, tenantID, limits.ResourceUsers)
		require.NoError(t, err)
		assert.Equal(t, int64(5), used)
		assert.Equal(t, int64(10), limit)

		// Check percentage
		percentage := svc.GetUsagePercentage(ctx, tenantID, limits.ResourceUsers)
		assert.Equal(t, 50, percentage)

		// Check features
		assert.False(t, svc.HasFeature(ctx, tenantID, limits.FeatureAI))
		assert.False(t, svc.HasFeature(ctx, tenantID, limits.FeatureSSO))

		// Check trial
		err = svc.CheckTrial(ctx, tenantID, time.Now().AddDate(0, 0, -3))
		assert.NoError(t, err, "trial should be active (started 3 days ago)")
	})

	t.Run("tenant on professional plan near limits", func(t *testing.T) {
		t.Parallel()

		tenantID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		ctx := limits.SetPlanIDToContext(context.Background(), "professional")

		// Check if can create more users
		err := svc.CanCreate(ctx, tenantID, limits.ResourceUsers)
		assert.NoError(t, err, "should be able to create users (45/50)")

		// Check if can create more projects
		err = svc.CanCreate(ctx, tenantID, limits.ResourceProjects)
		assert.NoError(t, err, "should be able to create projects (18/20)")

		// Check percentage for API calls
		percentage := svc.GetUsagePercentage(ctx, tenantID, "api_calls")
		assert.Equal(t, 80, percentage, "8000/10000 = 80%")

		// Check features
		assert.True(t, svc.HasFeature(ctx, tenantID, limits.FeatureAI))
		assert.True(t, svc.HasFeature(ctx, tenantID, limits.FeatureSSO))

		// Check if can downgrade
		err = svc.CanDowngrade(ctx, tenantID, "starter")
		assert.Error(t, err, "cannot downgrade - usage exceeds starter limits")
		assert.ErrorIs(t, err, limits.ErrDowngradeNotPossible)
	})

	t.Run("get all usage", func(t *testing.T) {
		t.Parallel()

		tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		ctx := limits.SetPlanIDToContext(context.Background(), "starter")

		allUsage, err := svc.GetAllUsage(ctx, tenantID)
		require.NoError(t, err)

		assert.Len(t, allUsage, 3)
		assert.Equal(t, limits.UsageInfo{Current: 5, Limit: 10}, allUsage[limits.ResourceUsers])
		assert.Equal(t, limits.UsageInfo{Current: 3, Limit: 5}, allUsage[limits.ResourceProjects])
		assert.Equal(t, limits.UsageInfo{Current: 500, Limit: 1000}, allUsage["api_calls"])
	})
}

func TestIntegration_DynamicCounters(t *testing.T) {
	t.Parallel()

	// Setup: Simple plan
	plans := map[string]limits.Plan{
		"dynamic": {
			ID:   "dynamic",
			Name: "Dynamic Plan",
			Limits: map[limits.Resource]int64{
				"counter": 10,
			},
		},
	}

	source := limits.NewInMemSource(plans)

	// Create counter with atomic value
	var currentValue atomic.Int64
	currentValue.Store(5)

	counters := limits.NewRegistry()
	counters.Register("counter", func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		return currentValue.Load(), nil
	})

	svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
	require.NoError(t, err)

	ctx := limits.SetPlanIDToContext(context.Background(), "dynamic")
	tenantID := uuid.New()

	// Initial check - should be able to create
	err = svc.CanCreate(ctx, tenantID, "counter")
	assert.NoError(t, err)

	// Simulate resource creation
	currentValue.Add(5)

	// Check again - should be at limit
	err = svc.CanCreate(ctx, tenantID, "counter")
	assert.Error(t, err)
	assert.ErrorIs(t, err, limits.ErrLimitExceeded)

	// Simulate resource deletion
	currentValue.Add(-2)

	// Should be able to create again
	err = svc.CanCreate(ctx, tenantID, "counter")
	assert.NoError(t, err)
}

func TestIntegration_CustomPlanResolver(t *testing.T) {
	t.Parallel()

	// Setup: Plans
	plans := map[string]limits.Plan{
		"basic": {
			ID:   "basic",
			Name: "Basic Plan",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers: 5,
			},
		},
		"premium": {
			ID:   "premium",
			Name: "Premium Plan",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers: 100,
			},
			Features: []limits.Feature{limits.FeatureAI},
		},
	}

	source := limits.NewInMemSource(plans)
	counters := limits.NewRegistry()
	counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		return 10, nil
	})

	// Custom resolver that returns plan based on tenant ID
	customResolver := func(ctx context.Context, tenantID uuid.UUID) (string, error) {
		// Simple logic: even UUIDs get basic, odd get premium
		if tenantID[15]%2 == 0 {
			return "basic", nil
		}
		return "premium", nil
	}

	svc, err := limits.NewLimitsService(context.Background(), source, counters, customResolver)
	require.NoError(t, err)

	// Test with different tenant IDs
	ctx := context.Background() // No plan ID in context

	// Find a tenant ID that gets basic plan
	var basicTenantID uuid.UUID
	for range 10 {
		id := uuid.New()
		if id[15]%2 == 0 {
			basicTenantID = id
			break
		}
	}

	// Basic tenant - should hit limit
	err = svc.CanCreate(ctx, basicTenantID, limits.ResourceUsers)
	assert.Error(t, err)
	assert.ErrorIs(t, err, limits.ErrLimitExceeded)

	// Find a tenant ID that gets premium plan
	var premiumTenantID uuid.UUID
	for range 10 {
		id := uuid.New()
		if id[15]%2 == 1 {
			premiumTenantID = id
			break
		}
	}

	// Premium tenant - should be under limit
	err = svc.CanCreate(ctx, premiumTenantID, limits.ResourceUsers)
	assert.NoError(t, err)

	// Check features
	assert.False(t, svc.HasFeature(ctx, basicTenantID, limits.FeatureAI))
	assert.True(t, svc.HasFeature(ctx, premiumTenantID, limits.FeatureAI))
}

func TestIntegration_ErrorHandling(t *testing.T) {
	t.Parallel()

	// Setup with failing components
	plans := map[string]limits.Plan{
		"test": {
			ID:   "test",
			Name: "Test Plan",
			Limits: map[limits.Resource]int64{
				"stable":   10,
				"unstable": 10,
			},
		},
	}

	source := limits.NewInMemSource(plans)

	// Counters with different behaviors
	stableCount := int64(5)
	unstableError := errors.New("database connection lost")

	counters := limits.NewRegistry()
	counters.Register("stable", func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		return stableCount, nil
	})
	counters.Register("unstable", func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		return 0, unstableError
	})

	svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
	require.NoError(t, err)

	ctx := limits.SetPlanIDToContext(context.Background(), "test")
	tenantID := uuid.New()

	t.Run("stable resource works", func(t *testing.T) {
		t.Parallel()

		err := svc.CanCreate(ctx, tenantID, "stable")
		assert.NoError(t, err)

		used, limit, err := svc.GetUsage(ctx, tenantID, "stable")
		require.NoError(t, err)
		assert.Equal(t, stableCount, used)
		assert.Equal(t, int64(10), limit)
	})

	t.Run("unstable resource fails gracefully", func(t *testing.T) {
		t.Parallel()

		err := svc.CanCreate(ctx, tenantID, "unstable")
		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrFailedToCountResourceUsage)

		used, limit, err := svc.GetUsage(ctx, tenantID, "unstable")
		assert.Error(t, err)
		assert.Equal(t, int64(0), used)
		assert.Equal(t, int64(0), limit)

		// GetUsageSafe returns zeros without error
		used, limit = svc.GetUsageSafe(ctx, tenantID, "unstable")
		assert.Equal(t, int64(0), used)
		assert.Equal(t, int64(0), limit)
	})

	t.Run("get all usage includes both", func(t *testing.T) {
		t.Parallel()

		allUsage, err := svc.GetAllUsage(ctx, tenantID)
		require.NoError(t, err)

		// Should return data for both, with unstable showing 0
		assert.Len(t, allUsage, 2)
		assert.Equal(t, limits.UsageInfo{Current: 5, Limit: 10}, allUsage["stable"])
		assert.Equal(t, limits.UsageInfo{Current: 0, Limit: 10}, allUsage["unstable"])
	})
}

func TestIntegration_PlanMigration(t *testing.T) {
	t.Parallel()

	// Simulate a plan migration scenario
	plans := map[string]limits.Plan{
		"legacy": {
			ID:   "legacy",
			Name: "Legacy Plan",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    20,
				limits.ResourceProjects: 10,
				"old_feature":           100,
			},
			Features: []limits.Feature{"legacy_feature"},
		},
		"modern": {
			ID:   "modern",
			Name: "Modern Plan",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    30,
				limits.ResourceProjects: 5, // Less projects
				"new_feature":           200,
				// old_feature removed
			},
			Features: []limits.Feature{limits.FeatureAI, limits.FeatureSSO},
		},
	}

	source := limits.NewInMemSource(plans)

	// Current usage
	counters := limits.NewRegistry()
	counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		return 15, nil
	})
	counters.Register(limits.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		return 8, nil
	})
	counters.Register("old_feature", func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		return 50, nil
	})
	counters.Register("new_feature", func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		return 0, nil
	})

	svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
	require.NoError(t, err)

	tenantID := uuid.New()

	t.Run("compare plans", func(t *testing.T) {
		t.Parallel()

		legacy := plans["legacy"]
		modern := plans["modern"]
		comparison := limits.ComparePlans(&legacy, &modern)

		assert.Contains(t, comparison.NewFeatures, limits.FeatureAI)
		assert.Contains(t, comparison.NewFeatures, limits.FeatureSSO)
		assert.Contains(t, comparison.LostFeatures, limits.Feature("legacy_feature"))

		assert.Equal(t, limits.ResourceChange{From: 20, To: 30}, comparison.IncreasedLimits[limits.ResourceUsers])
		assert.Equal(t, limits.ResourceChange{From: 10, To: 5}, comparison.DecreasedLimits[limits.ResourceProjects])

		assert.Equal(t, int64(200), comparison.NewResources["new_feature"])
		assert.Equal(t, int64(100), comparison.RemovedResources["old_feature"])

		assert.True(t, comparison.HasResourceDecreases())
	})

	t.Run("migration validation", func(t *testing.T) {
		t.Parallel()

		ctx := limits.SetPlanIDToContext(context.Background(), "legacy")

		// Cannot downgrade due to projects usage (8 > 5)
		err := svc.CanDowngrade(ctx, tenantID, "modern")
		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrDowngradeNotPossible)
	})
}
