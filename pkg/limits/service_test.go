package limits_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/limits"
)

// Test helpers
func createTestPlans() map[string]limits.Plan {
	return map[string]limits.Plan{
		"free": {
			ID:   "free",
			Name: "Free Plan",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    5,
				limits.ResourceProjects: 3,
			},
			Features:  []limits.Feature{},
			TrialDays: 0,
		},
		"pro": {
			ID:   "pro",
			Name: "Pro Plan",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    50,
				limits.ResourceProjects: 20,
			},
			Features:  []limits.Feature{limits.FeatureAI, limits.FeatureSSO},
			TrialDays: 14,
		},
		"enterprise": {
			ID:   "enterprise",
			Name: "Enterprise Plan",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    limits.Unlimited,
				limits.ResourceProjects: limits.Unlimited,
			},
			Features:  []limits.Feature{limits.FeatureAI, limits.FeatureSSO},
			TrialDays: 30,
		},
	}
}

func createTestCounters() limits.CounterRegistry {
	registry := limits.NewRegistry()

	// Default counters returning static values
	registry.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		return 3, nil
	})
	registry.Register(limits.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		return 2, nil
	})

	return registry
}

func TestNewLimitsService(t *testing.T) {
	t.Parallel()

	t.Run("successful creation", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)

		require.NoError(t, err)
		assert.NotNil(t, svc)
	})

	t.Run("with custom plan resolver", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		customResolver := func(ctx context.Context, tenantID uuid.UUID) (string, error) {
			return "pro", nil
		}

		svc, err := limits.NewLimitsService(context.Background(), source, counters, customResolver)

		require.NoError(t, err)
		assert.NotNil(t, svc)
	})

	t.Run("source load error", func(t *testing.T) {
		t.Parallel()

		// Create a source that returns an error
		failingSource := &failingSource{err: errors.New("load failed")}
		counters := createTestCounters()

		svc, err := limits.NewLimitsService(context.Background(), failingSource, counters, nil)

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrFailedToLoadPlans)
		assert.Nil(t, svc)
	})

	t.Run("nil source returns empty plans", func(t *testing.T) {
		t.Parallel()

		// Create a source that returns empty plans
		nilSource := &emptySource{}
		svc, err := limits.NewLimitsService(context.Background(), nilSource, nil, nil)

		require.NoError(t, err)
		assert.NotNil(t, svc)
	})

	t.Run("nil counters creates empty registry", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())

		svc, err := limits.NewLimitsService(context.Background(), source, nil, nil)

		require.NoError(t, err)
		assert.NotNil(t, svc)
	})
}

func TestService_CanCreate(t *testing.T) {
	t.Parallel()

	t.Run("within limits", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()

		err = svc.CanCreate(ctx, tenantID, limits.ResourceUsers)

		assert.NoError(t, err)
	})

	t.Run("at limit", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		// Counter returns exactly the limit
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 5, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()

		err = svc.CanCreate(ctx, tenantID, limits.ResourceUsers)

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrLimitExceeded)
	})

	t.Run("over limit", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 10, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()

		err = svc.CanCreate(ctx, tenantID, limits.ResourceUsers)

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrLimitExceeded)
	})

	t.Run("unlimited resource", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 1000000, nil // Very high usage
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "enterprise")
		tenantID := uuid.New()

		err = svc.CanCreate(ctx, tenantID, limits.ResourceUsers)

		assert.NoError(t, err)
	})

	t.Run("plan not found", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "non-existent")
		tenantID := uuid.New()

		err = svc.CanCreate(ctx, tenantID, limits.ResourceUsers)

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrPlanNotFound)
	})

	t.Run("resource not in plan", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()

		err = svc.CanCreate(ctx, tenantID, limits.Resource("unknown"))

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrInvalidResource)
	})

	t.Run("counter not registered", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry() // Empty registry
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()

		err = svc.CanCreate(ctx, tenantID, limits.ResourceUsers)

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrNoCounterRegistered)
	})

	t.Run("counter error", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		counterErr := errors.New("database error")
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 0, counterErr
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()

		err = svc.CanCreate(ctx, tenantID, limits.ResourceUsers)

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrFailedToCountResourceUsage)
	})

	t.Run("plan resolver error", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		// No plan ID in context
		ctx := context.Background()
		tenantID := uuid.New()

		err = svc.CanCreate(ctx, tenantID, limits.ResourceUsers)

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrPlanIDNotFound)
	})

	t.Run("empty plan ID", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "")
		tenantID := uuid.New()

		err = svc.CanCreate(ctx, tenantID, limits.ResourceUsers)

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrPlanNotFound)
	})
}

