package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/handler"
)

func TestContextKey_String(t *testing.T) {
	t.Parallel()

	key := handler.NewContextKey("test-key")
	assert.Equal(t, "test-key", key.String())
}

func TestContextValue(t *testing.T) {
	t.Parallel()

	type user struct {
		ID   int
		Name string
	}

	t.Run("string value", func(t *testing.T) {
		t.Parallel()
		key := handler.NewContextKey("test")
		ctx := context.WithValue(context.Background(), key, "hello")

		got := handler.ContextValue[string](ctx, key)
		assert.Equal(t, "hello", got)
	})

	t.Run("struct value", func(t *testing.T) {
		t.Parallel()
		key := handler.NewContextKey("user")
		u := user{ID: 123, Name: "Alice"}
		ctx := context.WithValue(context.Background(), key, u)

		got := handler.ContextValue[user](ctx, key)
		assert.Equal(t, u, got)
	})

	t.Run("pointer value", func(t *testing.T) {
		t.Parallel()
		key := handler.NewContextKey("user")
		u := &user{ID: 456, Name: "Bob"}
		ctx := context.WithValue(context.Background(), key, u)

		got := handler.ContextValue[*user](ctx, key)
		require.NotNil(t, got)
		assert.Equal(t, u, got)
	})

	t.Run("slice value", func(t *testing.T) {
		t.Parallel()
		key := handler.NewContextKey("ids")
		ids := []int{1, 2, 3}
		ctx := context.WithValue(context.Background(), key, ids)

		got := handler.ContextValue[[]int](ctx, key)
		assert.Equal(t, ids, got)
	})

	t.Run("missing key returns zero value", func(t *testing.T) {
		t.Parallel()
		key := handler.NewContextKey("missing")
		ctx := context.Background()

		got := handler.ContextValue[string](ctx, key)
		assert.Empty(t, got)
	})

	t.Run("wrong type returns zero value", func(t *testing.T) {
		t.Parallel()
		key := handler.NewContextKey("number")
		ctx := context.WithValue(context.Background(), key, "not-a-number")

		got := handler.ContextValue[int](ctx, key)
		assert.Zero(t, got)
	})

	t.Run("nil pointer value", func(t *testing.T) {
		t.Parallel()
		key := handler.NewContextKey("user")
		ctx := context.WithValue(context.Background(), key, (*user)(nil))

		got := handler.ContextValue[*user](ctx, key)
		assert.Nil(t, got)
	})
}

func TestContextValueOK(t *testing.T) {
	t.Parallel()

	type user struct {
		ID   int
		Name string
	}

	t.Run("present value with correct type", func(t *testing.T) {
		t.Parallel()
		key := handler.NewContextKey("test")
		ctx := context.WithValue(context.Background(), key, "hello")

		got, ok := handler.ContextValueOK[string](ctx, key)
		assert.True(t, ok)
		assert.Equal(t, "hello", got)
	})

	t.Run("missing key", func(t *testing.T) {
		t.Parallel()
		key := handler.NewContextKey("missing")
		ctx := context.Background()

		got, ok := handler.ContextValueOK[string](ctx, key)
		assert.False(t, ok)
		assert.Empty(t, got)
	})

	t.Run("wrong type", func(t *testing.T) {
		t.Parallel()
		key := handler.NewContextKey("number")
		ctx := context.WithValue(context.Background(), key, "not-a-number")

		got, ok := handler.ContextValueOK[int](ctx, key)
		assert.False(t, ok)
		assert.Zero(t, got)
	})

	t.Run("zero value vs missing key", func(t *testing.T) {
		t.Parallel()
		key := handler.NewContextKey("count")
		ctx := context.WithValue(context.Background(), key, 0)

		// With ContextValue, can't tell if missing or zero
		val1 := handler.ContextValue[int](ctx, key)
		assert.Equal(t, 0, val1)

		// With ContextValueOK, can distinguish
		val2, ok := handler.ContextValueOK[int](ctx, key)
		assert.True(t, ok)
		assert.Equal(t, 0, val2)

		// Missing key
		missingKey := handler.NewContextKey("missing")
		val3, ok := handler.ContextValueOK[int](ctx, missingKey)
		assert.False(t, ok)
		assert.Equal(t, 0, val3)
	})

	t.Run("pointer types", func(t *testing.T) {
		t.Parallel()
		key := handler.NewContextKey("user")
		u := &user{ID: 123, Name: "Alice"}
		ctx := context.WithValue(context.Background(), key, u)

		got, ok := handler.ContextValueOK[*user](ctx, key)
		assert.True(t, ok)
		assert.Equal(t, u, got)
	})

	t.Run("interface types", func(t *testing.T) {
		t.Parallel()
		key := handler.NewContextKey("error")
		err := context.DeadlineExceeded
		ctx := context.WithValue(context.Background(), key, err)

		got, ok := handler.ContextValueOK[error](ctx, key)
		assert.True(t, ok)
		assert.Equal(t, err, got)
	})
}

func TestContext_SSE(t *testing.T) {
	t.Parallel()

	t.Run("SSE request initializes SSE generator", func(t *testing.T) {
		t.Parallel()
		// Create a request with SSE Accept header
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/event-stream")
		w := httptest.NewRecorder()

		ctx := handler.NewContext(w, req)

		sse := ctx.SSE()
		assert.NotNil(t, sse, "SSE should be initialized for SSE requests")
	})

	t.Run("non-SSE request returns nil", func(t *testing.T) {
		t.Parallel()
		// Regular request without SSE headers
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		ctx := handler.NewContext(w, req)

		sse := ctx.SSE()
		assert.Nil(t, sse, "SSE should be nil for non-SSE requests")
	})

	t.Run("SSE request with datastar query param", func(t *testing.T) {
		t.Parallel()
		// Request with datastar query parameter
		req := httptest.NewRequest(http.MethodGet, "/test?datastar=true", nil)
		w := httptest.NewRecorder()

		ctx := handler.NewContext(w, req)

		sse := ctx.SSE()
		assert.NotNil(t, sse, "SSE should be initialized for requests with datastar query param")
	})

	t.Run("SSE functionality works", func(t *testing.T) {
		t.Parallel()
		// Create SSE request
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept", "text/event-stream")
		w := httptest.NewRecorder()

		ctx := handler.NewContext(w, req)
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
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		// Set a deadline on the request context
		reqCtx, cancel := context.WithCancel(req.Context())
		defer cancel()
		req = req.WithContext(reqCtx)

		ctx := handler.NewContext(w, req)

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
		key := handler.NewContextKey("test")
		reqWithValue := req.WithContext(context.WithValue(req.Context(), key, "test-value"))
		ctxWithValue := handler.NewContext(w, reqWithValue)
		assert.Equal(t, "test-value", ctxWithValue.Value(key))
	})
}
