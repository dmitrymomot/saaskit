package tenant

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

// Middleware creates HTTP middleware that extracts tenant information
// from incoming requests and adds it to the request context.
func Middleware(resolver Resolver, provider Provider, opts ...Option) func(http.Handler) http.Handler {
	// Apply default configuration
	cfg := &config{
		cache:         NewInMemoryCache(),
		cacheTTL:      5 * time.Minute,
		errorHandler:  defaultErrorHandler,
		requireActive: true,
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if we should skip this path
			for _, skip := range cfg.skipPaths {
				if strings.HasPrefix(r.URL.Path, skip) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Step 1: Resolve tenant identifier
			identifier, err := resolver.Resolve(r)
			if err != nil {
				cfg.errorHandler(w, r, err)
				return
			}

			// If no identifier found, continue without tenant
			if identifier == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Step 2: Check cache first
			if cached, ok := cfg.cache.Get(r.Context(), identifier); ok {
				// Validate cached tenant
				if cfg.requireActive && !cached.Active {
					cfg.errorHandler(w, r, ErrInactiveTenant)
					return
				}

				ctx := WithTenant(r.Context(), cached)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Step 3: Load from provider
			tenant, err := provider.GetByIdentifier(r.Context(), identifier)
			if err != nil {
				if errors.Is(err, ErrTenantNotFound) {
					cfg.errorHandler(w, r, err)
					return
				}
				cfg.errorHandler(w, r, err)
				return
			}

			// Step 4: Validate tenant
			if cfg.requireActive && !tenant.Active {
				cfg.errorHandler(w, r, ErrInactiveTenant)
				return
			}

			// Step 5: Cache the tenant
			cfg.cache.Set(r.Context(), identifier, tenant, cfg.cacheTTL)

			// Step 6: Add to context and continue
			ctx := WithTenant(r.Context(), tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireTenant creates middleware that ensures a tenant is present in the context.
// This is useful for protecting routes that require tenant context.
func RequireTenant(errorHandler ErrorHandler) func(http.Handler) http.Handler {
	if errorHandler == nil {
		errorHandler = defaultErrorHandler
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenant, ok := FromContext(r.Context())
			if !ok || tenant == nil {
				errorHandler(w, r, ErrNoTenantInContext)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