func TestService_GetUsage(t *testing.T) {
	t.Parallel()

	t.Run("successful usage retrieval", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 3, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()

		used, limit, err := svc.GetUsage(ctx, tenantID, limits.ResourceUsers)

		require.NoError(t, err)
		assert.Equal(t, int64(3), used)
		assert.Equal(t, int64(5), limit)
	})

	t.Run("unlimited resource", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 100, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "enterprise")
		tenantID := uuid.New()

		used, limit, err := svc.GetUsage(ctx, tenantID, limits.ResourceUsers)

		require.NoError(t, err)
		assert.Equal(t, int64(100), used)
		assert.Equal(t, limits.Unlimited, limit)
	})

	t.Run("error cases", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		testCases := []struct {
			name          string
			ctx           context.Context
			resource      limits.Resource
			expectedError error
		}{
			{
				name:          "no plan ID",
				ctx:           context.Background(),
				resource:      limits.ResourceUsers,
				expectedError: limits.ErrPlanIDNotFound,
			},
			{
				name:          "plan not found",
				ctx:           limits.SetPlanIDToContext(context.Background(), "invalid"),
				resource:      limits.ResourceUsers,
				expectedError: limits.ErrPlanNotFound,
			},
			{
				name:          "invalid resource",
				ctx:           limits.SetPlanIDToContext(context.Background(), "free"),
				resource:      limits.Resource("invalid"),
				expectedError: limits.ErrInvalidResource,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				used, limit, err := svc.GetUsage(tc.ctx, uuid.New(), tc.resource)

				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.expectedError)
				assert.Equal(t, int64(0), used)
				assert.Equal(t, int64(0), limit)
			})
		}
	})
}

func TestService_GetUsageSafe(t *testing.T) {
	t.Parallel()

	t.Run("returns values on success", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()

		used, limit := svc.GetUsageSafe(ctx, tenantID, limits.ResourceUsers)

		assert.Equal(t, int64(3), used)
		assert.Equal(t, int64(5), limit)
	})

	t.Run("returns zeros on error", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		// No plan ID in context
		ctx := context.Background()
		tenantID := uuid.New()

		used, limit := svc.GetUsageSafe(ctx, tenantID, limits.ResourceUsers)

		assert.Equal(t, int64(0), used)
		assert.Equal(t, int64(0), limit)
	})
}

func TestService_HasFeature(t *testing.T) {
	t.Parallel()

	t.Run("feature exists", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()

		hasAI := svc.HasFeature(ctx, tenantID, limits.FeatureAI)
		hasSSO := svc.HasFeature(ctx, tenantID, limits.FeatureSSO)

		assert.True(t, hasAI)
		assert.True(t, hasSSO)
	})

	t.Run("feature not in plan", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()

		hasAI := svc.HasFeature(ctx, tenantID, limits.FeatureAI)
		hasSSO := svc.HasFeature(ctx, tenantID, limits.FeatureSSO)

		assert.False(t, hasAI)
		assert.False(t, hasSSO)
	})

	t.Run("plan not found returns false", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "invalid")
		tenantID := uuid.New()

		hasFeature := svc.HasFeature(ctx, tenantID, limits.FeatureAI)

		assert.False(t, hasFeature)
	})

	t.Run("no plan ID returns false", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := context.Background()
		tenantID := uuid.New()

		hasFeature := svc.HasFeature(ctx, tenantID, limits.FeatureAI)

		assert.False(t, hasFeature)
	})

	t.Run("empty plan ID returns false", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "")
		tenantID := uuid.New()

		hasFeature := svc.HasFeature(ctx, tenantID, limits.FeatureAI)

		assert.False(t, hasFeature)
	})
}

