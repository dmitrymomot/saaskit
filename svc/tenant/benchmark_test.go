package tenant_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/dmitrymomot/saaskit/svc/tenant"
)

// Benchmark individual resolver functions

func BenchmarkSubdomainResolver(b *testing.B) {
	resolver := tenant.NewSubdomainResolver(".app.com")
	req := httptest.NewRequest("GET", "https://acme.app.com/api/users", nil)
	req.Host = "acme.app.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id, err := resolver(req)
		if err != nil {
			b.Fatal(err)
		}
		if id != "acme" {
			b.Fatalf("expected acme, got %s", id)
		}
	}
}

func BenchmarkSubdomainResolverNoMatch(b *testing.B) {
	resolver := tenant.NewSubdomainResolver(".app.com")
	req := httptest.NewRequest("GET", "https://app.com/api/users", nil)
	req.Host = "app.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id, err := resolver(req)
		if err != nil {
			b.Fatal(err)
		}
		if id != "" {
			b.Fatalf("expected empty, got %s", id)
		}
	}
}

func BenchmarkHeaderResolver(b *testing.B) {
	resolver := tenant.NewHeaderResolver("X-Tenant-ID")
	req := httptest.NewRequest("GET", "https://api.app.com/users", nil)
	req.Header.Set("X-Tenant-ID", "acme")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id, err := resolver(req)
		if err != nil {
			b.Fatal(err)
		}
		if id != "acme" {
			b.Fatalf("expected acme, got %s", id)
		}
	}
}

func BenchmarkHeaderResolverNoMatch(b *testing.B) {
	resolver := tenant.NewHeaderResolver("X-Tenant-ID")
	req := httptest.NewRequest("GET", "https://api.app.com/users", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id, err := resolver(req)
		if err != nil {
			b.Fatal(err)
		}
		if id != "" {
			b.Fatalf("expected empty, got %s", id)
		}
	}
}

func BenchmarkPathResolver(b *testing.B) {
	resolver := tenant.NewPathResolver(2)
	req := httptest.NewRequest("GET", "/api/acme/users", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id, err := resolver(req)
		if err != nil {
			b.Fatal(err)
		}
		if id != "acme" {
			b.Fatalf("expected acme, got %s", id)
		}
	}
}

func BenchmarkPathResolverNoMatch(b *testing.B) {
	resolver := tenant.NewPathResolver(5) // Position beyond path segments
	req := httptest.NewRequest("GET", "/api/users", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id, err := resolver(req)
		if err != nil {
			b.Fatal(err)
		}
		if id != "" {
			b.Fatalf("expected empty, got %s", id)
		}
	}
}

func BenchmarkCompositeResolver(b *testing.B) {
	resolver := tenant.NewCompositeResolver(
		tenant.NewHeaderResolver("X-Tenant-ID"),
		tenant.NewSubdomainResolver(".app.com"),
		tenant.NewPathResolver(2),
	)

	// Test with header match (first resolver)
	req := httptest.NewRequest("GET", "https://api.app.com/users", nil)
	req.Header.Set("X-Tenant-ID", "acme")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id, err := resolver(req)
		if err != nil {
			b.Fatal(err)
		}
		if id != "acme" {
			b.Fatalf("expected acme, got %s", id)
		}
	}
}

func BenchmarkCompositeResolverFallthrough(b *testing.B) {
	resolver := tenant.NewCompositeResolver(
		tenant.NewHeaderResolver("X-Tenant-ID"),
		tenant.NewSubdomainResolver(".app.com"),
		tenant.NewPathResolver(2),
	)

	// Test with path match (last resolver) - no subdomain, no header
	req := httptest.NewRequest("GET", "/api/acme/users", nil)
	req.Host = "app.com" // Base domain, no subdomain

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id, err := resolver(req)
		if err != nil {
			b.Fatal(err)
		}
		if id != "acme" {
			b.Fatalf("expected acme, got %s", id)
		}
	}
}

// Benchmark middleware resolution flow

