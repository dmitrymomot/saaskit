package ratelimiter_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/ratelimiter"
)

func TestMiddleware_RateLimitEnforcement(t *testing.T) {
	t.Parallel()

	config := ratelimiter.Config{
		Capacity:       3,
		RefillRate:     1,
		RefillInterval: 100 * time.Millisecond,
	}

	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	limiter, err := ratelimiter.NewTokenBucket(store, config)
	require.NoError(t, err)

	keyFunc := func(r *http.Request) string {
		return r.RemoteAddr
	}

	middleware := ratelimiter.Middleware(limiter, keyFunc)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	t.Run("allows requests within limit", func(t *testing.T) {
		for i := range config.Capacity {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "OK", rec.Body.String())

			assert.Equal(t, strconv.Itoa(config.Capacity), rec.Header().Get("X-RateLimit-Limit"))
			assert.Equal(t, strconv.Itoa(config.Capacity-i-1), rec.Header().Get("X-RateLimit-Remaining"))
			assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		assert.Contains(t, rec.Body.String(), "Too Many Requests")
		assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
		// Retry-After may or may not be set depending on timing
		retryAfter := rec.Header().Get("Retry-After")
		if retryAfter != "" {
			retrySeconds, err := strconv.Atoi(retryAfter)
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, retrySeconds, 0)
		}
	})

	t.Run("different keys have independent limits", func(t *testing.T) {
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "192.168.1.2:5678"
		rec1 := httptest.NewRecorder()

		handler.ServeHTTP(rec1, req1)

		assert.Equal(t, http.StatusOK, rec1.Code)
		assert.Equal(t, strconv.Itoa(config.Capacity-1), rec1.Header().Get("X-RateLimit-Remaining"))

		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "192.168.1.3:9012"
		rec2 := httptest.NewRecorder()

		handler.ServeHTTP(rec2, req2)

		assert.Equal(t, http.StatusOK, rec2.Code)
		assert.Equal(t, strconv.Itoa(config.Capacity-1), rec2.Header().Get("X-RateLimit-Remaining"))
	})
}

func TestMiddleware_Headers(t *testing.T) {
	t.Parallel()

	config := ratelimiter.Config{
		Capacity:       10,
		RefillRate:     2,
		RefillInterval: time.Second,
	}

	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	limiter, err := ratelimiter.NewTokenBucket(store, config)
	require.NoError(t, err)

	keyFunc := func(r *http.Request) string {
		return "test-key"
	}

	middleware := ratelimiter.Middleware(limiter, keyFunc)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("sets correct headers on allowed request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "10", rec.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "9", rec.Header().Get("X-RateLimit-Remaining"))

		resetTime := rec.Header().Get("X-RateLimit-Reset")
		assert.NotEmpty(t, resetTime)

		resetUnix, err := strconv.ParseInt(resetTime, 10, 64)
		assert.NoError(t, err)
		assert.Greater(t, resetUnix, time.Now().Unix())
	})

	t.Run("sets zero remaining when exhausted", func(t *testing.T) {
		for range config.Capacity + 1 {
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		assert.Equal(t, "10", rec.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
	})
}

func TestMiddleware_CustomErrorResponder(t *testing.T) {
	t.Parallel()

	config := ratelimiter.Config{
		Capacity:       1,
		RefillRate:     1,
		RefillInterval: time.Second,
	}

	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	limiter, err := ratelimiter.NewTokenBucket(store, config)
	require.NoError(t, err)

	keyFunc := func(r *http.Request) string {
		return "test"
	}

	customResponder := func(w http.ResponseWriter, r *http.Request, result *ratelimiter.Result, err error) {
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Service Unavailable"))
			return
		}

		if result != nil && !result.Allowed() {
			w.Header().Set("X-Custom-Header", "rate-limited")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Custom Rate Limit Message"))
		}
	}

	middleware := ratelimiter.Middleware(
		limiter,
		keyFunc,
		ratelimiter.WithErrorResponder(customResponder),
	)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("uses custom error responder", func(t *testing.T) {
		req1 := httptest.NewRequest("GET", "/test", nil)
		rec1 := httptest.NewRecorder()
		handler.ServeHTTP(rec1, req1)
		assert.Equal(t, http.StatusOK, rec1.Code)

		req2 := httptest.NewRequest("GET", "/test", nil)
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req2)

		assert.Equal(t, http.StatusForbidden, rec2.Code)
		assert.Equal(t, "Custom Rate Limit Message", rec2.Body.String())
		assert.Equal(t, "rate-limited", rec2.Header().Get("X-Custom-Header"))
	})
}

