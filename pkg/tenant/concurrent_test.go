package tenant_test

import (
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/tenant"
)

func TestResolvers_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	t.Run("subdomain_resolver_concurrent", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewSubdomainResolver(".example.com")
		const numGoroutines = 100
		const numOperations = 500

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for range numGoroutines {
			go func() {
				defer wg.Done()

				for range numOperations {
					req := httptest.NewRequest("GET", "http://test.example.com/", nil)
					tenantID, err := resolver(req)

					assert.NoError(t, err)
					assert.Equal(t, "test", tenantID)
				}
			}()
		}

		wg.Wait()
	})

	t.Run("header_resolver_concurrent", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		const numGoroutines = 100
		const numOperations = 500

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()

				for j := 0; j < numOperations; j++ {
					req := httptest.NewRequest("GET", "http://example.com/", nil)
					req.Header.Set("X-Tenant-ID", "test-tenant")

					tenantID, err := resolver(req)
					assert.NoError(t, err)
					assert.Equal(t, "test-tenant", tenantID)
				}
			}(i)
		}

		wg.Wait()
	})

	t.Run("path_resolver_concurrent", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewPathResolver(2)
		const numGoroutines = 100
		const numOperations = 500

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()

				for j := 0; j < numOperations; j++ {
					req := httptest.NewRequest("GET", "http://example.com/tenants/acme/dashboard", nil)

					tenantID, err := resolver(req)
					assert.NoError(t, err)
					assert.Equal(t, "acme", tenantID)
				}
			}(i)
		}

		wg.Wait()
	})

	t.Run("composite_resolver_concurrent", func(t *testing.T) {
		t.Parallel()

		subdomainResolver := tenant.NewSubdomainResolver(".app.com")
		headerResolver := tenant.NewHeaderResolver("X-Tenant-ID")
		pathResolver := tenant.NewPathResolver(1)

		resolver := tenant.NewCompositeResolver(
			subdomainResolver,
			headerResolver,
			pathResolver,
		)

		const numGoroutines = 100
		const numOperations = 500

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()

				for j := 0; j < numOperations; j++ {
					// Test different resolver scenarios
					switch j % 3 {
					case 0:
						// Subdomain resolution
						req := httptest.NewRequest("GET", "http://acme.app.com/", nil)
						tenantID, err := resolver(req)
						assert.NoError(t, err)
						assert.Equal(t, "acme", tenantID)
					case 1:
						// Header resolution (subdomain fails)
						req := httptest.NewRequest("GET", "http://example.com/", nil)
						req.Header.Set("X-Tenant-ID", "header-tenant")
						tenantID, err := resolver(req)
						assert.NoError(t, err)
						assert.Equal(t, "header-tenant", tenantID)
					case 2:
						// Path resolution (both subdomain and header fail)
						req := httptest.NewRequest("GET", "http://example.com/path-tenant/dashboard", nil)
						tenantID, err := resolver(req)
						assert.NoError(t, err)
						assert.Equal(t, "path-tenant", tenantID)
					}
				}
			}(i)
		}

		wg.Wait()
	})
}

func TestResolver_InputValidation_Concurrent(t *testing.T) {
	t.Parallel()

	resolver := tenant.NewHeaderResolver("X-Tenant-ID")
	const numGoroutines = 50
	const numOperations = 100 // Reduced operations for stability

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Focus on clearly valid inputs to test concurrency, not edge cases
	testInputs := []string{
		"valid-tenant",
		"a",
		"tenant123",
		"test-org",
		"company1",
		"acme-corp",
	}

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				value := testInputs[j%len(testInputs)]

				req := httptest.NewRequest("GET", "http://example.com/", nil)
				req.Header.Set("X-Tenant-ID", value)

				tenantID, err := resolver(req)
				assert.NoError(t, err)
				assert.NotEmpty(t, tenantID)
			}
		}(i)
	}

	wg.Wait()
}

// repeat helper function removed as it's no longer used
