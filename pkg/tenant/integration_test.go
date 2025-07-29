package tenant_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/tenant"
)

// Integration tests that test multiple components working together

func TestIntegration_CompleteFlow(t *testing.T) {
	t.Parallel()

	t.Run("multi-tenant API with different resolvers", func(t *testing.T) {
		t.Parallel()

		// Setup provider with multiple tenants
		mockProvider := new(mockProvider)
		acmeTenant := createTestTenant("acme", true)
		globexTenant := createTestTenant("globex", true)

		// Setup expectations
		mockProvider.On("GetByIdentifier", mock.Anything, "acme").Return(acmeTenant, nil).Maybe()
		mockProvider.On("GetByIdentifier", mock.Anything, "globex").Return(globexTenant, nil).Maybe()

		// Create composite resolver: header -> subdomain -> path
		resolver := tenant.NewCompositeResolver(
			tenant.NewHeaderResolver("X-Tenant-ID"),
			tenant.NewSubdomainResolver(".app.com"),
			tenant.NewPathResolver(2), // /api/{tenant}/...
		)

		// Setup middleware
		cache := &tenant.NoOpCache{}
		middleware := tenant.Middleware(resolver, mockProvider,
			tenant.WithCache(cache),
			tenant.WithSkipPaths([]string{"/health", "/metrics"}),
		)

		// Create test handler that requires tenant
		apiHandler := tenant.RequireTenant(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ten, ok := tenant.FromContext(r.Context())
			require.True(t, ok)
			w.Header().Set("X-Tenant-Name", ten.Name)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, "Hello %s", ten.Name)
		}))

		// Apply middleware
		handler := middleware(apiHandler)

		// Test 1: Header resolver
		t.Run("resolves via header", func(t *testing.T) {
			req := httptest.NewRequest("GET", "https://api.app.com/users", nil)
			req.Header.Set("X-Tenant-ID", "acme")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "acme Corp", w.Header().Get("X-Tenant-Name"))
			assert.Equal(t, "Hello acme Corp", w.Body.String())
		})

		// Test 2: Subdomain resolver
		t.Run("resolves via subdomain", func(t *testing.T) {
			req := httptest.NewRequest("GET", "https://globex.app.com/users", nil)
			req.Host = "globex.app.com"
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "globex Corp", w.Header().Get("X-Tenant-Name"))
		})

		// Test 3: Path resolver
		t.Run("resolves via path", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/acme/users", nil)
			req.Host = "localhost"
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "acme Corp", w.Header().Get("X-Tenant-Name"))
		})

		// Test 4: Skipped paths
		t.Run("skips health check", func(t *testing.T) {
			req := httptest.NewRequest("GET", "https://api.app.com/health", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			assert.Equal(t, http.StatusInternalServerError, w.Code) // RequireTenant will fail
		})

		// Verify mock expectations
		mockProvider.AssertExpectations(t)
	})
}

func TestIntegration_LoadTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	// Setup
	mockProvider := new(mockProvider)
	tenants := make(map[string]*tenant.Tenant)
	for i := range 100 {
		tenantID := fmt.Sprintf("tenant%d", i)
		testTenant := createTestTenant(tenantID, true)
		tenants[tenantID] = testTenant
	}

	// Setup expectations for all tenants
	for tenantID, testTenant := range tenants {
		mockProvider.On("GetByIdentifier", mock.Anything, tenantID).Return(testTenant, nil).Maybe()
	}

	resolver := tenant.NewHeaderResolver("X-Tenant-ID")
	cache := &tenant.NoOpCache{}
	middleware := tenant.Middleware(resolver, mockProvider,
		tenant.WithCache(cache),
	)

	var requestCount atomic.Int64
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)

		ten, ok := tenant.FromContext(r.Context())
		require.True(t, ok)

		// Simulate some work
		time.Sleep(1 * time.Millisecond)

		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "Processed for %s", ten.Name)
	}))

	// Run load test
	start := time.Now()
	var wg sync.WaitGroup
	concurrency := 100
	requestsPerWorker := 100

	for i := range concurrency {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := range requestsPerWorker {
				tenantID := fmt.Sprintf("tenant%d", (workerID+j)%100)
				req := httptest.NewRequest("GET", "/api/test", nil)
				req.Header.Set("X-Tenant-ID", tenantID)
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)
				assert.Equal(t, http.StatusOK, w.Code)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalRequests := int64(concurrency * requestsPerWorker)
	assert.Equal(t, totalRequests, requestCount.Load())

	// Should handle at least 1000 req/s
	reqPerSec := float64(totalRequests) / duration.Seconds()
	t.Logf("Processed %d requests in %v (%.2f req/s)", totalRequests, duration, reqPerSec)
	assert.Greater(t, reqPerSec, 1000.0)

	// Verify mock expectations
	mockProvider.AssertExpectations(t)
}