func TestComposite_KeyFunction(t *testing.T) {
	t.Parallel()

	t.Run("combines multiple keys", func(t *testing.T) {
		keyFunc1 := func(r *http.Request) string {
			return r.Header.Get("X-User-ID")
		}
		keyFunc2 := func(r *http.Request) string {
			return r.URL.Path
		}
		keyFunc3 := func(r *http.Request) string {
			return r.Method
		}

		composite := ratelimiter.Composite(keyFunc1, keyFunc2, keyFunc3)

		req := httptest.NewRequest("POST", "/api/users", nil)
		req.Header.Set("X-User-ID", "user123")

		key := composite(req)
		assert.Equal(t, "user123:/api/users:POST", key)
	})

	t.Run("skips empty keys", func(t *testing.T) {
		keyFunc1 := func(r *http.Request) string {
			return ""
		}
		keyFunc2 := func(r *http.Request) string {
			return "key2"
		}
		keyFunc3 := func(r *http.Request) string {
			return ""
		}
		keyFunc4 := func(r *http.Request) string {
			return "key4"
		}

		composite := ratelimiter.Composite(keyFunc1, keyFunc2, keyFunc3, keyFunc4)

		req := httptest.NewRequest("GET", "/", nil)
		key := composite(req)
		assert.Equal(t, "key2:key4", key)
	})

	t.Run("returns empty for all empty keys", func(t *testing.T) {
		keyFunc := func(r *http.Request) string {
			return ""
		}

		composite := ratelimiter.Composite(keyFunc, keyFunc, keyFunc)

		req := httptest.NewRequest("GET", "/", nil)
		key := composite(req)
		assert.Equal(t, "", key)
	})

	t.Run("returns single key without modification", func(t *testing.T) {
		keyFunc := func(r *http.Request) string {
			return "single-key"
		}

		composite := ratelimiter.Composite(keyFunc)

		req := httptest.NewRequest("GET", "/", nil)
		key := composite(req)
		assert.Equal(t, "single-key", key)
	})

	t.Run("hashes long keys", func(t *testing.T) {
		longString := strings.Repeat("a", 100)
		keyFunc := func(r *http.Request) string {
			return longString
		}

		composite := ratelimiter.Composite(keyFunc)

		req := httptest.NewRequest("GET", "/", nil)
		key := composite(req)

		assert.Less(t, len(key), 65)
		assert.NotEqual(t, longString, key)

		key2 := composite(req)
		assert.Equal(t, key, key2)
	})

	t.Run("hashes combined long keys", func(t *testing.T) {
		keyFunc1 := func(r *http.Request) string {
			return strings.Repeat("x", 30)
		}
		keyFunc2 := func(r *http.Request) string {
			return strings.Repeat("y", 30)
		}
		keyFunc3 := func(r *http.Request) string {
			return strings.Repeat("z", 30)
		}

		composite := ratelimiter.Composite(keyFunc1, keyFunc2, keyFunc3)

		req := httptest.NewRequest("GET", "/", nil)
		key := composite(req)

		assert.Less(t, len(key), 65)
	})
}

func TestMiddleware_EmptyKey(t *testing.T) {
	t.Parallel()

	config := ratelimiter.Config{
		Capacity:       2,
		RefillRate:     1,
		RefillInterval: time.Second,
	}

	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	limiter, err := ratelimiter.NewTokenBucket(store, config)
	require.NoError(t, err)

	keyFunc := func(r *http.Request) string {
		return ""
	}

	middleware := ratelimiter.Middleware(limiter, keyFunc)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("rate limits with empty key", func(t *testing.T) {
		for i := range config.Capacity {
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, strconv.Itoa(config.Capacity-i-1), rec.Header().Get("X-RateLimit-Remaining"))
		}

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	})
}
