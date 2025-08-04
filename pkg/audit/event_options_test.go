package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithMetadata(t *testing.T) {
	t.Parallel()

	t.Run("initializes metadata map if nil", func(t *testing.T) {
		t.Parallel()
		event := Event{Action: "test.action"}
		require.Nil(t, event.Metadata)

		option := WithMetadata("key", "value")
		option(&event)

		require.NotNil(t, event.Metadata)
		assert.Equal(t, "value", event.Metadata["key"])
	})

	t.Run("adds to existing metadata map", func(t *testing.T) {
		t.Parallel()
		event := Event{
			Action:   "test.action",
			Metadata: map[string]any{"existing": "value"},
		}

		option := WithMetadata("new", "data")
		option(&event)

		assert.Len(t, event.Metadata, 2)
		assert.Equal(t, "value", event.Metadata["existing"])
		assert.Equal(t, "data", event.Metadata["new"])
	})

	t.Run("overwrites existing metadata key", func(t *testing.T) {
		t.Parallel()
		event := Event{
			Action:   "test.action",
			Metadata: map[string]any{"key": "old-value"},
		}

		option := WithMetadata("key", "new-value")
		option(&event)

		assert.Len(t, event.Metadata, 1)
		assert.Equal(t, "new-value", event.Metadata["key"])
	})

	t.Run("can be called multiple times", func(t *testing.T) {
		t.Parallel()
		event := Event{Action: "test.action"}

		option1 := WithMetadata("first", "value1")
		option2 := WithMetadata("second", "value2")
		option3 := WithMetadata("third", "value3")

		option1(&event)
		option2(&event)
		option3(&event)

		require.NotNil(t, event.Metadata)
		assert.Len(t, event.Metadata, 3)
		assert.Equal(t, "value1", event.Metadata["first"])
		assert.Equal(t, "value2", event.Metadata["second"])
		assert.Equal(t, "value3", event.Metadata["third"])
	})
}

func TestEventOptions_Composition(t *testing.T) {
	t.Parallel()

	t.Run("multiple options can be applied together", func(t *testing.T) {
		t.Parallel()
		event := Event{Action: "test.action"}

		options := []EventOption{
			WithResource("document", "doc-123"),
			WithMetadata("size", 1024),
			WithMetadata("type", "pdf"),
			WithResult(ResultSuccess),
		}

		for _, opt := range options {
			opt(&event)
		}

		assert.Equal(t, "document", event.Resource)
		assert.Equal(t, "doc-123", event.ResourceID)
		assert.Equal(t, ResultSuccess, event.Result)
		require.NotNil(t, event.Metadata)
		assert.Equal(t, 1024, event.Metadata["size"])
		assert.Equal(t, "pdf", event.Metadata["type"])
	})

	t.Run("options applied in order", func(t *testing.T) {
		t.Parallel()
		event := Event{Action: "test.action"}

		// Test that last option wins for conflicting fields
		options := []EventOption{
			WithResult(ResultSuccess),
			WithResult(ResultFailure),
			WithResult(ResultError), // This should be the final value
			WithResource("first", "id1"),
			WithResource("second", "id2"), // This should be the final value
		}

		for _, opt := range options {
			opt(&event)
		}

		assert.Equal(t, ResultError, event.Result)
		assert.Equal(t, "second", event.Resource)
		assert.Equal(t, "id2", event.ResourceID)
	})

	t.Run("metadata options are cumulative", func(t *testing.T) {
		t.Parallel()
		event := Event{Action: "test.action"}

		options := []EventOption{
			WithMetadata("key1", "value1"),
			WithMetadata("key2", "value2"),
			WithMetadata("key1", "updated-value1"), // Should overwrite key1
			WithMetadata("key3", "value3"),
		}

		for _, opt := range options {
			opt(&event)
		}

		require.NotNil(t, event.Metadata)
		assert.Len(t, event.Metadata, 3)
		assert.Equal(t, "updated-value1", event.Metadata["key1"])
		assert.Equal(t, "value2", event.Metadata["key2"])
		assert.Equal(t, "value3", event.Metadata["key3"])
	})
}
