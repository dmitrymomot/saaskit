package feature_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/feature"
)

// Test helper context keys
type (
	testUserIDKey      struct{}
	testUserGroupsKey  struct{}
	testEnvironmentKey struct{}
)

// Test helper extractors
func testUserIDExtractor(ctx context.Context) string {
	userID, _ := ctx.Value(testUserIDKey{}).(string)
	return userID
}

func testUserGroupsExtractor(ctx context.Context) []string {
	groups, _ := ctx.Value(testUserGroupsKey{}).([]string)
	return groups
}

func testEnvironmentExtractor(ctx context.Context) string {
	env, _ := ctx.Value(testEnvironmentKey{}).(string)
	return env
}

func TestAlwaysStrategy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("AlwaysOn", func(t *testing.T) {
		t.Parallel()
		strategy := feature.NewAlwaysOnStrategy()
		enabled, err := strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.True(t, enabled)
	})

	t.Run("AlwaysOff", func(t *testing.T) {
		t.Parallel()
		strategy := feature.NewAlwaysOffStrategy()
		enabled, err := strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.False(t, enabled)
	})
}

func TestTargetedStrategy(t *testing.T) {
	t.Parallel()
	t.Run("EmptyCriteria", func(t *testing.T) {
		t.Parallel()
		strategy := feature.NewTargetedStrategy(feature.TargetCriteria{})
		enabled, err := strategy.Evaluate(context.Background())
		require.Error(t, err)
		assert.Equal(t, feature.ErrInvalidStrategy, err)
		assert.False(t, enabled)
	})

	t.Run("SpecificUserIDs", func(t *testing.T) {
		t.Parallel()
		criteria := feature.TargetCriteria{
			UserIDs: []string{"user1", "user2", "user3"},
		}
		strategy := feature.NewTargetedStrategy(criteria,
			feature.WithUserIDExtractor(testUserIDExtractor),
		)

		// Test with matching user ID
		ctx := context.WithValue(context.Background(), testUserIDKey{}, "user2")
		enabled, err := strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.True(t, enabled)

		// Test with non-matching user ID
		ctx = context.WithValue(context.Background(), testUserIDKey{}, "user4")
		enabled, err = strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.False(t, enabled)

		// Test with missing user ID
		enabled, err = strategy.Evaluate(context.Background())
		require.NoError(t, err)
		assert.False(t, enabled)

		// Test without extractor configured
		strategyNoExtractor := feature.NewTargetedStrategy(criteria)
		ctx = context.WithValue(context.Background(), testUserIDKey{}, "user2")
		enabled, err = strategyNoExtractor.Evaluate(ctx)
		require.NoError(t, err)
		assert.False(t, enabled) // Should be false without extractor
	})

	t.Run("UserGroups", func(t *testing.T) {
		t.Parallel()
		criteria := feature.TargetCriteria{
			Groups: []string{"admin", "beta-testers"},
		}
		strategy := feature.NewTargetedStrategy(criteria,
			feature.WithUserGroupsExtractor(testUserGroupsExtractor),
		)

		// Test with matching group
		ctx := context.WithValue(context.Background(), testUserGroupsKey{}, []string{"user", "beta-testers"})
		enabled, err := strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.True(t, enabled)

		// Test with non-matching group
		ctx = context.WithValue(context.Background(), testUserGroupsKey{}, []string{"user", "guest"})
		enabled, err = strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.False(t, enabled)

		// Test with missing groups
		enabled, err = strategy.Evaluate(context.Background())
		require.NoError(t, err)
		assert.False(t, enabled)

		// Test without extractor configured
		strategyNoExtractor := feature.NewTargetedStrategy(criteria)
		ctx = context.WithValue(context.Background(), testUserGroupsKey{}, []string{"beta-testers"})
		enabled, err = strategyNoExtractor.Evaluate(ctx)
		require.NoError(t, err)
		assert.False(t, enabled) // Should be false without extractor
	})

	t.Run("PercentageRollout", func(t *testing.T) {
		t.Parallel()
		// Set up a 50% rollout
		percentage := 50
		criteria := feature.TargetCriteria{
			Percentage: &percentage,
		}
		strategy := feature.NewTargetedStrategy(criteria,
			feature.WithUserIDExtractor(testUserIDExtractor),
		)

		// We can't test specific outcomes since it's based on hash values,
		// but we can test that it works without error for different user IDs
		for _, userID := range []string{"user1", "user2", "user3", "user4"} {
			ctx := context.WithValue(context.Background(), testUserIDKey{}, userID)
			_, err := strategy.Evaluate(ctx)
			require.NoError(t, err)
		}

		// Test invalid percentage values
		invalidPercentage := 101
		invalidCriteria := feature.TargetCriteria{
			Percentage: &invalidPercentage,
		}
		strategy = feature.NewTargetedStrategy(invalidCriteria,
			feature.WithUserIDExtractor(testUserIDExtractor),
		)
		ctx := context.WithValue(context.Background(), testUserIDKey{}, "user1")
		_, err := strategy.Evaluate(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "percentage must be between 0 and 100")

		// Test with missing user ID
		percentage = 50
		criteria = feature.TargetCriteria{
			Percentage: &percentage,
		}
		strategy = feature.NewTargetedStrategy(criteria,
			feature.WithUserIDExtractor(testUserIDExtractor),
		)
		enabled, err := strategy.Evaluate(context.Background())
		require.NoError(t, err)
		assert.False(t, enabled)
	})

	t.Run("AllowDenyLists", func(t *testing.T) {
		t.Parallel()
		criteria := feature.TargetCriteria{
			AllowList: []string{"special-user", "vip-user"},
			DenyList:  []string{"banned-user"},
			// Additional criteria that would normally enable a user
			Groups: []string{"beta-testers"},
		}
		strategy := feature.NewTargetedStrategy(criteria,
			feature.WithUserIDExtractor(testUserIDExtractor),
			feature.WithUserGroupsExtractor(testUserGroupsExtractor),
		)

		// Test with user on allow list
		ctx := context.WithValue(context.Background(), testUserIDKey{}, "special-user")
		enabled, err := strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.True(t, enabled)

		// Test with user on deny list (even if they'd be eligible by other criteria)
		ctx = context.WithValue(context.Background(), testUserIDKey{}, "banned-user")
		ctx = context.WithValue(ctx, testUserGroupsKey{}, []string{"beta-testers"})
		enabled, err = strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.False(t, enabled)

		// Test with user eligible by criteria but not on allow or deny list
		ctx = context.WithValue(context.Background(), testUserIDKey{}, "regular-user")
		ctx = context.WithValue(ctx, testUserGroupsKey{}, []string{"beta-testers"})
		enabled, err = strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.True(t, enabled)
	})
}

