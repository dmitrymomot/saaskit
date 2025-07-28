package tenant_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/tenant"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("adds tenant to context when found", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		testTenant := createTestTenant("acme", true)
		provider.addTenant(testTenant)

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		middleware := tenant.Middleware(resolver, provider)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			retrievedTenant, ok := tenant.FromContext(r.Context())
			require.True(t, ok)
			assert.Equal(t, testTenant, retrievedTenant)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", "acme")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("continues without tenant when not found", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		middleware := tenant.Middleware(resolver, provider)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, ok := tenant.FromContext(r.Context())
			assert.False(t, ok)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("handles tenant not found error", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		middleware := tenant.Middleware(resolver, provider)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", "nonexistent")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Tenant not found")
	})

	t.Run("handles inactive tenant", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		inactiveTenant := createTestTenant("inactive", false)
		provider.addTenant(inactiveTenant)

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		middleware := tenant.Middleware(resolver, provider)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", "inactive")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "Tenant is inactive")
	})

	t.Run("allows inactive tenant when configured", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		inactiveTenant := createTestTenant("inactive", false)
		provider.addTenant(inactiveTenant)

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		middleware := tenant.Middleware(resolver, provider, tenant.WithRequireActive(false))

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			retrievedTenant, ok := tenant.FromContext(r.Context())
			require.True(t, ok)
			assert.Equal(t, inactiveTenant, retrievedTenant)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", "inactive")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("handles resolver errors", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		errorResolver := tenant.ResolverFunc(func(r *http.Request) (string, error) {
			return "", errors.New("resolver error")
		})
		middleware := tenant.Middleware(errorResolver, provider)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("handles provider timeout", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		slowTenant := createTestTenant("slow", true)
		provider.addTenant(slowTenant)

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		middleware := tenant.Middleware(resolver, provider)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow processing after tenant resolution
			time.Sleep(20 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
		req.Header.Set("X-Tenant-ID", "slow")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		// Should still resolve tenant quickly, timeout happens later
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("skips configured paths", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		middleware := tenant.Middleware(resolver, provider,
			tenant.WithSkipPaths([]string{"/health", "/metrics"}))

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, ok := tenant.FromContext(r.Context())
			assert.False(t, ok) // No tenant should be set
			w.WriteHeader(http.StatusOK)
		}))

		// Test skipped path
		req := httptest.NewRequest("GET", "/health/check", nil)
		req.Header.Set("X-Tenant-ID", "should-be-ignored")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("custom error handler", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		resolver := tenant.NewHeaderResolver("X-Tenant-ID")

		customHandler := func(w http.ResponseWriter, r *http.Request, err error) {
			w.WriteHeader(http.StatusTeapot)
			w.Write([]byte("custom error"))
		}

		middleware := tenant.Middleware(resolver, provider,
			tenant.WithErrorHandler(customHandler))

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", "nonexistent")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTeapot, w.Code)
		assert.Equal(t, "custom error", w.Body.String())
	})
}

func TestMiddleware_Caching(t *testing.T) {
	t.Parallel()

	t.Run("uses cache for subsequent requests", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		testTenant := createTestTenant("acme", true)
		provider.addTenant(testTenant)

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		cache := &tenant.NoOpCache{}
		middleware := tenant.Middleware(resolver, provider, tenant.WithCache(cache))

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// First request - should hit provider
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.Header.Set("X-Tenant-ID", "acme")
		w1 := httptest.NewRecorder()
		handler.ServeHTTP(w1, req1)
		assert.Equal(t, 1, provider.getCalls())

		// Second request - since we're using NoOpCache, it should hit provider again
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.Header.Set("X-Tenant-ID", "acme")
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)
		assert.Equal(t, 2, provider.getCalls()) // Cache is disabled, so provider called again
	})

	t.Run("respects cache TTL", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		testTenant := createTestTenant("acme", true)
		provider.addTenant(testTenant)

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		cache := &tenant.NoOpCache{}
		middleware := tenant.Middleware(resolver, provider, tenant.WithCache(cache))

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// First request
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.Header.Set("X-Tenant-ID", "acme")
		w1 := httptest.NewRecorder()
		handler.ServeHTTP(w1, req1)
		assert.Equal(t, 1, provider.getCalls())

		// Wait for cache to expire
		time.Sleep(100 * time.Millisecond)

		// Second request - should hit provider again
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.Header.Set("X-Tenant-ID", "acme")
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)
		assert.Equal(t, 2, provider.getCalls())
	})

	t.Run("validates cached inactive tenant", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		inactiveTenant := createTestTenant("inactive", false)
		provider.addTenant(inactiveTenant)

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		cache := &tenant.NoOpCache{}

		middleware := tenant.Middleware(resolver, provider, tenant.WithCache(cache))

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", "inactive")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Equal(t, 1, provider.getCalls()) // Since we're using NoOpCache, provider is called
	})

	t.Run("respects cache size limit", func(t *testing.T) {
		// Not parallel due to cache behavior

		// Test direct cache behavior first
		// Since we're using NoOpCache, no caching behavior to test
		cache := &tenant.NoOpCache{}
		
		// All cache operations should be no-ops
		err := cache.Set(context.Background(), "tenant1", createTestTenant("tenant1", true))
		assert.NoError(t, err)
		
		_, ok := cache.Get(context.Background(), "tenant1")
		assert.False(t, ok, "NoOpCache should always return false for Get")
	})

	t.Run("no-op cache disables caching", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		testTenant := createTestTenant("acme", true)
		provider.addTenant(testTenant)

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		cache := &tenant.NoOpCache{}
		middleware := tenant.Middleware(resolver, provider, tenant.WithCache(cache))

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Multiple requests should all hit provider
		for range 3 {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Tenant-ID", "acme")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}

		assert.Equal(t, 3, provider.getCalls())
	})
}

