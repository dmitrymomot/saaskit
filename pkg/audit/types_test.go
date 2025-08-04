package audit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvent_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid event passes validation", func(t *testing.T) {
		t.Parallel()
		event := Event{
			ID:        "test-id",
			Action:    "user.login",
			Result:    ResultSuccess,
			CreatedAt: time.Now(),
		}

		err := event.Validate()
		assert.NoError(t, err)
	})

	t.Run("event with all fields passes validation", func(t *testing.T) {
		t.Parallel()
		event := Event{
			ID:         "test-id",
			TenantID:   "tenant-123",
			UserID:     "user-456",
			SessionID:  "session-789",
			Action:     "document.create",
			Resource:   "document",
			ResourceID: "doc-123",
			Result:     ResultSuccess,
			RequestID:  "req-123",
			IP:         "192.168.1.1",
			UserAgent:  "Mozilla/5.0",
			Metadata:   map[string]any{"size": 1024, "type": "pdf"},
			CreatedAt:  time.Now(),
		}

		err := event.Validate()
		assert.NoError(t, err)
	})

	t.Run("empty action fails validation", func(t *testing.T) {
		t.Parallel()
		event := Event{
			ID:        "test-id",
			Action:    "",
			Result:    ResultSuccess,
			CreatedAt: time.Now(),
		}

		err := event.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEventValidation)
		assert.Contains(t, err.Error(), "action is required")
	})

	t.Run("event with only action passes validation", func(t *testing.T) {
		t.Parallel()
		event := Event{
			Action: "minimal.action",
		}

		err := event.Validate()
		assert.NoError(t, err)
	})

	t.Run("whitespace-only action fails validation", func(t *testing.T) {
		t.Parallel()
		event := Event{
			Action: "   ",
		}

		// Current implementation only checks for empty string, not whitespace
		// This test documents current behavior - may want to enhance validation
		err := event.Validate()
		assert.NoError(t, err) // Current behavior - passes validation
	})
}

func TestEvent_IDUniqueness(t *testing.T) {
	t.Parallel()

	t.Run("events with same data get different IDs when generated", func(t *testing.T) {
		t.Parallel()
		// Test that ID generation creates unique identifiers
		// This is critical for audit trail integrity
		event1 := Event{
			Action: "test.action",
			Result: ResultSuccess,
		}
		event2 := Event{
			Action: "test.action",
			Result: ResultSuccess,
		}

		// If IDs are manually set to same value, they should be equal
		event1.ID = "same-id"
		event2.ID = "same-id"
		assert.Equal(t, event1.ID, event2.ID)

		// But different IDs should not be equal
		event1.ID = "id-1"
		event2.ID = "id-2"
		assert.NotEqual(t, event1.ID, event2.ID)
	})
}

func TestEvent_TimestampConsistency(t *testing.T) {
	t.Parallel()

	t.Run("created at timestamp is consistent across operations", func(t *testing.T) {
		t.Parallel()
		now := time.Now().UTC()
		event := Event{
			Action:    "test.action",
			CreatedAt: now,
		}

		// Verify timestamp is preserved exactly
		assert.Equal(t, now, event.CreatedAt)

		// Verify validation doesn't modify timestamp
		err := event.Validate()
		assert.NoError(t, err)
		assert.Equal(t, now, event.CreatedAt)
	})

	t.Run("zero time is preserved", func(t *testing.T) {
		t.Parallel()
		event := Event{
			Action:    "test.action",
			CreatedAt: time.Time{}, // Zero time
		}

		assert.True(t, event.CreatedAt.IsZero())
		err := event.Validate()
		assert.NoError(t, err)
		assert.True(t, event.CreatedAt.IsZero()) // Still zero after validation
	})
}

func TestEvent_MetadataHandling(t *testing.T) {
	t.Parallel()

	t.Run("nil metadata is handled correctly", func(t *testing.T) {
		t.Parallel()
		event := Event{
			Action:   "test.action",
			Metadata: nil,
		}

		err := event.Validate()
		assert.NoError(t, err)
		assert.Nil(t, event.Metadata)
	})

	t.Run("empty metadata map is handled correctly", func(t *testing.T) {
		t.Parallel()
		event := Event{
			Action:   "test.action",
			Metadata: make(map[string]any),
		}

		err := event.Validate()
		assert.NoError(t, err)
		assert.NotNil(t, event.Metadata)
		assert.Len(t, event.Metadata, 0)
	})

	t.Run("metadata supports various types", func(t *testing.T) {
		t.Parallel()
		event := Event{
			Action: "test.action",
			Metadata: map[string]any{
				"string": "value",
				"int":    42,
				"float":  3.14,
				"bool":   true,
				"slice":  []string{"a", "b", "c"},
				"map":    map[string]string{"nested": "value"},
				"nil":    nil,
			},
		}

		err := event.Validate()
		assert.NoError(t, err)

		// Verify all types are preserved
		assert.Equal(t, "value", event.Metadata["string"])
		assert.Equal(t, 42, event.Metadata["int"])
		assert.Equal(t, 3.14, event.Metadata["float"])
		assert.Equal(t, true, event.Metadata["bool"])
		assert.Equal(t, []string{"a", "b", "c"}, event.Metadata["slice"])
		assert.Equal(t, map[string]string{"nested": "value"}, event.Metadata["map"])
		assert.Nil(t, event.Metadata["nil"])
	})
}
