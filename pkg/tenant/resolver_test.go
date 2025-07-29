package tenant_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/tenant"
)

func TestSubdomainResolver(t *testing.T) {
	t.Parallel()

	t.Run("extracts subdomain from host", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewSubdomainResolver("")
		req := httptest.NewRequest("GET", "https://acme.app.com/test", nil)

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Equal(t, "acme", id)
	})

	t.Run("extracts subdomain with custom suffix", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewSubdomainResolver(".saas.com")
		req := httptest.NewRequest("GET", "https://acme.saas.com/test", nil)
		req.Host = "acme.saas.com"

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Equal(t, "acme", id)
	})

	t.Run("handles host with port", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewSubdomainResolver("")
		req := httptest.NewRequest("GET", "http://acme.app.localhost:8080/test", nil)
		req.Host = "acme.app.localhost:8080"

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Equal(t, "acme", id)
	})

	t.Run("skips www prefix", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewSubdomainResolver("")
		req := httptest.NewRequest("GET", "https://www.acme.app.com/test", nil)

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Equal(t, "acme", id)
	})

	t.Run("returns empty for base domain", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewSubdomainResolver("")
		req := httptest.NewRequest("GET", "https://app.com/test", nil)
		req.Host = "app.com"

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Empty(t, id)
	})

	t.Run("returns empty for single part domain", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewSubdomainResolver("")
		req := httptest.NewRequest("GET", "https://localhost/test", nil)

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Empty(t, id)
	})

	t.Run("returns empty for www only", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewSubdomainResolver("")
		req := httptest.NewRequest("GET", "https://www/test", nil)

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Empty(t, id)
	})

	t.Run("validates subdomain format", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewSubdomainResolver("")

		invalidSubdomains := []string{
			"invalid!@#",     // special characters
			"tenant_123",     // underscore not allowed
			"tenant@123",     // @ not allowed
			"tenant%20space", // space (encoded) not allowed
		}

		for _, subdomain := range invalidSubdomains {
			req := httptest.NewRequest("GET", "https://app.com/test", nil)
			req.Host = subdomain + ".app.com"

			id, err := resolver(req)
			assert.Error(t, err, "subdomain %s should be invalid", subdomain)
			assert.Empty(t, id)
			assert.ErrorIs(t, err, tenant.ErrInvalidIdentifier)
		}
	})

	t.Run("accepts valid subdomain formats", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewSubdomainResolver("")
		validSubdomains := []string{
			"tenant123",
			"tenant-123",
			"TENANT123",
			"a1b2c3d4-e5f6-7890-1234-567890abcdef", // UUID format with dashes only
		}

		for _, subdomain := range validSubdomains {
			req := httptest.NewRequest("GET", "https://"+subdomain+".app.com/test", nil)
			req.Host = subdomain + ".app.com"

			id, err := resolver(req)
			require.NoError(t, err, "subdomain %s should be valid", subdomain)
			assert.Equal(t, subdomain, id)
		}
	})

	t.Run("handles empty host", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewSubdomainResolver("")
		req := httptest.NewRequest("GET", "/test", nil)
		req.Host = ""

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Empty(t, id)
	})
}

func TestHeaderResolver(t *testing.T) {
	t.Parallel()

	t.Run("extracts tenant from custom header", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", "tenant123")

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Equal(t, "tenant123", id)
	})

	t.Run("uses default header when empty", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewHeaderResolver("")
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", "tenant123")

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Equal(t, "tenant123", id)
	})

	t.Run("returns empty for missing header", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		req := httptest.NewRequest("GET", "/test", nil)

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Empty(t, id)
	})

	t.Run("handles different header names", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewHeaderResolver("X-Company-ID")
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Company-ID", "company456")

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Equal(t, "company456", id)
	})

	t.Run("validates header value format", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")

		invalidIDs := []string{
			"invalid!@#$%", // special characters
			"tenant_123",   // underscore not allowed
			"tenant.com",   // dot not allowed
			"tenant@corp",  // @ not allowed
			"tenant space", // space not allowed
		}

		for _, invalidID := range invalidIDs {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Tenant-ID", invalidID)

			id, err := resolver(req)
			assert.Error(t, err, "tenant ID %s should be invalid", invalidID)
			assert.Empty(t, id)
			assert.ErrorIs(t, err, tenant.ErrInvalidIdentifier)
		}
	})

	t.Run("accepts valid header values", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewHeaderResolver("X-Tenant-ID")
		validIDs := []string{
			"tenant123",
			"tenant-123",
			"a1b2c3d4-e5f6-7890-1234-567890abcdef", // UUID format also works
		}

		for _, tenantID := range validIDs {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Tenant-ID", tenantID)

			id, err := resolver(req)
			require.NoError(t, err, "tenant ID %s should be valid", tenantID)
			assert.Equal(t, tenantID, id)
		}
	})
}

