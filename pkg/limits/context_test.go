package limits_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/limits"
)

func TestSetGetPlanIDContext(t *testing.T) {
	t.Parallel()

	t.Run("set and get plan ID", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		planID := "pro-plan"

		// Set plan ID
		ctx = limits.SetPlanIDToContext(ctx, planID)

		// Get plan ID
		result, ok := limits.GetPlanIDFromContext(ctx)

		assert.True(t, ok)
		assert.Equal(t, planID, result)
	})

	t.Run("get from empty context", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		result, ok := limits.GetPlanIDFromContext(ctx)

		assert.False(t, ok)
		assert.Empty(t, result)
	})

	t.Run("overwrite existing plan ID", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		// Set initial plan ID
		ctx = limits.SetPlanIDToContext(ctx, "basic")

		// Overwrite with new plan ID
		ctx = limits.SetPlanIDToContext(ctx, "premium")

		result, ok := limits.GetPlanIDFromContext(ctx)

		assert.True(t, ok)
		assert.Equal(t, "premium", result)
	})

	t.Run("context isolation", func(t *testing.T) {
		t.Parallel()

		baseCtx := context.Background()
		ctx1 := limits.SetPlanIDToContext(baseCtx, "plan1")
		ctx2 := limits.SetPlanIDToContext(baseCtx, "plan2")

		result1, ok1 := limits.GetPlanIDFromContext(ctx1)
		result2, ok2 := limits.GetPlanIDFromContext(ctx2)

		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.Equal(t, "plan1", result1)
		assert.Equal(t, "plan2", result2)
	})

	t.Run("empty string plan ID", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctx = limits.SetPlanIDToContext(ctx, "")

		result, ok := limits.GetPlanIDFromContext(ctx)

		assert.True(t, ok)
		assert.Equal(t, "", result)
	})
}

func TestPlanIDContextResolver(t *testing.T) {
	t.Parallel()

	t.Run("resolve from context", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctx = limits.SetPlanIDToContext(ctx, "enterprise")
		tenantID := uuid.New()

		planID, err := limits.PlanIDContextResolver(ctx, tenantID)

		require.NoError(t, err)
		assert.Equal(t, "enterprise", planID)
	})

	t.Run("error when not in context", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		tenantID := uuid.New()

		planID, err := limits.PlanIDContextResolver(ctx, tenantID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, limits.ErrPlanIDNotFound)
		assert.ErrorIs(t, err, limits.ErrPlanIDNotInContext)
		assert.Empty(t, planID)
	})

	t.Run("ignores tenant ID parameter", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctx = limits.SetPlanIDToContext(ctx, "basic")

		// Try with different tenant IDs
		tenantID1 := uuid.New()
		tenantID2 := uuid.New()

		planID1, err1 := limits.PlanIDContextResolver(ctx, tenantID1)
		planID2, err2 := limits.PlanIDContextResolver(ctx, tenantID2)

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, "basic", planID1)
		assert.Equal(t, "basic", planID2)
	})

	t.Run("works with cancelled context", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		ctx = limits.SetPlanIDToContext(ctx, "pro")
		cancel()

		tenantID := uuid.New()
		planID, err := limits.PlanIDContextResolver(ctx, tenantID)

		// Should still work even with cancelled context
		require.NoError(t, err)
		assert.Equal(t, "pro", planID)
	})
}