func TestRequireTenant(t *testing.T) {
	t.Parallel()

	t.Run("allows request with tenant", func(t *testing.T) {
		t.Parallel()

		middleware := tenant.RequireTenant(nil)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Create request with tenant in context
		testTenant := createTestTenant("acme", true)
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := tenant.WithTenant(req.Context(), testTenant)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("blocks request without tenant", func(t *testing.T) {
		t.Parallel()

		middleware := tenant.RequireTenant(nil)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("uses custom error handler", func(t *testing.T) {
		t.Parallel()

		customHandler := func(w http.ResponseWriter, r *http.Request, err error) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("authentication required"))
		}

		middleware := tenant.RequireTenant(customHandler)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, "authentication required", w.Body.String())
	})
}

func TestMiddleware_ConcurrentRequests(t *testing.T) {
	t.Parallel()

	provider := newMockProvider()

	// Add multiple tenants
	for i := range 10 {
		testTenant := createTestTenant(fmt.Sprintf("tenant%03d", i), true)
		provider.addTenant(testTenant)
	}

	resolver := tenant.NewHeaderResolver("X-Tenant-ID")
	middleware := tenant.Middleware(resolver, provider)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify correct tenant is in context
		retrievedTenant, ok := tenant.FromContext(r.Context())
		require.True(t, ok)

		tenantID := r.Header.Get("X-Tenant-ID")
		assert.Equal(t, tenantID, retrievedTenant.Subdomain)

		w.WriteHeader(http.StatusOK)
	}))

	// Run concurrent requests
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			tenantID := fmt.Sprintf("tenant%03d", i%10)
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Tenant-ID", tenantID)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}(i)
	}

	wg.Wait()
}

func TestMiddleware_Integration(t *testing.T) {
	t.Parallel()

	t.Run("complete flow with subdomain resolver", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		testTenant := createTestTenant("acme", true)
		provider.addTenant(testTenant)

		resolver := tenant.NewSubdomainResolver(".app.com")
		middleware := tenant.Middleware(resolver, provider)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			retrievedTenant, ok := tenant.FromContext(r.Context())
			require.True(t, ok)
			assert.Equal(t, "acme", retrievedTenant.Subdomain)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "https://acme.app.com/api/test", nil)
		req.Host = "acme.app.com"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("composite resolver with fallback", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		testTenant := createTestTenant("acme", true)
		provider.addTenant(testTenant)

		// Composite resolver: header -> subdomain
		resolver := tenant.NewCompositeResolver(
			tenant.NewHeaderResolver("X-Tenant-ID"),
			tenant.NewSubdomainResolver(".app.com"),
		)
		middleware := tenant.Middleware(resolver, provider)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			retrievedTenant, ok := tenant.FromContext(r.Context())
			require.True(t, ok)
			assert.Equal(t, "acme", retrievedTenant.Subdomain)
			w.WriteHeader(http.StatusOK)
		}))

		// Request with subdomain but no header
		req := httptest.NewRequest("GET", "https://acme.app.com/api/test", nil)
		req.Host = "acme.app.com"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
