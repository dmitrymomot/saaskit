# tenant

Multi-tenancy support for SaaS applications with flexible tenant resolution and context propagation.

## Features

- Multiple resolver strategies (subdomain, header, path, session)
- Request-scoped tenant context propagation
- Built-in caching with configurable TTL
- Automatic inactive tenant blocking

## Installation

```go
import "github.com/dmitrymomot/saaskit/svc/tenant"
```

## Usage

```go
// Implement Provider interface for your database
type myProvider struct {
    db *sql.DB
}

func (p *myProvider) GetByIdentifier(ctx context.Context, id string) (*tenant.Tenant, error) {
    var t tenant.Tenant
    err := p.db.QueryRow(`
        SELECT id, subdomain, name, logo_url, plan_id, active, created_at
        FROM tenants
        WHERE subdomain = $1 OR id::text = $1
    `, id).Scan(&t.ID, &t.Subdomain, &t.Name, &t.Logo, &t.PlanID, &t.Active, &t.CreatedAt)

    if err == sql.ErrNoRows {
        return nil, tenant.ErrTenantNotFound
    }
    return &t, err
}

// Setup middleware
provider := &myProvider{db: db}
resolver := tenant.NewSubdomainResolver(".app.com")

router.Use(tenant.Middleware(resolver, provider,
    tenant.WithCacheTTL(10*time.Minute),
    tenant.WithSkipPaths([]string{"/health", "/metrics"}),
))
```

## Common Operations

### Using Tenant in Handlers

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Get tenant from context
    t, ok := tenant.FromContext(r.Context())
    if !ok {
        http.Error(w, "Tenant required", http.StatusForbidden)
        return
    }

    // Use tenant for data isolation
    db.Query("SELECT * FROM items WHERE tenant_id = $1", t.ID)
}
```

### Multiple Resolver Strategies

```go
// Try subdomain first, then header
resolver := tenant.NewCompositeResolver(
    tenant.NewSubdomainResolver(".app.com"),      // acme.app.com
    tenant.NewHeaderResolver("X-Tenant-ID"),       // X-Tenant-ID: 123
    tenant.NewPathResolver(2),                     // /tenants/123/dashboard
)

// Custom resolver
custom := tenant.ResolverFunc(func(r *http.Request) (string, error) {
    return r.URL.Query().Get("tenant"), nil
})
```

### Protecting Routes

```go
// Require tenant for specific routes
protectedRoutes := router.Group("/api")
protectedRoutes.Use(tenant.RequireTenant(nil))
```

## Error Handling

```go
// Package errors:
var (
    ErrTenantNotFound    = errors.New("tenant not found")
    ErrInvalidIdentifier = errors.New("invalid tenant identifier")
    ErrNoTenantInContext = errors.New("no tenant in context")
    ErrInactiveTenant    = errors.New("tenant is inactive")
)

// Custom error handler
tenant.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
    if errors.Is(err, tenant.ErrTenantNotFound) {
        http.Error(w, "Organization not found", http.StatusNotFound)
    }
})
```

## Configuration

```go
// Middleware options
tenant.Middleware(resolver, provider,
    tenant.WithCache(customCache),                 // Custom cache implementation
    tenant.WithCacheTTL(10*time.Minute),          // Cache duration
    tenant.WithErrorHandler(customHandler),        // Error handling
    tenant.WithSkipPaths([]string{"/public"}),    // Skip paths
    tenant.WithRequireActive(false),              // Allow inactive tenants
)
```

## API Documentation

```bash
# Full API documentation
go doc github.com/dmitrymomot/saaskit/svc/tenant

# Specific function or type
go doc github.com/dmitrymomot/saaskit/svc/tenant.Middleware
```

## Notes

- Thread-safe implementation with zero external dependencies
- Framework-agnostic - works with any Go HTTP router
- Cache implementation can be replaced with Redis/Memcached for distributed systems
- Designed for complete tenant isolation in multi-tenant SaaS applications