func TestPathResolver(t *testing.T) {
	t.Parallel()

	t.Run("extracts tenant from path position", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewPathResolver(2)
		req := httptest.NewRequest("GET", "/api/tenant123/users", nil)

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Equal(t, "tenant123", id)
	})

	t.Run("handles different positions", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewPathResolver(1)
		req := httptest.NewRequest("GET", "/tenant123/api/users", nil)

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Equal(t, "tenant123", id)
	})

	t.Run("returns error for invalid position", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewPathResolver(0)
		req := httptest.NewRequest("GET", "/api/tenant123/users", nil)

		_, err := resolver(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid path position")
	})

	t.Run("returns empty for position beyond path length", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewPathResolver(5)
		req := httptest.NewRequest("GET", "/api/v1", nil)

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Empty(t, id)
	})

	t.Run("handles trailing slashes", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewPathResolver(2)
		req := httptest.NewRequest("GET", "/api/tenant123/", nil)

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Equal(t, "tenant123", id)
	})

	t.Run("handles leading slashes", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewPathResolver(1)
		req := httptest.NewRequest("GET", "/tenant123/api", nil)

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Equal(t, "tenant123", id)
	})

	t.Run("returns empty for root path", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewPathResolver(1)
		req := httptest.NewRequest("GET", "/", nil)

		id, err := resolver(req)
		require.NoError(t, err)
		assert.Empty(t, id)
	})

	t.Run("validates path segment format", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewPathResolver(2)

		invalidPaths := []string{
			"/api/invalid!@#/users",
			"/api/tenant_123/users",     // underscore not allowed
			"/api/tenant.com/users",     // dot not allowed
			"/api/tenant@corp/users",    // @ not allowed
			"/api/tenant%20space/users", // space (encoded) not allowed
		}

		for _, path := range invalidPaths {
			req := httptest.NewRequest("GET", path, nil)

			id, err := resolver(req)
			assert.Error(t, err, "path %s should be invalid", path)
			assert.Empty(t, id)
			assert.ErrorIs(t, err, tenant.ErrInvalidIdentifier)
		}
	})

	t.Run("accepts valid path segments", func(t *testing.T) {
		t.Parallel()

		resolver := tenant.NewPathResolver(1)
		validIDs := []string{
			"tenant123",
			"tenant-123",
			"a1b2c3d4-e5f6-7890-1234-567890abcdef", // UUID format
		}

		for _, tenantID := range validIDs {
			req := httptest.NewRequest("GET", "/"+tenantID+"/api/users", nil)

			id, err := resolver(req)
			require.NoError(t, err, "tenant ID %s should be valid", tenantID)
			assert.Equal(t, tenantID, id)
		}
	})
}

func TestCompositeResolver(t *testing.T) {
	t.Parallel()

	t.Run("tries resolvers in order", func(t *testing.T) {
		t.Parallel()

		headerResolver := tenant.NewHeaderResolver("X-Tenant-ID")
		pathResolver := tenant.NewPathResolver(2)
		composite := tenant.NewCompositeResolver(headerResolver, pathResolver)

		// Request with header
		req := httptest.NewRequest("GET", "/api/users", nil)
		req.Header.Set("X-Tenant-ID", "header-tenant")

		id, err := composite(req)
		require.NoError(t, err)
		assert.Equal(t, "header-tenant", id)
	})

	t.Run("falls back to next resolver", func(t *testing.T) {
		t.Parallel()

		headerResolver := tenant.NewHeaderResolver("X-Tenant-ID")
		pathResolver := tenant.NewPathResolver(2)
		composite := tenant.NewCompositeResolver(headerResolver, pathResolver)

		// Request without header but with path
		req := httptest.NewRequest("GET", "/api/path-tenant/users", nil)

		id, err := composite(req)
		require.NoError(t, err)
		assert.Equal(t, "path-tenant", id)
	})

	t.Run("returns empty when all resolvers return empty", func(t *testing.T) {
		t.Parallel()

		headerResolver := tenant.NewHeaderResolver("X-Tenant-ID")
		pathResolver := tenant.NewPathResolver(5) // Beyond path length
		composite := tenant.NewCompositeResolver(headerResolver, pathResolver)

		req := httptest.NewRequest("GET", "/api", nil)

		id, err := composite(req)
		require.NoError(t, err)
		assert.Empty(t, id)
	})

	t.Run("aggregates errors from resolvers", func(t *testing.T) {
		t.Parallel()

		// Create resolvers that return errors
		errorResolver1 := func(r *http.Request) (string, error) {
			return "", errors.New("resolver1 error")
		}
		errorResolver2 := func(r *http.Request) (string, error) {
			return "", errors.New("resolver2 error")
		}

		composite := tenant.NewCompositeResolver(errorResolver1, errorResolver2)
		req := httptest.NewRequest("GET", "/test", nil)

		_, err := composite(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "composite resolver errors")
		assert.Contains(t, err.Error(), "resolver1 error")
		assert.Contains(t, err.Error(), "resolver2 error")
	})

	t.Run("continues on error if later resolver succeeds", func(t *testing.T) {
		t.Parallel()

		errorResolver := func(r *http.Request) (string, error) {
			return "", errors.New("resolver error")
		}
		successResolver := func(r *http.Request) (string, error) {
			return "success-tenant", nil
		}

		composite := tenant.NewCompositeResolver(errorResolver, successResolver)
		req := httptest.NewRequest("GET", "/test", nil)

		id, err := composite(req)
		require.NoError(t, err)
		assert.Equal(t, "success-tenant", id)
	})

	t.Run("works with all resolver types", func(t *testing.T) {
		t.Parallel()

		composite := tenant.NewCompositeResolver(
			tenant.NewHeaderResolver("X-Tenant-ID"),
			tenant.NewSubdomainResolver(".app.com"),
			tenant.NewPathResolver(1),
		)

		// Test path resolution (others return empty)
		req := httptest.NewRequest("GET", "/tenant123/api", nil)
		req.Host = "api.com" // No subdomain

		id, err := composite(req)
		require.NoError(t, err)
		assert.Equal(t, "tenant123", id)
	})
}