func TestIntegration_CacheInvalidation(t *testing.T) {
	t.Parallel()

	// Provider that can change tenant state
	mockProvider := new(mockProvider)

	activeTenant := createTestTenant("acme", true)
	inactiveTenant := createTestTenant("acme", false)

	resolver := tenant.NewHeaderResolver("X-Tenant-ID")
	cache := &tenant.NoOpCache{}
	middleware := tenant.Middleware(resolver, mockProvider,
		tenant.WithCache(cache),
	)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ten, ok := tenant.FromContext(r.Context())
		require.True(t, ok)

		if ten.Active {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Active"))
		} else {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("Inactive"))
		}
	}))

	t.Run("handles cache and tenant changes", func(t *testing.T) {
		// Setup expectations - NoOpCache means provider is called each time
		// First two requests return active tenant
		mockProvider.On("GetByIdentifier", mock.Anything, "acme").Return(activeTenant, nil).Twice()
		// Third request returns inactive tenant
		mockProvider.On("GetByIdentifier", mock.Anything, "acme").Return(inactiveTenant, nil).Once()

		// First request - active tenant
		req1 := httptest.NewRequest("GET", "/api", nil)
		req1.Header.Set("X-Tenant-ID", "acme")
		w1 := httptest.NewRecorder()

		handler.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)
		assert.Equal(t, "Active", w1.Body.String())

		// Second request - still active tenant (NoOpCache doesn't cache)
		req2 := httptest.NewRequest("GET", "/api", nil)
		req2.Header.Set("X-Tenant-ID", "acme")
		w2 := httptest.NewRecorder()

		handler.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)

		// Third request - now inactive tenant
		req3 := httptest.NewRequest("GET", "/api", nil)
		req3.Header.Set("X-Tenant-ID", "acme")
		w3 := httptest.NewRecorder()

		handler.ServeHTTP(w3, req3)
		assert.Equal(t, http.StatusForbidden, w3.Code)

		// Verify mock expectations
		mockProvider.AssertExpectations(t)
	})
}

// Benchmark complete middleware stack
func BenchmarkIntegration_MiddlewareStack(b *testing.B) {
	mockProvider := new(mockProvider)
	tenants := make(map[string]*tenant.Tenant)
	for i := range 100 {
		tenantID := fmt.Sprintf("tenant%d", i)
		testTenant := createTestTenant(tenantID, true)
		tenants[tenantID] = testTenant
	}

	// Setup expectations for all tenants
	for tenantID, testTenant := range tenants {
		mockProvider.On("GetByIdentifier", mock.Anything, tenantID).Return(testTenant, nil).Maybe()
	}

	resolver := tenant.NewCompositeResolver(
		tenant.NewHeaderResolver("X-Tenant-ID"),
		tenant.NewSubdomainResolver(".app.com"),
	)

	cache := &tenant.NoOpCache{}
	middleware := tenant.Middleware(resolver, mockProvider,
		tenant.WithCache(cache),
	)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ten, _ := tenant.FromContext(r.Context())
		w.Header().Set("X-Tenant", ten.Subdomain)
		w.WriteHeader(http.StatusOK)
	}))

	// Pre-warm cache
	for i := range 10 {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", fmt.Sprintf("tenant%d", i))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Tenant-ID", fmt.Sprintf("tenant%d", i%100))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			i++
		}
	})
}