func TestService_CheckTrial(t *testing.T) {
	t.Parallel()

	t.Run("active trial", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()
		startedAt := time.Now().AddDate(0, 0, -1) // Started yesterday

		err = svc.CheckTrial(ctx, tenantID, startedAt)

		assert.NoError(t, err)
	})

	t.Run("expired trial", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()
		startedAt := time.Now().AddDate(0, 0, -15) // Started 15 days ago

		err = svc.CheckTrial(ctx, tenantID, startedAt)

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrTrialExpired)
	})

	t.Run("no trial available", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()
		startedAt := time.Now()

		err = svc.CheckTrial(ctx, tenantID, startedAt)

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrTrialNotAvailable)
	})

	t.Run("plan not found", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "invalid")
		tenantID := uuid.New()
		startedAt := time.Now()

		err = svc.CheckTrial(ctx, tenantID, startedAt)

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrPlanNotFound)
	})

	t.Run("negative trial days", func(t *testing.T) {
		t.Parallel()

		// Create plans with a plan that has negative trial days
		plans := map[string]limits.Plan{
			"broken": {
				ID:        "broken",
				Name:      "Broken Plan",
				TrialDays: -7, // Negative trial days
				Limits: map[limits.Resource]int64{
					limits.ResourceUsers: 10,
				},
			},
		}

		source := limits.NewInMemSource(plans)
		counters := createTestCounters()

		// Service creation should fail due to invalid plan configuration
		_, err := limits.NewLimitsService(context.Background(), source, counters, nil)

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrInvalidPlanConfiguration)
		assert.Contains(t, err.Error(), "negative trial days")
		assert.Contains(t, err.Error(), "broken")
	})

	t.Run("zero trial days should be valid", func(t *testing.T) {
		t.Parallel()

		// Create plans with zero trial days (should be valid)
		plans := map[string]limits.Plan{
			"valid": {
				ID:        "valid",
				Name:      "Valid Plan",
				TrialDays: 0, // Zero trial days (no trial)
				Limits: map[limits.Resource]int64{
					limits.ResourceUsers: 10,
				},
			},
		}

		source := limits.NewInMemSource(plans)
		counters := createTestCounters()

		// Service creation should succeed
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)

		assert.NoError(t, err)
		assert.NotNil(t, svc)

		// Verify that CheckTrial works correctly for zero trial days
		ctx := limits.SetPlanIDToContext(context.Background(), "valid")
		tenantID := uuid.New()
		startedAt := time.Now().UTC()

		err = svc.CheckTrial(ctx, tenantID, startedAt)
		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrTrialNotAvailable) // Zero trial days means no trial available
	})
}

func TestService_VerifyPlan(t *testing.T) {
	t.Parallel()

	t.Run("valid plan", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		err = svc.VerifyPlan(context.Background(), "free")
		assert.NoError(t, err)

		err = svc.VerifyPlan(context.Background(), "pro")
		assert.NoError(t, err)

		err = svc.VerifyPlan(context.Background(), "enterprise")
		assert.NoError(t, err)
	})

	t.Run("invalid plan", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		err = svc.VerifyPlan(context.Background(), "invalid")

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrPlanNotFound)
	})
}

