package feature_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/feature"
)

func TestAlwaysStrategy(t *testing.T) {
	ctx := context.Background()

	t.Run("AlwaysOn", func(t *testing.T) {
		strategy := feature.NewAlwaysOnStrategy()
		enabled, err := strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.True(t, enabled)
	})

	t.Run("AlwaysOff", func(t *testing.T) {
		strategy := feature.NewAlwaysOffStrategy()
		enabled, err := strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.False(t, enabled)
	})
}

func TestTargetedStrategy(t *testing.T) {
	t.Run("EmptyCriteria", func(t *testing.T) {
		strategy := feature.NewTargetedStrategy(feature.TargetCriteria{})
		enabled, err := strategy.Evaluate(context.Background())
		require.Error(t, err)
		assert.Equal(t, feature.ErrInvalidStrategy, err)
		assert.False(t, enabled)
	})

	t.Run("SpecificUserIDs", func(t *testing.T) {
		criteria := feature.TargetCriteria{
			UserIDs: []string{"user1", "user2", "user3"},
		}
		strategy := feature.NewTargetedStrategy(criteria)

		// Test with matching user ID
		ctx := context.WithValue(context.Background(), feature.UserIDKey, "user2")
		enabled, err := strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.True(t, enabled)

		// Test with non-matching user ID
		ctx = context.WithValue(context.Background(), feature.UserIDKey, "user4")
		enabled, err = strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.False(t, enabled)

		// Test with missing user ID
		enabled, err = strategy.Evaluate(context.Background())
		require.NoError(t, err)
		assert.False(t, enabled)
	})

	t.Run("UserGroups", func(t *testing.T) {
		criteria := feature.TargetCriteria{
			Groups: []string{"admin", "beta-testers"},
		}
		strategy := feature.NewTargetedStrategy(criteria)

		// Test with matching group
		ctx := context.WithValue(context.Background(), feature.UserGroupsKey, []string{"user", "beta-testers"})
		enabled, err := strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.True(t, enabled)

		// Test with non-matching group
		ctx = context.WithValue(context.Background(), feature.UserGroupsKey, []string{"user", "guest"})
		enabled, err = strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.False(t, enabled)

		// Test with missing groups
		enabled, err = strategy.Evaluate(context.Background())
		require.NoError(t, err)
		assert.False(t, enabled)
	})

	t.Run("PercentageRollout", func(t *testing.T) {
		// Set up a 50% rollout
		percentage := 50
		criteria := feature.TargetCriteria{
			Percentage: &percentage,
		}
		strategy := feature.NewTargetedStrategy(criteria)

		// We can't test specific outcomes since it's based on hash values,
		// but we can test that it works without error for different user IDs
		for _, userID := range []string{"user1", "user2", "user3", "user4"} {
			ctx := context.WithValue(context.Background(), feature.UserIDKey, userID)
			_, err := strategy.Evaluate(ctx)
			require.NoError(t, err)
		}

		// Test invalid percentage values
		invalidPercentage := 101
		invalidCriteria := feature.TargetCriteria{
			Percentage: &invalidPercentage,
		}
		strategy = feature.NewTargetedStrategy(invalidCriteria)
		_, err := strategy.Evaluate(context.WithValue(context.Background(), feature.UserIDKey, "user1"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "percentage must be between 0 and 100")

		// Test with missing user ID
		percentage = 50
		criteria = feature.TargetCriteria{
			Percentage: &percentage,
		}
		strategy = feature.NewTargetedStrategy(criteria)
		enabled, err := strategy.Evaluate(context.Background())
		require.NoError(t, err)
		assert.False(t, enabled)
	})

	t.Run("AllowDenyLists", func(t *testing.T) {
		criteria := feature.TargetCriteria{
			AllowList: []string{"special-user", "vip-user"},
			DenyList:  []string{"banned-user"},
			// Additional criteria that would normally enable a user
			Groups: []string{"beta-testers"},
		}
		strategy := feature.NewTargetedStrategy(criteria)

		// Test with user on allow list
		ctx := context.WithValue(context.Background(), feature.UserIDKey, "special-user")
		enabled, err := strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.True(t, enabled)

		// Test with user on deny list (even if they'd be eligible by other criteria)
		ctx = context.WithValue(context.Background(), feature.UserIDKey, "banned-user")
		ctx = context.WithValue(ctx, feature.UserGroupsKey, []string{"beta-testers"})
		enabled, err = strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.False(t, enabled)

		// Test with user eligible by criteria but not on allow or deny list
		ctx = context.WithValue(context.Background(), feature.UserIDKey, "regular-user")
		ctx = context.WithValue(ctx, feature.UserGroupsKey, []string{"beta-testers"})
		enabled, err = strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.True(t, enabled)
	})
}

func TestEnvironmentStrategy(t *testing.T) {
	t.Run("EmptyEnvironments", func(t *testing.T) {
		strategy := feature.NewEnvironmentStrategy()
		enabled, err := strategy.Evaluate(context.Background())
		require.Error(t, err)
		assert.Equal(t, feature.ErrInvalidStrategy, err)
		assert.False(t, enabled)
	})

	t.Run("MatchingEnvironment", func(t *testing.T) {
		strategy := feature.NewEnvironmentStrategy("dev", "staging")

		// Test with matching environment
		ctx := context.WithValue(context.Background(), feature.EnvironmentKey, "dev")
		enabled, err := strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.True(t, enabled)

		// Test with non-matching environment
		ctx = context.WithValue(context.Background(), feature.EnvironmentKey, "production")
		enabled, err = strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.False(t, enabled)

		// Test with missing environment
		enabled, err = strategy.Evaluate(context.Background())
		require.NoError(t, err)
		assert.False(t, enabled)
	})
}

func TestCompositeStrategy(t *testing.T) {
	t.Run("EmptyStrategies", func(t *testing.T) {
		strategy := feature.NewAndStrategy()
		enabled, err := strategy.Evaluate(context.Background())
		require.Error(t, err)
		assert.Equal(t, feature.ErrInvalidStrategy, err)
		assert.False(t, enabled)

		strategy = feature.NewOrStrategy()
		enabled, err = strategy.Evaluate(context.Background())
		require.Error(t, err)
		assert.Equal(t, feature.ErrInvalidStrategy, err)
		assert.False(t, enabled)
	})

	t.Run("AndStrategy", func(t *testing.T) {
		// Create some test strategies
		alwaysOn := feature.NewAlwaysOnStrategy()
		alwaysOff := feature.NewAlwaysOffStrategy()

		// Test with a passing combination (all true)
		strategy := feature.NewAndStrategy(alwaysOn, alwaysOn)
		enabled, err := strategy.Evaluate(context.Background())
		require.NoError(t, err)
		assert.True(t, enabled)

		// Test with a failing combination (one false)
		strategy = feature.NewAndStrategy(alwaysOn, alwaysOff)
		enabled, err = strategy.Evaluate(context.Background())
		require.NoError(t, err)
		assert.False(t, enabled)

		// Test with all failing
		strategy = feature.NewAndStrategy(alwaysOff, alwaysOff)
		enabled, err = strategy.Evaluate(context.Background())
		require.NoError(t, err)
		assert.False(t, enabled)
	})

	t.Run("OrStrategy", func(t *testing.T) {
		// Create some test strategies
		alwaysOn := feature.NewAlwaysOnStrategy()
		alwaysOff := feature.NewAlwaysOffStrategy()

		// Test with a passing combination (one true)
		strategy := feature.NewOrStrategy(alwaysOn, alwaysOff)
		enabled, err := strategy.Evaluate(context.Background())
		require.NoError(t, err)
		assert.True(t, enabled)

		// Test with all passing
		strategy = feature.NewOrStrategy(alwaysOn, alwaysOn)
		enabled, err = strategy.Evaluate(context.Background())
		require.NoError(t, err)
		assert.True(t, enabled)

		// Test with all failing
		strategy = feature.NewOrStrategy(alwaysOff, alwaysOff)
		enabled, err = strategy.Evaluate(context.Background())
		require.NoError(t, err)
		assert.False(t, enabled)
	})
}
