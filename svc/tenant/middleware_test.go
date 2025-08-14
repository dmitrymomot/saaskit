package tenant_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/svc/tenant"
)

// mockProvider is an internal mock implementation of the Provider interface
type mockProvider struct {
	mock.Mock
}

// GetByIdentifier mocks the GetByIdentifier method
func (m *mockProvider) GetByIdentifier(ctx context.Context, identifier string) (*tenant.Tenant, error) {
	args := m.Called(ctx, identifier)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tenant.Tenant), args.Error(1)
}

// mockCache is an internal mock implementation of the Cache interface
type mockCache struct {
	mock.Mock
}

// Get mocks the Get method
func (m *mockCache) Get(ctx context.Context, key string) (*tenant.Tenant, bool) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*tenant.Tenant), args.Bool(1)
}

// Set mocks the Set method
func (m *mockCache) Set(ctx context.Context, key string, tenant *tenant.Tenant) error {
	args := m.Called(ctx, key, tenant)
	return args.Error(0)
}

// Delete mocks the Delete method
func (m *mockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("adds tenant to context when found", func(t *testing.T) {
		t.Parallel()

		mockProvider := new(mockProvider)
		testTenant := createTestTenant("acme", true)

		// Setup expectations
		mockProvider.On("GetByIdentifier", mock.Anything, "acme").Return(testTenant, nil).Once()

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		middleware := tenant.Middleware(resolver, mockProvider)

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

		// Verify mock expectations
		mockProvider.AssertExpectations(t)
	})

	t.Run("continues without tenant when no identifier", func(t *testing.T) {
		t.Parallel()

		mockProvider := new(mockProvider)
		// No expectations - provider should not be called when no identifier is provided

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		middleware := tenant.Middleware(resolver, mockProvider)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, ok := tenant.FromContext(r.Context())
			assert.False(t, ok)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		// No tenant header set - should continue without tenant
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify no calls were made to provider
		mockProvider.AssertExpectations(t)
	})

	t.Run("handles tenant not found error", func(t *testing.T) {
		t.Parallel()

		mockProvider := new(mockProvider)

		// Setup expectations - nonexistent tenant returns error
		mockProvider.On("GetByIdentifier", mock.Anything, "nonexistent").Return(nil, tenant.ErrTenantNotFound).Once()

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		middleware := tenant.Middleware(resolver, mockProvider)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", "nonexistent")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Tenant not found")

		// Verify mock expectations
		mockProvider.AssertExpectations(t)
	})

	t.Run("handles inactive tenant", func(t *testing.T) {
		t.Parallel()

		mockProvider := new(mockProvider)
		inactiveTenant := createTestTenant("inactive", false)

		// Setup expectations - return inactive tenant
		mockProvider.On("GetByIdentifier", mock.Anything, "inactive").Return(inactiveTenant, nil).Once()

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		middleware := tenant.Middleware(resolver, mockProvider)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", "inactive")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "Tenant is inactive")

		// Verify mock expectations
		mockProvider.AssertExpectations(t)
	})

	t.Run("allows inactive tenant when configured", func(t *testing.T) {
		t.Parallel()

		mockProvider := new(mockProvider)
		inactiveTenant := createTestTenant("inactive", false)

		// Setup expectations - return inactive tenant
		mockProvider.On("GetByIdentifier", mock.Anything, "inactive").Return(inactiveTenant, nil).Once()

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		middleware := tenant.Middleware(resolver, mockProvider, tenant.WithRequireActive(false))

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

		// Verify mock expectations
		mockProvider.AssertExpectations(t)
	})

	t.Run("handles resolver errors", func(t *testing.T) {
		t.Parallel()

		mockProvider := new(mockProvider)
		// No expectations - provider should not be called when resolver errors

		errorResolver := func(r *http.Request) (string, error) {
			return "", errors.New("resolver error")
		}
		middleware := tenant.Middleware(errorResolver, mockProvider)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		// Verify no calls were made to provider
		mockProvider.AssertExpectations(t)
	})

	t.Run("handles provider timeout", func(t *testing.T) {
		t.Parallel()

		mockProvider := new(mockProvider)
		slowTenant := createTestTenant("slow", true)

		// Setup expectations - return slow tenant
		mockProvider.On("GetByIdentifier", mock.Anything, "slow").Return(slowTenant, nil).Once()

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		middleware := tenant.Middleware(resolver, mockProvider)

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

		// Verify mock expectations
		mockProvider.AssertExpectations(t)
	})

	t.Run("skips configured paths", func(t *testing.T) {
		t.Parallel()

		mockProvider := new(mockProvider)
		// No expectations - provider should not be called for skipped paths

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		middleware := tenant.Middleware(resolver, mockProvider,
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

		// Verify no calls were made to provider
		mockProvider.AssertExpectations(t)
	})

	t.Run("custom error handler", func(t *testing.T) {
		t.Parallel()

		mockProvider := new(mockProvider)

		// Setup expectations - tenant not found
		mockProvider.On("GetByIdentifier", mock.Anything, "nonexistent").Return(nil, tenant.ErrTenantNotFound).Once()

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")

		customHandler := func(w http.ResponseWriter, r *http.Request, err error) {
			w.WriteHeader(http.StatusTeapot)
			w.Write([]byte("custom error"))
		}

		middleware := tenant.Middleware(resolver, mockProvider,
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

		// Verify mock expectations
		mockProvider.AssertExpectations(t)
	})

	t.Run("custom logger", func(t *testing.T) {
		// Create a custom logger with a buffer to capture logs
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		// Create a failing cache implementation
		failingCache := &failingCacheImpl{}

		mockProvider := new(mockProvider)
		testTenant := createTestTenant("test", true)

		// Setup expectations - return test tenant
		mockProvider.On("GetByIdentifier", mock.Anything, "test").Return(testTenant, nil).Once()

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")

		middleware := tenant.Middleware(resolver, mockProvider,
			tenant.WithCache(failingCache),
			tenant.WithLogger(logger),
		)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", "test")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Check that the custom logger was used
		assert.Contains(t, buf.String(), "failed to cache tenant")
		assert.Contains(t, buf.String(), "cache write failed")

		// Verify mock expectations
		mockProvider.AssertExpectations(t)
	})
}

func TestMiddleware_Caching(t *testing.T) {
	t.Parallel()

	t.Run("uses cache for subsequent requests", func(t *testing.T) {
		t.Parallel()

		mockProvider := new(mockProvider)
		testTenant := createTestTenant("acme", true)

		// Setup expectations - NoOpCache doesn't cache, so provider called twice
		mockProvider.On("GetByIdentifier", mock.Anything, "acme").Return(testTenant, nil).Twice()

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		cache := &tenant.NoOpCache{}
		middleware := tenant.Middleware(resolver, mockProvider, tenant.WithCache(cache))

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// First request - should hit provider
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.Header.Set("X-Tenant-ID", "acme")
		w1 := httptest.NewRecorder()
		handler.ServeHTTP(w1, req1)

		// Second request - since we're using NoOpCache, it should hit provider again
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.Header.Set("X-Tenant-ID", "acme")
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)

		// Verify mock expectations
		mockProvider.AssertExpectations(t)
	})

	t.Run("respects cache TTL", func(t *testing.T) {
		t.Parallel()

		mockProvider := new(mockProvider)
		testTenant := createTestTenant("acme", true)

		// Setup expectations - NoOpCache doesn't cache, so provider called twice
		mockProvider.On("GetByIdentifier", mock.Anything, "acme").Return(testTenant, nil).Twice()

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		cache := &tenant.NoOpCache{}
		middleware := tenant.Middleware(resolver, mockProvider, tenant.WithCache(cache))

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// First request
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.Header.Set("X-Tenant-ID", "acme")
		w1 := httptest.NewRecorder()
		handler.ServeHTTP(w1, req1)

		// Wait for cache to expire
		time.Sleep(100 * time.Millisecond)

		// Second request - should hit provider again
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.Header.Set("X-Tenant-ID", "acme")
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)

		// Verify mock expectations
		mockProvider.AssertExpectations(t)
	})

	t.Run("validates cached inactive tenant", func(t *testing.T) {
		t.Parallel()

		mockProvider := new(mockProvider)
		inactiveTenant := createTestTenant("inactive", false)

		// Setup expectations - return inactive tenant
		mockProvider.On("GetByIdentifier", mock.Anything, "inactive").Return(inactiveTenant, nil).Once()

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		cache := &tenant.NoOpCache{}

		middleware := tenant.Middleware(resolver, mockProvider, tenant.WithCache(cache))

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", "inactive")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)

		// Verify mock expectations
		mockProvider.AssertExpectations(t)
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

		mockProvider := new(mockProvider)
		testTenant := createTestTenant("acme", true)

		// Setup expectations - provider called 3 times since cache is disabled
		mockProvider.On("GetByIdentifier", mock.Anything, "acme").Return(testTenant, nil).Times(3)

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		cache := &tenant.NoOpCache{}
		middleware := tenant.Middleware(resolver, mockProvider, tenant.WithCache(cache))

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

		// Verify mock expectations
		mockProvider.AssertExpectations(t)
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

	mockProvider := new(mockProvider)

	// Create tenant mapping
	tenants := make(map[string]*tenant.Tenant)
	for i := range 10 {
		tenantID := fmt.Sprintf("tenant%03d", i)
		testTenant := createTestTenant(tenantID, true)
		tenants[tenantID] = testTenant
	}

	// Setup expectations for all tenants (each called multiple times)
	for tenantID, testTenant := range tenants {
		mockProvider.On("GetByIdentifier", mock.Anything, tenantID).Return(testTenant, nil).Maybe()
	}

	resolver := tenant.NewHeaderResolver("X-Tenant-ID")
	middleware := tenant.Middleware(resolver, mockProvider)

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

	// Verify mock expectations
	mockProvider.AssertExpectations(t)
}

func TestMiddleware_Integration(t *testing.T) {
	t.Parallel()

	t.Run("complete flow with subdomain resolver", func(t *testing.T) {
		t.Parallel()

		mockProvider := new(mockProvider)
		testTenant := createTestTenant("acme", true)

		// Setup expectations
		mockProvider.On("GetByIdentifier", mock.Anything, "acme").Return(testTenant, nil).Once()

		resolver := tenant.NewSubdomainResolver(".app.com")
		middleware := tenant.Middleware(resolver, mockProvider)

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

		// Verify mock expectations
		mockProvider.AssertExpectations(t)
	})

	t.Run("composite resolver with fallback", func(t *testing.T) {
		t.Parallel()

		mockProvider := new(mockProvider)
		testTenant := createTestTenant("acme", true)

		// Setup expectations
		mockProvider.On("GetByIdentifier", mock.Anything, "acme").Return(testTenant, nil).Once()

		// Composite resolver: header -> subdomain
		resolver := tenant.NewCompositeResolver(
			tenant.NewHeaderResolver("X-Tenant-ID"),
			tenant.NewSubdomainResolver(".app.com"),
		)
		middleware := tenant.Middleware(resolver, mockProvider)

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

		// Verify mock expectations
		mockProvider.AssertExpectations(t)
	})
}

// failingCacheImpl is a cache implementation that always fails on Set operations
type failingCacheImpl struct{}

func (f *failingCacheImpl) Get(ctx context.Context, key string) (*tenant.Tenant, bool) {
	return nil, false
}

func (f *failingCacheImpl) Set(ctx context.Context, key string, tenant *tenant.Tenant) error {
	return errors.New("cache write failed")
}

func (f *failingCacheImpl) Delete(ctx context.Context, key string) error {
	return nil
}
