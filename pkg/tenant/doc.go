// Package tenant provides multi-tenancy support for SaaS applications through flexible tenant identification and context management.
//
// The package offers a complete solution for extracting tenant information from HTTP requests,
// caching tenant data, and propagating tenant context throughout the application. It supports
// multiple identification strategies including subdomain, header, path, and session-based resolution.
//
// # Architecture
//
// The package is built around three core concepts:
//
// 1. Resolvers - Extract tenant identifiers from HTTP requests using various strategies
// 2. Providers - Load full tenant information from a data source
// 3. Middleware - Orchestrates resolution, caching, and context propagation
//
// This separation allows applications to mix and match identification strategies
// while keeping the tenant loading logic independent.
//
// # Usage
//
//	import "github.com/dmitrymomot/saaskit/pkg/tenant"
//
//	// Create a resolver (e.g., subdomain-based)
//	resolver := tenant.NewSubdomainResolver(".saas.com")
//
//	// Implement a provider to load tenant data
//	provider := &myTenantProvider{}
//
//	// Create middleware with caching
//	mw := tenant.Middleware(resolver, provider,
//		tenant.WithCacheTTL(10*time.Minute),
//		tenant.WithSkipPaths([]string{"/health", "/metrics"}),
//	)
//
//	// Apply to your router
//	router.Use(mw)
//
//	// Access tenant in handlers
//	func handler(w http.ResponseWriter, r *http.Request) {
//		tenant, ok := tenant.FromContext(r.Context())
//		if !ok {
//			// Handle no tenant case
//			return
//		}
//		// Use tenant data
//	}
//
// # Resolver Strategies
//
// The package includes several built-in resolvers:
//
// - SubdomainResolver: Extracts tenant from subdomain (e.g., "acme" from "acme.app.com")
// - HeaderResolver: Reads tenant from HTTP header (e.g., "X-Tenant-ID")
// - PathResolver: Extracts from URL path segment (e.g., "/tenants/{id}/dashboard")
// - SessionResolver: Retrieves from session data for user-switchable tenancy
// - CompositeResolver: Tries multiple strategies in order
//
// Custom resolvers can be created by implementing the Resolver interface.
//
// # Caching
//
// The middleware includes automatic caching to reduce database lookups. The default
// in-memory cache handles TTL expiration and concurrent access. Custom cache
// implementations can be provided via the Cache interface for Redis or other backends.
//
// # Error Handling
//
// The package defines specific errors for common failure scenarios:
//
//   - ErrTenantNotFound: Tenant does not exist
//   - ErrInactiveTenant: Tenant exists but is not active
//   - ErrNoTenantInContext: Required tenant is missing from context
//   - ErrInvalidIdentifier: Malformed tenant identifier
//
// Custom error handlers can be configured to return appropriate HTTP responses.
//
// # Security Considerations
//
// - Always validate tenant access in handlers for sensitive operations
// - Use RequireTenant middleware to protect tenant-specific routes
// - Consider rate limiting per tenant to prevent abuse
// - Validate tenant state (active/inactive) based on business rules
//
// # Performance
//
// The middleware is designed for high-throughput applications:
//
// - Configurable caching reduces database queries
// - Concurrent-safe cache implementation
// - Minimal allocations in the request path
// - Skip paths to bypass resolution for public routes
//
// # Examples
//
// See the package examples and README.md for detailed usage patterns including
// session-based multi-tenancy, custom providers, and testing strategies.
package tenant
