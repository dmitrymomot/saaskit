package limits_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/limits"
)

func TestCounterRegistry(t *testing.T) {
	t.Parallel()

	t.Run("new registry is empty", func(t *testing.T) {
		t.Parallel()

		registry := limits.NewRegistry()

		assert.NotNil(t, registry)
		assert.Empty(t, registry)
	})

	t.Run("register and retrieve counter", func(t *testing.T) {
		t.Parallel()

		registry := limits.NewRegistry()
		counter := func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 42, nil
		}

		registry.Register(limits.ResourceUsers, counter)

		// Verify counter was registered
		registeredCounter, exists := registry[limits.ResourceUsers]
		assert.True(t, exists)
		assert.NotNil(t, registeredCounter)

		// Test counter function
		result, err := registeredCounter(context.Background(), uuid.New())
		require.NoError(t, err)
		assert.Equal(t, int64(42), result)
	})

	t.Run("register multiple counters", func(t *testing.T) {
		t.Parallel()

		registry := limits.NewRegistry()

		userCounter := func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 10, nil
		}
		projectCounter := func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 5, nil
		}

		registry.Register(limits.ResourceUsers, userCounter)
		registry.Register(limits.ResourceProjects, projectCounter)

		assert.Len(t, registry, 2)

		// Test both counters
		userCount, err := registry[limits.ResourceUsers](context.Background(), uuid.New())
		require.NoError(t, err)
		assert.Equal(t, int64(10), userCount)

		projectCount, err := registry[limits.ResourceProjects](context.Background(), uuid.New())
		require.NoError(t, err)
		assert.Equal(t, int64(5), projectCount)
	})

	t.Run("replace existing counter", func(t *testing.T) {
		t.Parallel()

		registry := limits.NewRegistry()

		// Register initial counter
		registry.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 1, nil
		})

		// Replace with new counter
		registry.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 2, nil
		})

		assert.Len(t, registry, 1)

		// Verify new counter is used
		count, err := registry[limits.ResourceUsers](context.Background(), uuid.New())
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	t.Run("panic on nil counter", func(t *testing.T) {
		t.Parallel()

		registry := limits.NewRegistry()

		assert.Panics(t, func() {
			registry.Register(limits.ResourceUsers, nil)
		}, "should panic when registering nil counter")
	})

	t.Run("counter with error", func(t *testing.T) {
		t.Parallel()

		registry := limits.NewRegistry()
		expectedErr := errors.New("database error")

		counter := func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 0, expectedErr
		}

		registry.Register(limits.ResourceUsers, counter)

		count, err := registry[limits.ResourceUsers](context.Background(), uuid.New())
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("counter using context", func(t *testing.T) {
		t.Parallel()

		registry := limits.NewRegistry()

		// Counter that checks context cancellation
		counter := func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			default:
				return 15, nil
			}
		}

		registry.Register(limits.ResourceUsers, counter)

		// Test with normal context
		count, err := registry[limits.ResourceUsers](context.Background(), uuid.New())
		require.NoError(t, err)
		assert.Equal(t, int64(15), count)

		// Test with cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		count, err = registry[limits.ResourceUsers](ctx, uuid.New())
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("counter using tenant ID", func(t *testing.T) {
		t.Parallel()

		registry := limits.NewRegistry()

		// Counter that returns different values based on tenant ID
		tenantCounts := map[uuid.UUID]int64{
			uuid.MustParse("11111111-1111-1111-1111-111111111111"): 10,
			uuid.MustParse("22222222-2222-2222-2222-222222222222"): 20,
		}

		counter := func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			if count, ok := tenantCounts[tenantID]; ok {
				return count, nil
			}
			return 0, nil
		}

		registry.Register(limits.ResourceUsers, counter)

		// Test different tenants
		tenant1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		count1, err := registry[limits.ResourceUsers](context.Background(), tenant1)
		require.NoError(t, err)
		assert.Equal(t, int64(10), count1)

		tenant2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		count2, err := registry[limits.ResourceUsers](context.Background(), tenant2)
		require.NoError(t, err)
		assert.Equal(t, int64(20), count2)

		// Unknown tenant
		unknownTenant := uuid.New()
		count3, err := registry[limits.ResourceUsers](context.Background(), unknownTenant)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count3)
	})

	t.Run("custom resource types", func(t *testing.T) {
		t.Parallel()

		registry := limits.NewRegistry()

		// Register counter for custom resource
		customResource := limits.Resource("api_calls")
		counter := func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 1000, nil
		}

		registry.Register(customResource, counter)

		count, err := registry[customResource](context.Background(), uuid.New())
		require.NoError(t, err)
		assert.Equal(t, int64(1000), count)
	})
}