func TestService_GetUsagePercentage(t *testing.T) {
	t.Parallel()

	t.Run("normal percentage", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 3, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()

		percentage := svc.GetUsagePercentage(ctx, tenantID, limits.ResourceUsers)

		assert.Equal(t, 60, percentage) // 3/5 * 100 = 60%
	})

	t.Run("at limit", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 5, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()

		percentage := svc.GetUsagePercentage(ctx, tenantID, limits.ResourceUsers)

		assert.Equal(t, 100, percentage)
	})

	t.Run("over limit capped at 100", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 10, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()

		percentage := svc.GetUsagePercentage(ctx, tenantID, limits.ResourceUsers)

		assert.Equal(t, 100, percentage)
	})

	t.Run("unlimited returns -1", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 100, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "enterprise")
		tenantID := uuid.New()

		percentage := svc.GetUsagePercentage(ctx, tenantID, limits.ResourceUsers)

		assert.Equal(t, -1, percentage)
	})

	t.Run("zero limit returns 100", func(t *testing.T) {
		t.Parallel()

		plans := map[string]limits.Plan{
			"zero": {
				ID:   "zero",
				Name: "Zero Plan",
				Limits: map[limits.Resource]int64{
					limits.ResourceUsers: 0,
				},
			},
		}
		source := limits.NewInMemSource(plans)
		counters := limits.NewRegistry()
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 0, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "zero")
		tenantID := uuid.New()

		percentage := svc.GetUsagePercentage(ctx, tenantID, limits.ResourceUsers)

		assert.Equal(t, 100, percentage)
	})

	t.Run("error returns 0", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := context.Background() // No plan ID
		tenantID := uuid.New()

		percentage := svc.GetUsagePercentage(ctx, tenantID, limits.ResourceUsers)

		assert.Equal(t, 0, percentage)
	})
}

func TestService_CanDowngrade(t *testing.T) {
	t.Parallel()

	t.Run("downgrade possible", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		// Low usage fits in free plan
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 3, nil
		})
		counters.Register(limits.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 2, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()

		err = svc.CanDowngrade(ctx, tenantID, "free")

		assert.NoError(t, err)
	})

	t.Run("downgrade not possible - over limit", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		// Usage exceeds free plan limits
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 10, nil
		})
		counters.Register(limits.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 2, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()

		err = svc.CanDowngrade(ctx, tenantID, "free")

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrDowngradeNotPossible)
	})

	t.Run("from unlimited to limited", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		// High usage that won't fit in pro plan
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 100, nil
		})
		counters.Register(limits.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 30, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "enterprise")
		tenantID := uuid.New()

		err = svc.CanDowngrade(ctx, tenantID, "pro")

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrDowngradeNotPossible)
	})

	t.Run("target plan not found", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()

		err = svc.CanDowngrade(ctx, tenantID, "invalid")

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrPlanNotFound)
	})

	t.Run("current plan not found", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "invalid")
		tenantID := uuid.New()

		err = svc.CanDowngrade(ctx, tenantID, "free")

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrPlanNotFound)
	})

	t.Run("no counter for resource allows downgrade", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		// Only register users counter, not projects
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 3, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()

		err = svc.CanDowngrade(ctx, tenantID, "free")

		assert.NoError(t, err)
	})

	t.Run("counter error", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		counterErr := errors.New("database error")
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 0, counterErr
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()

		err = svc.CanDowngrade(ctx, tenantID, "free")

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrFailedToCountResourceUsage)
	})
}

