package ratelimit

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("panic on nil keyFunc", func(t *testing.T) {
		store := NewMemoryStore()
		limiter, _ := NewTokenBucket(store, 10, time.Second)

		assert.Panics(t, func() {
			Middleware(limiter, nil)
		})
	})

	t.Run("rate limit headers set correctly", func(t *testing.T) {
		t.Parallel()

		store := NewMemoryStore()
		limiter, _ := NewTokenBucket(store, 5, time.Second)
		keyFunc := func(r *http.Request) string { return "test-key" }

		handler := Middleware(limiter, keyFunc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "5", rec.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "4", rec.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
	})

	t.Run("429 status when rate limit exceeded", func(t *testing.T) {
		t.Parallel()

		store := NewMemoryStore()
		limiter, _ := NewTokenBucket(store, 2, time.Second)
		keyFunc := func(r *http.Request) string { return "test-key-429" }

		handler := Middleware(limiter, keyFunc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		for range 2 {
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		}

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		assert.Equal(t, "Too Many Requests\n", rec.Body.String())
		assert.NotEmpty(t, rec.Header().Get("Retry-After"))
		assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))

		retryAfter, err := strconv.Atoi(rec.Header().Get("Retry-After"))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, retryAfter, 1)
	})

	t.Run("empty key bypasses rate limiting", func(t *testing.T) {
		t.Parallel()

		store := NewMemoryStore()
		limiter, _ := NewTokenBucket(store, 1, time.Second)
		keyFunc := func(r *http.Request) string { return "" }

		handler := Middleware(limiter, keyFunc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		for range 5 {
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Empty(t, rec.Header().Get("X-RateLimit-Limit"))
		}
	})

	t.Run("fail-open on limiter error", func(t *testing.T) {
		t.Parallel()

		failingLimiter := &failingLimiter{err: errors.New("storage error")}
		keyFunc := func(r *http.Request) string { return "test-key" }

		handler := Middleware(failingLimiter, keyFunc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Empty(t, rec.Header().Get("X-RateLimit-Limit"))
	})
}

func TestMiddlewareWithOptions(t *testing.T) {
	t.Parallel()

	t.Run("panic on nil keyFunc", func(t *testing.T) {
		store := NewMemoryStore()
		limiter, _ := NewTokenBucket(store, 10, time.Second)

		assert.Panics(t, func() {
			MiddlewareWithOptions(limiter, nil)
		})
	})

	t.Run("custom key function", func(t *testing.T) {
		t.Parallel()

		store := NewMemoryStore()
		limiter, _ := NewTokenBucket(store, 5, time.Second)

		customKeyFunc := func(r *http.Request) string {
			return r.Header.Get("X-API-Key")
		}

		handler := MiddlewareWithOptions(
			limiter,
			func(r *http.Request) string { return "default" },
			WithKeyFunc(customKeyFunc),
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.Header.Set("X-API-Key", "key1")
		rec1 := httptest.NewRecorder()
		handler.ServeHTTP(rec1, req1)
		assert.Equal(t, "4", rec1.Header().Get("X-RateLimit-Remaining"))

		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.Header.Set("X-API-Key", "key2")
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req2)
		assert.Equal(t, "4", rec2.Header().Get("X-RateLimit-Remaining"))
	})

	t.Run("custom on limit reached handler", func(t *testing.T) {
		t.Parallel()

		store := NewMemoryStore()
		limiter, _ := NewTokenBucket(store, 1, time.Second)

		customHandlerCalled := false
		customHandler := func(w http.ResponseWriter, r *http.Request, result *Result) {
			customHandlerCalled = true
			w.Header().Set("X-Custom-Header", "rate-limited")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Custom rate limit message"))
		}

		handler := MiddlewareWithOptions(
			limiter,
			func(r *http.Request) string { return "test-key-custom" },
			WithOnLimitReached(customHandler),
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		req = httptest.NewRequest("GET", "/test", nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.True(t, customHandlerCalled)
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.Equal(t, "rate-limited", rec.Header().Get("X-Custom-Header"))
		assert.Equal(t, "Custom rate limit message", rec.Body.String())
	})

	t.Run("skip function", func(t *testing.T) {
		t.Parallel()

		store := NewMemoryStore()
		limiter, _ := NewTokenBucket(store, 1, time.Second)

		skipFunc := func(r *http.Request) bool {
			return r.Header.Get("X-Skip-RateLimit") == "true"
		}

		handler := MiddlewareWithOptions(
			limiter,
			func(r *http.Request) string { return "test-key-skip" },
			WithSkipFunc(skipFunc),
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))

		req = httptest.NewRequest("GET", "/test", nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)

		req = httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Skip-RateLimit", "true")
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Empty(t, rec.Header().Get("X-RateLimit-Limit"))
	})

	t.Run("multiple options combined", func(t *testing.T) {
		t.Parallel()

		store := NewMemoryStore()
		limiter, _ := NewTokenBucket(store, 2, time.Second)

		handler := MiddlewareWithOptions(
			limiter,
			func(r *http.Request) string { return "default" },
			WithKeyFunc(func(r *http.Request) string {
				return r.Header.Get("X-User-ID")
			}),
			WithSkipFunc(func(r *http.Request) bool {
				return r.Header.Get("X-Admin") == "true"
			}),
			WithOnLimitReached(func(w http.ResponseWriter, r *http.Request, result *Result) {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("Forbidden"))
			}),
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Admin", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		for range 2 {
			req = httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-User-ID", "user1")
			rec = httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		}

		req = httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-ID", "user1")
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Equal(t, "Forbidden", rec.Body.String())
	})
}

func TestHandlerFunc(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	limiter, _ := NewTokenBucket(store, 3, time.Second, WithBurst(3))

	// Pre-fill the token bucket
	ctx := context.Background()
	_, _, _ = store.IncrementAndGet(ctx, "test-handler-func-key", 3, time.Second)

	keyFunc := func(r *http.Request) string { return "test-handler-func-key" }

	handler := HandlerFunc(limiter, keyFunc, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	for i := range 3 {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		handler(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "OK", rec.Body.String())
		assert.Equal(t, strconv.Itoa(2-i), rec.Header().Get("X-RateLimit-Remaining"))
	}

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestPerEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("panic on nil keyFunc in config", func(t *testing.T) {
		store := NewMemoryStore()
		limiter, _ := NewTokenBucket(store, 10, time.Second)

		configs := []EndpointConfig{
			{
				Path:    "/api/users",
				Limiter: limiter,
				KeyFunc: nil,
			},
		}

		assert.Panics(t, func() {
			PerEndpoint(configs, limiter, func(r *http.Request) string { return "default" })
		})
	})

	t.Run("different limits per endpoint", func(t *testing.T) {
		t.Parallel()

		// Use separate stores for different limiters to avoid shared state
		strictStore := NewMemoryStore()
		strictLimiter, _ := NewTokenBucket(strictStore, 2, time.Second, WithBurst(2))

		relaxedStore := NewMemoryStore()
		relaxedLimiter, _ := NewTokenBucket(relaxedStore, 10, time.Second, WithBurst(10))

		defaultStore := NewMemoryStore()
		defaultLimiter, _ := NewTokenBucket(defaultStore, 5, time.Second, WithBurst(5))

		// Pre-fill token buckets
		ctx := context.Background()
		_, _, _ = strictStore.IncrementAndGet(ctx, "strict-key", 2, time.Second)
		_, _, _ = relaxedStore.IncrementAndGet(ctx, "relaxed-key", 10, time.Second)
		_, _, _ = defaultStore.IncrementAndGet(ctx, "default-key", 5, time.Second)

		configs := []EndpointConfig{
			{
				Path:    "/api/strict",
				Limiter: strictLimiter,
				KeyFunc: func(r *http.Request) string { return "strict-key" },
			},
			{
				Path:    "/api/relaxed",
				Limiter: relaxedLimiter,
				KeyFunc: func(r *http.Request) string { return "relaxed-key" },
			},
		}

		handler := PerEndpoint(
			configs,
			defaultLimiter,
			func(r *http.Request) string { return "default-key" },
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/api/strict", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, "2", rec.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "1", rec.Header().Get("X-RateLimit-Remaining"))

		req = httptest.NewRequest("GET", "/api/relaxed", nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, "10", rec.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "9", rec.Header().Get("X-RateLimit-Remaining"))

		req = httptest.NewRequest("GET", "/api/other", nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, "5", rec.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "4", rec.Header().Get("X-RateLimit-Remaining"))
	})

	t.Run("endpoint-specific rate limit enforcement", func(t *testing.T) {
		t.Parallel()

		store := NewMemoryStore()
		limiter, _ := NewTokenBucket(store, 2, time.Second, WithBurst(2))

		// Pre-fill the token bucket to ensure we start with full capacity
		ctx := context.Background()
		_, _, _ = store.IncrementAndGet(ctx, "limited-endpoint", 2, time.Second)

		configs := []EndpointConfig{
			{
				Path:    "/api/limited",
				Limiter: limiter,
				KeyFunc: func(r *http.Request) string { return "limited-endpoint" },
			},
		}

		handler := PerEndpoint(
			configs,
			nil,
			nil,
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// First request should succeed
		req := httptest.NewRequest("GET", "/api/limited", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		// Second request should also succeed (limit is 2)
		req = httptest.NewRequest("GET", "/api/limited", nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		// Third request should be rate limited
		req = httptest.NewRequest("GET", "/api/limited", nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		assert.Contains(t, rec.Body.String(), "Rate limit exceeded for /api/limited")
	})

	t.Run("nil limiter or keyFunc bypasses", func(t *testing.T) {
		t.Parallel()

		handler := PerEndpoint(
			[]EndpointConfig{},
			nil,
			nil,
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		for range 10 {
			req := httptest.NewRequest("GET", "/api/test", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("fail-open on limiter error", func(t *testing.T) {
		t.Parallel()

		failingLimiter := &failingLimiter{err: errors.New("storage error")}

		configs := []EndpointConfig{
			{
				Path:    "/api/failing",
				Limiter: failingLimiter,
				KeyFunc: func(r *http.Request) string { return "key" },
			},
		}

		handler := PerEndpoint(
			configs,
			nil,
			nil,
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/api/failing", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestMiddleware_RealWorldScenarios(t *testing.T) {
	t.Parallel()

	t.Run("API protection with burst handling", func(t *testing.T) {
		t.Parallel()

		store := NewMemoryStore()
		limiter, _ := NewTokenBucket(store, 10, 100*time.Millisecond, WithBurst(20))

		// Pre-fill the token bucket with burst capacity
		ctx := context.Background()
		_, _, _ = store.IncrementAndGet(ctx, "192.168.1.1:8080", 20, 100*time.Millisecond)

		handler := Middleware(
			limiter,
			func(r *http.Request) string { return r.RemoteAddr },
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		for range 20 {
			req := httptest.NewRequest("GET", "/api/resource", nil)
			req.RemoteAddr = "192.168.1.1:8080"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		}

		req := httptest.NewRequest("GET", "/api/resource", nil)
		req.RemoteAddr = "192.168.1.1:8080"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)

		time.Sleep(110 * time.Millisecond)

		for range 10 {
			req := httptest.NewRequest("GET", "/api/resource", nil)
			req.RemoteAddr = "192.168.1.1:8080"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("authenticated vs unauthenticated rate limits", func(t *testing.T) {
		t.Parallel()

		// Use separate stores to avoid shared state
		authStore := NewMemoryStore()
		authLimiter, _ := NewTokenBucket(authStore, 100, time.Second, WithBurst(100))

		unauthStore := NewMemoryStore()
		unauthLimiter, _ := NewTokenBucket(unauthStore, 10, time.Second, WithBurst(10))

		// Pre-fill token buckets
		ctx := context.Background()
		_, _, _ = unauthStore.IncrementAndGet(ctx, "ip:192.168.1.1:8080", 10, time.Second)
		_, _, _ = authStore.IncrementAndGet(ctx, "user:user123", 100, time.Second)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var limiter Limiter
			var key string

			if userID := r.Header.Get("X-User-ID"); userID != "" {
				limiter = authLimiter
				key = "user:" + userID
			} else {
				limiter = unauthLimiter
				key = "ip:" + r.RemoteAddr
			}

			result, _ := limiter.Allow(r.Context(), key)

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))

			if !result.Allowed {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}

			handler.ServeHTTP(w, r)
		})

		// Unauthenticated requests - should allow 10
		for range 10 {
			req := httptest.NewRequest("GET", "/api/data", nil)
			req.RemoteAddr = "192.168.1.1:8080"
			rec := httptest.NewRecorder()
			protectedHandler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		}

		// 11th unauthenticated request should be denied
		req := httptest.NewRequest("GET", "/api/data", nil)
		req.RemoteAddr = "192.168.1.1:8080"
		rec := httptest.NewRecorder()
		protectedHandler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)

		// Authenticated requests - should allow more
		for range 20 {
			req := httptest.NewRequest("GET", "/api/data", nil)
			req.Header.Set("X-User-ID", "user123")
			rec := httptest.NewRecorder()
			protectedHandler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})
}

type failingLimiter struct {
	err error
}

func (f *failingLimiter) Allow(ctx context.Context, key string) (*Result, error) {
	return nil, f.err
}

func (f *failingLimiter) AllowN(ctx context.Context, key string, n int) (*Result, error) {
	return nil, f.err
}

func (f *failingLimiter) Status(ctx context.Context, key string) (*Result, error) {
	return nil, f.err
}

func (f *failingLimiter) Reset(ctx context.Context, key string) error {
	return f.err
}
