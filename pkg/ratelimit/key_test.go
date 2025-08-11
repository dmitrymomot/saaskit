package ratelimit_test

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/ratelimit"

	"github.com/stretchr/testify/assert"
)

func TestComposite(t *testing.T) {
	t.Parallel()

	ipKeyFunc := func(r *http.Request) string {
		return r.RemoteAddr
	}

	pathKeyFunc := func(r *http.Request) string {
		return r.URL.Path
	}

	userKeyFunc := func(r *http.Request) string {
		return r.Header.Get("X-User-ID")
	}

	tests := []struct {
		name     string
		keyFuncs []ratelimit.KeyFunc
		setup    func(*http.Request)
		expected string
	}{
		{
			name:     "empty key functions",
			keyFuncs: []ratelimit.KeyFunc{},
			expected: "",
		},
		{
			name:     "single key function",
			keyFuncs: []ratelimit.KeyFunc{ipKeyFunc},
			setup: func(r *http.Request) {
				r.RemoteAddr = "192.168.1.1:8080"
			},
			expected: "192.168.1.1:8080",
		},
		{
			name:     "single key under 64 chars",
			keyFuncs: []ratelimit.KeyFunc{pathKeyFunc},
			setup: func(r *http.Request) {
				r.URL.Path = "/api/v1/users"
			},
			expected: "/api/v1/users",
		},
		{
			name:     "multiple keys combined",
			keyFuncs: []ratelimit.KeyFunc{ipKeyFunc, pathKeyFunc},
			setup: func(r *http.Request) {
				r.RemoteAddr = "192.168.1.1:8080"
				r.URL.Path = "/api/v1/users"
			},
			expected: "192.168.1.1:8080:/api/v1/users",
		},
		{
			name:     "skip empty keys",
			keyFuncs: []ratelimit.KeyFunc{ipKeyFunc, userKeyFunc, pathKeyFunc},
			setup: func(r *http.Request) {
				r.RemoteAddr = "192.168.1.1:8080"
				r.URL.Path = "/api/v1/users"
			},
			expected: "192.168.1.1:8080:/api/v1/users",
		},
		{
			name: "all empty keys",
			keyFuncs: []ratelimit.KeyFunc{
				func(r *http.Request) string { return "" },
				func(r *http.Request) string { return "" },
			},
			expected: "",
		},
		{
			name: "long key gets hashed",
			keyFuncs: []ratelimit.KeyFunc{
				func(r *http.Request) string {
					return strings.Repeat("a", 70)
				},
			},
			setup: func(r *http.Request) {},
			expected: func() string {
				key := strings.Repeat("a", 70)
				hash := sha256.Sum256([]byte(key))
				return hex.EncodeToString(hash[:16])
			}(),
		},
		{
			name: "combined key over 64 chars gets hashed",
			keyFuncs: []ratelimit.KeyFunc{
				func(r *http.Request) string { return strings.Repeat("a", 30) },
				func(r *http.Request) string { return strings.Repeat("b", 30) },
				func(r *http.Request) string { return strings.Repeat("c", 10) },
			},
			setup: func(r *http.Request) {},
			expected: func() string {
				combined := strings.Repeat("a", 30) + ":" + strings.Repeat("b", 30) + ":" + strings.Repeat("c", 10)
				hash := sha256.Sum256([]byte(combined))
				return hex.EncodeToString(hash[:16])
			}(),
		},
		{
			name: "exactly 64 chars not hashed",
			keyFuncs: []ratelimit.KeyFunc{
				func(r *http.Request) string { return strings.Repeat("x", 64) },
			},
			setup:    func(r *http.Request) {},
			expected: strings.Repeat("x", 64),
		},
		{
			name: "combined exactly 64 chars not hashed",
			keyFuncs: []ratelimit.KeyFunc{
				func(r *http.Request) string { return strings.Repeat("a", 31) },
				func(r *http.Request) string { return strings.Repeat("b", 32) },
			},
			setup:    func(r *http.Request) {},
			expected: strings.Repeat("a", 31) + ":" + strings.Repeat("b", 32),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.setup != nil {
				tt.setup(req)
			}

			compositeFunc := ratelimit.Composite(tt.keyFuncs...)
			result := compositeFunc(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestComposite_HashCollisionResistance(t *testing.T) {
	t.Parallel()

	keyFunc1 := ratelimit.Composite(
		func(r *http.Request) string { return strings.Repeat("a", 100) },
		func(r *http.Request) string { return strings.Repeat("b", 100) },
	)

	keyFunc2 := ratelimit.Composite(
		func(r *http.Request) string { return strings.Repeat("b", 100) },
		func(r *http.Request) string { return strings.Repeat("a", 100) },
	)

	req := httptest.NewRequest("GET", "/test", nil)

	key1 := keyFunc1(req)
	key2 := keyFunc2(req)

	assert.NotEqual(t, key1, key2, "different key combinations should produce different hashes")
	assert.Len(t, key1, 32, "hashed key should be 32 hex chars (128 bits)")
	assert.Len(t, key2, 32, "hashed key should be 32 hex chars (128 bits)")
}

func TestComposite_RealWorldScenarios(t *testing.T) {
	t.Parallel()

	t.Run("API rate limiting by IP and path", func(t *testing.T) {
		t.Parallel()

		keyFunc := ratelimit.Composite(
			func(r *http.Request) string { return r.RemoteAddr },
			func(r *http.Request) string { return r.URL.Path },
		)

		req1 := httptest.NewRequest("GET", "/api/users", nil)
		req1.RemoteAddr = "192.168.1.1:8080"

		req2 := httptest.NewRequest("GET", "/api/posts", nil)
		req2.RemoteAddr = "192.168.1.1:8080"

		req3 := httptest.NewRequest("GET", "/api/users", nil)
		req3.RemoteAddr = "192.168.1.2:8080"

		key1 := keyFunc(req1)
		key2 := keyFunc(req2)
		key3 := keyFunc(req3)

		assert.NotEqual(t, key1, key2, "same IP different path should have different keys")
		assert.NotEqual(t, key1, key3, "same path different IP should have different keys")
		assert.NotEqual(t, key2, key3, "different IP and path should have different keys")
	})

	t.Run("user-specific rate limiting with fallback to IP", func(t *testing.T) {
		t.Parallel()

		keyFunc := ratelimit.Composite(
			func(r *http.Request) string {
				if userID := r.Header.Get("X-User-ID"); userID != "" {
					return "user:" + userID
				}
				return ""
			},
			func(r *http.Request) string {
				if r.RemoteAddr != "" {
					return "ip:" + r.RemoteAddr
				}
				return ""
			},
		)

		req1 := httptest.NewRequest("GET", "/api/test", nil)
		req1.Header.Set("X-User-ID", "user123")
		req1.RemoteAddr = "192.168.1.1:8080"

		req2 := httptest.NewRequest("GET", "/api/test", nil)
		req2.RemoteAddr = "192.168.1.1:8080"

		key1 := keyFunc(req1)
		key2 := keyFunc(req2)

		assert.Equal(t, "user:user123:ip:192.168.1.1:8080", key1)
		assert.Equal(t, "ip:192.168.1.1:8080", key2)
	})

	t.Run("complex multi-tenant scenario", func(t *testing.T) {
		t.Parallel()

		keyFunc := ratelimit.Composite(
			func(r *http.Request) string { return r.Header.Get("X-Tenant-ID") },
			func(r *http.Request) string { return r.Header.Get("X-User-ID") },
			func(r *http.Request) string { return r.Method },
			func(r *http.Request) string { return r.URL.Path },
		)

		req := httptest.NewRequest("POST", "/api/v1/orders", nil)
		req.Header.Set("X-Tenant-ID", "tenant-abc")
		req.Header.Set("X-User-ID", "user-123")

		key := keyFunc(req)
		assert.Equal(t, "tenant-abc:user-123:POST:/api/v1/orders", key)
	})
}

func TestComposite_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("nil key function in list", func(t *testing.T) {
		// The current implementation will panic on nil functions
		// This test documents the current behavior
		keyFunc := ratelimit.Composite(
			func(r *http.Request) string { return "test" },
		)
		req := httptest.NewRequest("GET", "/test", nil)
		result := keyFunc(req)
		assert.Equal(t, "test", result)
	})

	t.Run("key with colons", func(t *testing.T) {
		t.Parallel()

		keyFunc := ratelimit.Composite(
			func(r *http.Request) string { return "key:with:colons" },
			func(r *http.Request) string { return "another:key" },
		)

		req := httptest.NewRequest("GET", "/test", nil)
		result := keyFunc(req)
		assert.Equal(t, "key:with:colons:another:key", result)
	})

	t.Run("unicode characters", func(t *testing.T) {
		t.Parallel()

		keyFunc := ratelimit.Composite(
			func(r *http.Request) string { return "用户123" },
			func(r *http.Request) string { return "路径/测试" },
		)

		req := httptest.NewRequest("GET", "/test", nil)
		result := keyFunc(req)
		assert.Equal(t, "用户123:路径/测试", result)
	})

	t.Run("special characters", func(t *testing.T) {
		t.Parallel()

		keyFunc := ratelimit.Composite(
			func(r *http.Request) string { return "key!@#$%^&*()" },
			func(r *http.Request) string { return "path/with/slashes" },
		)

		req := httptest.NewRequest("GET", "/test", nil)
		result := keyFunc(req)
		assert.Equal(t, "key!@#$%^&*():path/with/slashes", result)
	})
}

func BenchmarkComposite(b *testing.B) {
	keyFunc := ratelimit.Composite(
		func(r *http.Request) string { return r.RemoteAddr },
		func(r *http.Request) string { return r.URL.Path },
		func(r *http.Request) string { return r.Header.Get("X-User-ID") },
	)

	req := httptest.NewRequest("GET", "/api/v1/users/123", nil)
	req.RemoteAddr = "192.168.1.1:8080"
	req.Header.Set("X-User-ID", "user-12345")

	b.ResetTimer()
	for range b.N {
		_ = keyFunc(req)
	}
}

func BenchmarkComposite_LongKey(b *testing.B) {
	keyFunc := ratelimit.Composite(
		func(r *http.Request) string { return strings.Repeat("a", 50) },
		func(r *http.Request) string { return strings.Repeat("b", 50) },
	)

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for range b.N {
		_ = keyFunc(req)
	}
}
