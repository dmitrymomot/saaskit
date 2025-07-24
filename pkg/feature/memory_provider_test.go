package feature_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/feature"
)

// Test helper context key for memory provider tests
type testMemoryUserIDKey struct{}

// Test helper extractor for memory provider tests
func testMemoryUserIDExtractor(ctx context.Context) string {
	userID, _ := ctx.Value(testMemoryUserIDKey{}).(string)
	return userID
}

func TestMemoryProvider(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("NewMemoryProvider", func(t *testing.T) {
		t.Parallel()
		// Test creating an empty provider
		provider, err := feature.NewMemoryProvider()
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Test creating with initial flags
		flags := []*feature.Flag{
			{
				Name:        "test-flag-1",
				Description: "Test flag 1",
				Enabled:     true,
				Strategy:    feature.NewAlwaysOnStrategy(),
			},
			{
				Name:        "test-flag-2",
				Description: "Test flag 2",
				Enabled:     false,
				Strategy:    feature.NewAlwaysOffStrategy(),
			},
		}

		provider, err = feature.NewMemoryProvider(flags...)
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Test creating with invalid flag
		_, err = feature.NewMemoryProvider(&feature.Flag{Name: ""})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "flag name cannot be empty")
	})

	t.Run("CreateFlag", func(t *testing.T) {
		t.Parallel()
		provider, _ := feature.NewMemoryProvider()

		// Test creating a valid flag
		flag := &feature.Flag{
			Name:        "new-flag",
			Description: "A new flag",
			Enabled:     true,
			Strategy:    feature.NewAlwaysOnStrategy(),
			Tags:        []string{"test", "new"},
		}

		err := provider.CreateFlag(ctx, flag)
		require.NoError(t, err)

		// Test creating with empty name
		err = provider.CreateFlag(ctx, &feature.Flag{Name: ""})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "flag name cannot be empty")

		// Test creating a nil flag
		err = provider.CreateFlag(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "flag cannot be nil")

		// Test creating a duplicate flag
		err = provider.CreateFlag(ctx, flag)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "flag already exists")
	})

	t.Run("GetFlag", func(t *testing.T) {
		t.Parallel()
		// Create a provider with a test flag
		testFlag := &feature.Flag{
			Name:        "test-flag",
			Description: "Test flag",
			Enabled:     true,
			Strategy:    feature.NewAlwaysOnStrategy(),
			Tags:        []string{"test"},
		}
		provider, _ := feature.NewMemoryProvider(testFlag)

		// Test retrieving an existing flag
		flag, err := provider.GetFlag(ctx, "test-flag")
		require.NoError(t, err)
		assert.Equal(t, testFlag.Name, flag.Name)
		assert.Equal(t, testFlag.Description, flag.Description)
		assert.Equal(t, testFlag.Enabled, flag.Enabled)
		assert.Equal(t, testFlag.Tags, flag.Tags)

		// Test retrieving a non-existent flag
		_, err = provider.GetFlag(ctx, "non-existent")
		require.Error(t, err)
		assert.Equal(t, feature.ErrFlagNotFound, err)

		// Verify GetFlag returns a copy, not the original reference
		retrievedFlag, _ := provider.GetFlag(ctx, "test-flag")
		retrievedFlag.Enabled = false
		// Get the flag again and verify it's unchanged
		originalFlag, _ := provider.GetFlag(ctx, "test-flag")
		assert.True(t, originalFlag.Enabled, "Original flag should be unmodified")
	})

	t.Run("UpdateFlag", func(t *testing.T) {
		t.Parallel()
		// Create a provider with a test flag
		testFlag := &feature.Flag{
			Name:        "update-flag",
			Description: "Flag to update",
			Enabled:     true,
			Strategy:    feature.NewAlwaysOnStrategy(),
			CreatedAt:   time.Now().Add(-1 * time.Hour), // Set creation time in the past
		}
		provider, _ := feature.NewMemoryProvider(testFlag)

		// Update the flag
		updatedFlag := &feature.Flag{
			Name:        "update-flag",
			Description: "Updated description",
			Enabled:     false,
			Strategy:    feature.NewAlwaysOffStrategy(),
			Tags:        []string{"updated"},
		}

		err := provider.UpdateFlag(ctx, updatedFlag)
		require.NoError(t, err)

		// Retrieve and verify the update
		flag, err := provider.GetFlag(ctx, "update-flag")
		require.NoError(t, err)
		assert.Equal(t, "Updated description", flag.Description)
		assert.False(t, flag.Enabled)
		assert.Equal(t, []string{"updated"}, flag.Tags)
		assert.Equal(t, testFlag.CreatedAt, flag.CreatedAt, "CreatedAt should be preserved")
		assert.True(t, flag.UpdatedAt.After(testFlag.CreatedAt), "UpdatedAt should be more recent")

		// Test updating a non-existent flag
		err = provider.UpdateFlag(ctx, &feature.Flag{Name: "non-existent"})
		require.Error(t, err)
		assert.Equal(t, feature.ErrFlagNotFound, err)

		// Test invalid updates
		err = provider.UpdateFlag(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "flag cannot be nil")

		err = provider.UpdateFlag(ctx, &feature.Flag{Name: ""})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "flag name cannot be empty")
	})

	t.Run("DeleteFlag", func(t *testing.T) {
		t.Parallel()
		// Create a provider with a test flag
		testFlag := &feature.Flag{
			Name:    "delete-flag",
			Enabled: true,
		}
		provider, _ := feature.NewMemoryProvider(testFlag)

		// Delete the flag
		err := provider.DeleteFlag(ctx, "delete-flag")
		require.NoError(t, err)

		// Verify it's gone
		_, err = provider.GetFlag(ctx, "delete-flag")
		require.Error(t, err)
		assert.Equal(t, feature.ErrFlagNotFound, err)

		// Test deleting a non-existent flag
		err = provider.DeleteFlag(ctx, "non-existent")
		require.Error(t, err)
		assert.Equal(t, feature.ErrFlagNotFound, err)
	})

	t.Run("ListFlags", func(t *testing.T) {
		t.Parallel()
		// Create a provider with multiple flags
		flags := []*feature.Flag{
			{
				Name:    "flag1",
				Enabled: true,
				Tags:    []string{"production", "backend"},
			},
			{
				Name:    "flag2",
				Enabled: false,
				Tags:    []string{"test", "frontend"},
			},
			{
				Name:    "flag3",
				Enabled: true,
				Tags:    []string{"production", "frontend"},
			},
		}
		provider, _ := feature.NewMemoryProvider(flags...)

		// List all flags
		allFlags, err := provider.ListFlags(ctx)
		require.NoError(t, err)
		assert.Len(t, allFlags, 3)

		// List flags by tag
		prodFlags, err := provider.ListFlags(ctx, "production")
		require.NoError(t, err)
		assert.Len(t, prodFlags, 2)

		frontendFlags, err := provider.ListFlags(ctx, "frontend")
		require.NoError(t, err)
		assert.Len(t, frontendFlags, 2)

		// List flags by multiple tags
		combinedFlags, err := provider.ListFlags(ctx, "production", "test")
		require.NoError(t, err)
		assert.Len(t, combinedFlags, 3)

		// List flags with non-existent tag
		nonExistentFlags, err := provider.ListFlags(ctx, "non-existent")
		require.NoError(t, err)
		assert.Len(t, nonExistentFlags, 0)

		// Verify lists return copies, not original references
		flags[0].Enabled = false
		retrievedFlags, _ := provider.ListFlags(ctx)
		// Check that flag1 is still enabled in the retrieved list
		for _, flag := range retrievedFlags {
			if flag.Name == "flag1" {
				assert.True(t, flag.Enabled, "Retrieved flag should not be affected by external changes")
				break
			}
		}
	})

	t.Run("IsEnabled", func(t *testing.T) {
		t.Parallel()
		// Create flags with different strategies
		alwaysOnFlag := &feature.Flag{
			Name:     "always-on",
			Enabled:  true,
			Strategy: feature.NewAlwaysOnStrategy(),
		}

		alwaysOffFlag := &feature.Flag{
			Name:     "always-off",
			Enabled:  true,
			Strategy: feature.NewAlwaysOffStrategy(),
		}

		disabledFlag := &feature.Flag{
			Name:     "disabled-flag",
			Enabled:  false, // Globally disabled, strategy doesn't matter
			Strategy: feature.NewAlwaysOnStrategy(),
		}

		targetedFlag := &feature.Flag{
			Name:    "targeted-flag",
			Enabled: true,
			Strategy: feature.NewTargetedStrategy(feature.TargetCriteria{
				UserIDs: []string{"test-user"},
			}, feature.WithUserIDExtractor(testMemoryUserIDExtractor)),
		}

		provider, _ := feature.NewMemoryProvider(
			alwaysOnFlag, alwaysOffFlag, disabledFlag, targetedFlag,
		)

		// Test always-on flag
		enabled, err := provider.IsEnabled(ctx, "always-on")
		require.NoError(t, err)
		assert.True(t, enabled)

		// Test always-off flag
		enabled, err = provider.IsEnabled(ctx, "always-off")
		require.NoError(t, err)
		assert.False(t, enabled)

		// Test globally disabled flag
		enabled, err = provider.IsEnabled(ctx, "disabled-flag")
		require.NoError(t, err)
		assert.False(t, enabled)

		// Test targeted flag with matching user
		userCtx := context.WithValue(ctx, testMemoryUserIDKey{}, "test-user")
		enabled, err = provider.IsEnabled(userCtx, "targeted-flag")
		require.NoError(t, err)
		assert.True(t, enabled)

		// Test targeted flag with non-matching user
		userCtx = context.WithValue(ctx, testMemoryUserIDKey{}, "other-user")
		enabled, err = provider.IsEnabled(userCtx, "targeted-flag")
		require.NoError(t, err)
		assert.False(t, enabled)

		// Test non-existent flag
		_, err = provider.IsEnabled(ctx, "non-existent")
		require.Error(t, err)
		assert.Equal(t, feature.ErrFlagNotFound, err)
	})

	t.Run("Close", func(t *testing.T) {
		t.Parallel()
		provider, _ := feature.NewMemoryProvider()
		err := provider.Close()
		require.NoError(t, err)
	})
}
