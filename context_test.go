package saaskit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit"
)

func TestContextValue(t *testing.T) {
	type user struct {
		ID   int
		Name string
	}

	t.Run("string value", func(t *testing.T) {
		key := saaskit.NewContextKey("test")
		ctx := context.WithValue(context.Background(), key, "hello")

		got := saaskit.ContextValue[string](ctx, key)
		assert.Equal(t, "hello", got)
	})

	t.Run("struct value", func(t *testing.T) {
		key := saaskit.NewContextKey("user")
		u := user{ID: 123, Name: "Alice"}
		ctx := context.WithValue(context.Background(), key, u)

		got := saaskit.ContextValue[user](ctx, key)
		assert.Equal(t, u, got)
	})

	t.Run("pointer value", func(t *testing.T) {
		key := saaskit.NewContextKey("user")
		u := &user{ID: 456, Name: "Bob"}
		ctx := context.WithValue(context.Background(), key, u)

		got := saaskit.ContextValue[*user](ctx, key)
		require.NotNil(t, got)
		assert.Equal(t, u, got)
	})

	t.Run("slice value", func(t *testing.T) {
		key := saaskit.NewContextKey("ids")
		ids := []int{1, 2, 3}
		ctx := context.WithValue(context.Background(), key, ids)

		got := saaskit.ContextValue[[]int](ctx, key)
		assert.Equal(t, ids, got)
	})

	t.Run("missing key returns zero value", func(t *testing.T) {
		key := saaskit.NewContextKey("missing")
		ctx := context.Background()

		got := saaskit.ContextValue[string](ctx, key)
		assert.Empty(t, got)
	})

	t.Run("wrong type returns zero value", func(t *testing.T) {
		key := saaskit.NewContextKey("number")
		ctx := context.WithValue(context.Background(), key, "not-a-number")

		got := saaskit.ContextValue[int](ctx, key)
		assert.Zero(t, got)
	})

	t.Run("nil pointer value", func(t *testing.T) {
		key := saaskit.NewContextKey("user")
		ctx := context.WithValue(context.Background(), key, (*user)(nil))

		got := saaskit.ContextValue[*user](ctx, key)
		assert.Nil(t, got)
	})
}
