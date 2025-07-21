package saaskit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func TestContext_SSE(t *testing.T) {
	t.Run("SSE request initializes SSE generator", func(t *testing.T) {
		// Create a request with SSE Accept header
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/event-stream")
		w := httptest.NewRecorder()

		ctx := saaskit.NewContext(w, req)

		sse := ctx.SSE()
		assert.NotNil(t, sse, "SSE should be initialized for SSE requests")
	})

	t.Run("non-SSE request returns nil", func(t *testing.T) {
		// Regular request without SSE headers
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		ctx := saaskit.NewContext(w, req)

		sse := ctx.SSE()
		assert.Nil(t, sse, "SSE should be nil for non-SSE requests")
	})

	t.Run("SSE request with datastar query param", func(t *testing.T) {
		// Request with datastar query parameter
		req := httptest.NewRequest(http.MethodGet, "/test?datastar=true", nil)
		w := httptest.NewRecorder()

		ctx := saaskit.NewContext(w, req)

		sse := ctx.SSE()
		assert.NotNil(t, sse, "SSE should be initialized for requests with datastar query param")
	})

	t.Run("SSE functionality works", func(t *testing.T) {
		// Create SSE request
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/event-stream")
		w := httptest.NewRecorder()

		ctx := saaskit.NewContext(w, req)
		sse := ctx.SSE()
		require.NotNil(t, sse)

		// Test that we can use the SSE generator - test Redirect method
		err := sse.Redirect("/test-redirect")
		assert.NoError(t, err)

		// Check response headers were set correctly
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	})

	t.Run("Context implements all interface methods", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		// Set a deadline on the request context
		reqCtx, cancel := context.WithCancel(req.Context())
		defer cancel()
		req = req.WithContext(reqCtx)

		ctx := saaskit.NewContext(w, req)

		// Test Context interface methods
		assert.Equal(t, req, ctx.Request())
		assert.Equal(t, w, ctx.ResponseWriter())

		// Test context.Context interface methods
		assert.Equal(t, ctx.Done(), req.Context().Done())
		assert.Equal(t, ctx.Err(), req.Context().Err())

		deadline, ok := ctx.Deadline()
		expectedDeadline, expectedOk := req.Context().Deadline()
		assert.Equal(t, expectedDeadline, deadline)
		assert.Equal(t, expectedOk, ok)

		// Test Value method
		key := saaskit.NewContextKey("test")
		reqWithValue := req.WithContext(context.WithValue(req.Context(), key, "test-value"))
		ctxWithValue := saaskit.NewContext(w, reqWithValue)
		assert.Equal(t, "test-value", ctxWithValue.Value(key))
	})
}
