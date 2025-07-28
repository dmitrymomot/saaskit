package tenant_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/tenant"
)

// Integration tests that test multiple components working together

func TestIntegration_CompleteFlow(t *testing.T) {
	t.Parallel()

	t.Run("multi-tenant API with different resolvers", func(t *testing.T) {
		t.Parallel()

		// Setup provider with multiple tenants
		provider := newMockProvider()
		acmeTenant := createTestTenant("acme", true)
		globexTenant := createTestTenant("globex", true)
		provider.addTenant(acmeTenant)
		provider.addTenant(globexTenant)

		// Create composite resolver: header -> subdomain -> path
		resolver := tenant.NewCompositeResolver(
			tenant.NewHeaderResolver("X-Tenant-ID"),
			tenant.NewSubdomainResolver(".app.com"),
			tenant.NewPathResolver(2), // /api/{tenant}/...
		)

		// Setup middleware
		middleware := tenant.Middleware(resolver, provider,
			tenant.WithCacheTTL(5*time.Minute),
			tenant.WithSkipPaths([]string{"/health", "/metrics"}),
		)

		// Create test handler that requires tenant
		apiHandler := tenant.RequireTenant(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ten, ok := tenant.FromContext(r.Context())
			require.True(t, ok)
			w.Header().Set("X-Tenant-Name", ten.Name)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Hello %s", ten.Name)
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
	})
}

func TestIntegration_SessionBasedMultiTenancy(t *testing.T) {
	t.Parallel()

	// Simulate a session store
	type sessionStore struct {
		mu       sync.RWMutex
		sessions map[string]map[string]string
	}

	store := &sessionStore{
		sessions: make(map[string]map[string]string),
	}

	// Session middleware that sets session ID
	sessionMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionID := r.Header.Get("X-Session-ID")
			if sessionID == "" {
				sessionID = uuid.New().String()
				w.Header().Set("X-Session-ID", sessionID)
			}

			ctx := context.WithValue(r.Context(), "session-id", sessionID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	// GetSession function for resolver
	getSession := func(r *http.Request) (tenant.SessionData, error) {
		sessionID, ok := r.Context().Value("session-id").(string)
		if !ok {
			return nil, fmt.Errorf("no session ID in context")
		}

		store.mu.RLock()
		defer store.mu.RUnlock()

		data, exists := store.sessions[sessionID]
		if !exists {
			return &mockSession{data: make(map[string]string)}, nil
		}

		return &mockSession{data: data}, nil
	}

	// Setup tenant resolution
	provider := newMockProvider()
	acmeTenant := createTestTenant("acme", true)
	globexTenant := createTestTenant("globex", true)
	provider.addTenant(acmeTenant)
	provider.addTenant(globexTenant)

	resolver := tenant.NewSessionResolver(getSession)
	tenantMiddleware := tenant.Middleware(resolver, provider)

	// API handler
	handler := sessionMiddleware(tenantMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ten, ok := tenant.FromContext(r.Context())
		if ok {
			w.Header().Set("X-Current-Tenant", ten.Subdomain)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Current tenant: %s", ten.Name)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("No tenant selected"))
		}
	})))

	// Test flow
	t.Run("session-based tenant switching", func(t *testing.T) {
		sessionID := uuid.New().String()

		// Initial request - no tenant
		req1 := httptest.NewRequest("GET", "/dashboard", nil)
		req1.Header.Set("X-Session-ID", sessionID)
		w1 := httptest.NewRecorder()

		handler.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)
		assert.Equal(t, "No tenant selected", w1.Body.String())

		// Set tenant in session
		store.mu.Lock()
		store.sessions[sessionID] = map[string]string{"tenant_id": "acme"}
		store.mu.Unlock()

		// Second request - should have tenant
		req2 := httptest.NewRequest("GET", "/dashboard", nil)
		req2.Header.Set("X-Session-ID", sessionID)
		w2 := httptest.NewRecorder()

		handler.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Equal(t, "acme", w2.Header().Get("X-Current-Tenant"))
		assert.Equal(t, "Current tenant: acme Corp", w2.Body.String())

		// Switch tenant
		store.mu.Lock()
		store.sessions[sessionID]["tenant_id"] = "globex"
		store.mu.Unlock()

		// Third request - should have new tenant
		req3 := httptest.NewRequest("GET", "/dashboard", nil)
		req3.Header.Set("X-Session-ID", sessionID)
		w3 := httptest.NewRecorder()

		handler.ServeHTTP(w3, req3)
		assert.Equal(t, http.StatusOK, w3.Code)
		assert.Equal(t, "globex", w3.Header().Get("X-Current-Tenant"))
		assert.Equal(t, "Current tenant: globex Corp", w3.Body.String())
	})
}