func TestEnvironmentStrategy(t *testing.T) {
	t.Parallel()
	t.Run("EmptyEnvironments", func(t *testing.T) {
		t.Parallel()
		strategy := feature.NewEnvironmentStrategy([]string{})
		enabled, err := strategy.Evaluate(context.Background())
		require.Error(t, err)
		assert.Equal(t, feature.ErrInvalidStrategy, err)
		assert.False(t, enabled)
	})

	t.Run("MatchingEnvironment", func(t *testing.T) {
		t.Parallel()
		strategy := feature.NewEnvironmentStrategy([]string{"dev", "staging"},
			feature.WithEnvironmentExtractor(testEnvironmentExtractor),
		)

		// Test with matching environment
		ctx := context.WithValue(context.Background(), testEnvironmentKey{}, "dev")
		enabled, err := strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.True(t, enabled)

		// Test with non-matching environment
		ctx = context.WithValue(context.Background(), testEnvironmentKey{}, "production")
		enabled, err = strategy.Evaluate(ctx)
		require.NoError(t, err)
		assert.False(t, enabled)

		// Test with missing environment
		enabled, err = strategy.Evaluate(context.Background())
		require.NoError(t, err)
		assert.False(t, enabled)

		// Test without extractor configured
		strategyNoExtractor := feature.NewEnvironmentStrategy([]string{"dev", "staging"})
		ctx = context.WithValue(context.Background(), testEnvironmentKey{}, "dev")
		enabled, err = strategyNoExtractor.Evaluate(ctx)
		require.NoError(t, err)
		assert.False(t, enabled) // Should be false without extractor
	})
}

func TestCompositeStrategy(t *testing.T) {
	t.Parallel()
	t.Run("EmptyStrategies", func(t *testing.T) {
		t.Parallel()
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
		t.Parallel()
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
		t.Parallel()
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