func BenchmarkMiddlewareResolution(b *testing.B) {
	// Setup mock provider
	mockProvider := new(mockProvider)
	testTenant := createTestTenant("acme", true)
	mockProvider.On("GetByIdentifier", mock.Anything, "acme").Return(testTenant, nil).Maybe()

	resolver := tenant.NewHeaderResolver("X-Tenant-ID")
	cache := &tenant.NoOpCache{}
	middleware := tenant.Middleware(resolver, mockProvider, tenant.WithCache(cache))

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ten, ok := tenant.FromContext(r.Context())
		if !ok {
			b.Fatal("tenant not found in context")
		}
		if ten.Subdomain != "acme" {
			b.Fatalf("expected tenant subdomain acme, got %s", ten.Subdomain)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Tenant-ID", "acme")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("expected status 200, got %d", w.Code)
		}
	}
}

// Benchmark context operations

func BenchmarkContextOperations(b *testing.B) {
	testTenant := &tenant.Tenant{
		ID:        uuid.New(),
		Name:      "Acme Corp",
		Subdomain: "acme",
		Active:    true,
		PlanID:    "pro",
		CreatedAt: time.Now(),
	}

	b.Run("WithTenant", func(b *testing.B) {
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			newCtx := tenant.WithTenant(ctx, testTenant)
			if newCtx == nil {
				b.Fatal("context is nil")
			}
		}
	})

	b.Run("FromContext", func(b *testing.B) {
		ctx := tenant.WithTenant(context.Background(), testTenant)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ten, ok := tenant.FromContext(ctx)
			if !ok {
				b.Fatal("tenant not found")
			}
			if ten.Subdomain != "acme" {
				b.Fatalf("expected tenant subdomain acme, got %s", ten.Subdomain)
			}
		}
	})
}

// Benchmark validation through resolvers (validation functions are not exported)

func BenchmarkValidation_ValidPath(b *testing.B) {
	resolver := tenant.NewPathResolver(1)
	req := httptest.NewRequest("GET", "/valid-tenant-123/users", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id, err := resolver(req)
		if err != nil {
			b.Fatal(err)
		}
		if id != "valid-tenant-123" {
			b.Fatalf("expected valid-tenant-123, got %s", id)
		}
	}
}

func BenchmarkValidation_InvalidPath(b *testing.B) {
	resolver := tenant.NewPathResolver(1)
	req := httptest.NewRequest("GET", "/.invalid-start/users", nil) // Invalid: starts with dot

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver(req)
		if err == nil {
			b.Fatal("expected error for invalid path")
		}
	}
}

func BenchmarkValidation_ValidSubdomain(b *testing.B) {
	resolver := tenant.NewSubdomainResolver(".app.com")
	req := httptest.NewRequest("GET", "https://valid-tenant-123.app.com/users", nil)
	req.Host = "valid-tenant-123.app.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id, err := resolver(req)
		if err != nil {
			b.Fatal(err)
		}
		if id != "valid-tenant-123" {
			b.Fatalf("expected valid-tenant-123, got %s", id)
		}
	}
}

func BenchmarkValidation_InvalidSubdomain(b *testing.B) {
	resolver := tenant.NewSubdomainResolver(".app.com")
	req := httptest.NewRequest("GET", "https://-invalid.app.com/users", nil) // Invalid: starts with hyphen
	req.Host = "-invalid.app.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver(req)
		if err == nil {
			b.Fatal("expected error for invalid subdomain")
		}
	}
}

func BenchmarkValidation_MaxLengthPath(b *testing.B) {
	// Test with maximum allowed length (63 characters)
	maxLengthID := "a12345678901234567890123456789012345678901234567890123456789012" // 63 chars
	resolver := tenant.NewPathResolver(1)
	req := httptest.NewRequest("GET", "/"+maxLengthID+"/users", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id, err := resolver(req)
		if err != nil {
			b.Fatal(err)
		}
		if id != maxLengthID {
			b.Fatalf("expected %s, got %s", maxLengthID, id)
		}
	}
}

func BenchmarkValidation_TooLongPath(b *testing.B) {
	// Test with length exceeding maximum (64 characters)
	tooLongID := "a123456789012345678901234567890123456789012345678901234567890123" // 64 chars
	resolver := tenant.NewPathResolver(1)
	req := httptest.NewRequest("GET", "/"+tooLongID+"/users", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver(req)
		if err == nil {
			b.Fatal("expected error for too long path")
		}
	}
}