func TestIntegration_LoadTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	// Setup
	provider := newMockProvider()
	for i := range 100 {
		testTenant := createTestTenant(fmt.Sprintf("tenant%d", i), true)
		provider.addTenant(testTenant)
	}

	resolver := tenant.NewHeaderResolver("X-Tenant-ID")
	middleware := tenant.Middleware(resolver, provider,
		tenant.WithCacheTTL(1*time.Minute),
	)

	var requestCount atomic.Int64
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)

		ten, ok := tenant.FromContext(r.Context())
		require.True(t, ok)

		// Simulate some work
		time.Sleep(1 * time.Millisecond)

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Processed for %s", ten.Name)
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
}

func TestIntegration_CacheInvalidation(t *testing.T) {
	t.Parallel()

	// Provider that can change tenant state
	provider := newMockProvider()

	activeTenant := createTestTenant("acme", true)
	provider.addTenant(activeTenant)

	resolver := tenant.NewHeaderResolver("X-Tenant-ID")
	cache := tenant.NewInMemoryCache()
	middleware := tenant.Middleware(resolver, provider,
		tenant.WithCache(cache),
		tenant.WithCacheTTL(1*time.Hour), // Long TTL
	)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ten, ok := tenant.FromContext(r.Context())
		require.True(t, ok)

		if ten.Active {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Active"))
		} else {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Inactive"))
		}
	}))

	t.Run("handles cache and tenant changes", func(t *testing.T) {
		// First request - active tenant gets cached
		req1 := httptest.NewRequest("GET", "/api", nil)
		req1.Header.Set("X-Tenant-ID", "acme")
		w1 := httptest.NewRecorder()

		handler.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)
		assert.Equal(t, "Active", w1.Body.String())

		// Second request - should use cached active tenant
		req2 := httptest.NewRequest("GET", "/api", nil)
		req2.Header.Set("X-Tenant-ID", "acme")
		w2 := httptest.NewRecorder()

		handler.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code) // Still OK because cached tenant is active

		// Deactivate tenant in provider
		provider.mu.Lock()
		provider.tenants["acme"].Active = false
		provider.mu.Unlock()

		// Manually invalidate cache to force reload
		cache.Delete(context.Background(), "acme")

		// Third request - should load from provider and see inactive tenant
		req3 := httptest.NewRequest("GET", "/api", nil)
		req3.Header.Set("X-Tenant-ID", "acme")
		w3 := httptest.NewRecorder()

		handler.ServeHTTP(w3, req3)
		assert.Equal(t, http.StatusForbidden, w3.Code)
	})
}

// Benchmark complete middleware stack
func BenchmarkIntegration_MiddlewareStack(b *testing.B) {
	provider := newMockProvider()
	for i := range 100 {
		testTenant := createTestTenant(fmt.Sprintf("tenant%d", i), true)
		provider.addTenant(testTenant)
	}

	resolver := tenant.NewCompositeResolver(
		tenant.NewHeaderResolver("X-Tenant-ID"),
		tenant.NewSubdomainResolver(".app.com"),
	)

	middleware := tenant.Middleware(resolver, provider,
		tenant.WithCacheTTL(5*time.Minute),
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