func TestService_GetAllUsage(t *testing.T) {
	t.Parallel()

	t.Run("get all usage", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 3, nil
		})
		counters.Register(limits.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 2, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()

		usage, err := svc.GetAllUsage(ctx, tenantID)

		require.NoError(t, err)
		assert.Len(t, usage, 2)

		assert.Equal(t, limits.UsageInfo{Current: 3, Limit: 5}, usage[limits.ResourceUsers])
		assert.Equal(t, limits.UsageInfo{Current: 2, Limit: 3}, usage[limits.ResourceProjects])
	})

	t.Run("missing counter returns zero usage", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		// Only register users counter
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 3, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()

		usage, err := svc.GetAllUsage(ctx, tenantID)

		require.NoError(t, err)
		assert.Len(t, usage, 2)

		assert.Equal(t, limits.UsageInfo{Current: 3, Limit: 5}, usage[limits.ResourceUsers])
		assert.Equal(t, limits.UsageInfo{Current: 0, Limit: 3}, usage[limits.ResourceProjects])
	})

	t.Run("counter error returns zero usage", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 0, errors.New("database error")
		})
		counters.Register(limits.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 2, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()

		usage, err := svc.GetAllUsage(ctx, tenantID)

		// Should not error, just return zero for failed counter
		require.NoError(t, err)
		assert.Len(t, usage, 2)

		assert.Equal(t, limits.UsageInfo{Current: 0, Limit: 5}, usage[limits.ResourceUsers])
		assert.Equal(t, limits.UsageInfo{Current: 2, Limit: 3}, usage[limits.ResourceProjects])
	})

	t.Run("plan not found", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "invalid")
		tenantID := uuid.New()

		usage, err := svc.GetAllUsage(ctx, tenantID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrPlanNotFound)
		assert.Nil(t, usage)
	})
}

func TestService_Concurrency(t *testing.T) {
	t.Parallel()

	t.Run("concurrent reads", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := limits.NewRegistry()

		// Thread-safe counter
		var mu sync.Mutex
		callCount := 0
		counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			mu.Lock()
			callCount++
			mu.Unlock()
			return 3, nil
		})

		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()

		// Run concurrent operations
		var wg sync.WaitGroup
		concurrency := 100
		wg.Add(concurrency)

		errors := make([]error, concurrency)
		for i := range concurrency {
			go func(index int) {
				defer wg.Done()
				err := svc.CanCreate(ctx, tenantID, limits.ResourceUsers)
				errors[index] = err
			}(i)
		}

		wg.Wait()

		// Verify all succeeded
		for _, err := range errors {
			assert.NoError(t, err)
		}

		mu.Lock()
		assert.Equal(t, concurrency, callCount)
		mu.Unlock()
	})

	t.Run("concurrent different operations", func(t *testing.T) {
		t.Parallel()

		source := limits.NewInMemSource(createTestPlans())
		counters := createTestCounters()
		svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
		require.NoError(t, err)

		ctx := limits.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()

		// Run different operations concurrently
		var wg sync.WaitGroup
		wg.Add(5)

		// CanCreate
		go func() {
			defer wg.Done()
			for range 20 {
				_ = svc.CanCreate(ctx, tenantID, limits.ResourceUsers)
			}
		}()

		// GetUsage
		go func() {
			defer wg.Done()
			for range 20 {
				_, _, _ = svc.GetUsage(ctx, tenantID, limits.ResourceUsers)
			}
		}()

		// HasFeature
		go func() {
			defer wg.Done()
			for range 20 {
				_ = svc.HasFeature(ctx, tenantID, limits.FeatureAI)
			}
		}()

		// GetUsagePercentage
		go func() {
			defer wg.Done()
			for range 20 {
				_ = svc.GetUsagePercentage(ctx, tenantID, limits.ResourceUsers)
			}
		}()

		// GetAllUsage
		go func() {
			defer wg.Done()
			for range 20 {
				_, _ = svc.GetAllUsage(ctx, tenantID)
			}
		}()

		wg.Wait()
		// Test passes if no panic/race
	})
}

// Helper types for testing
type failingSource struct {
	err error
}

func (s *failingSource) Load(ctx context.Context) (map[string]limits.Plan, error) {
	return nil, s.err
}

type emptySource struct{}

func (s *emptySource) Load(ctx context.Context) (map[string]limits.Plan, error) {
	return map[string]limits.Plan{}, nil
}
