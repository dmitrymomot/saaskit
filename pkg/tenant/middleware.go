package tenant

import (
	"log/slog"
	"net/http"
	"strings"
)

// Middleware extracts tenant information from requests and adds it to context.
// Supports caching, path skipping, and configurable error handling.
func Middleware(resolver Resolver, provider Provider, opts ...Option) func(http.Handler) http.Handler {
	cfg := &config{
		cache:         &NoOpCache{},
		errorHandler:  defaultErrorHandler,
		requireActive: true,
		logger:        slog.Default(),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip tenant resolution for configured paths (health checks, etc.)
			for _, skip := range cfg.skipPaths {
				if strings.HasPrefix(r.URL.Path, skip) {
					next.ServeHTTP(w, r)
					return
				}
			}

			identifier, err := resolver(r)
			if err != nil {
				cfg.errorHandler(w, r, err)
				return
			}

			// Allow requests without tenant identification
			if identifier == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check cache to avoid database lookup
			if cached, ok := cfg.cache.Get(r.Context(), identifier); ok {
				if cfg.requireActive && !cached.Active {
					cfg.errorHandler(w, r, ErrInactiveTenant)
					return
				}

				ctx := WithTenant(r.Context(), cached)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			tenant, err := provider.GetByIdentifier(r.Context(), identifier)
			if err != nil {
				cfg.errorHandler(w, r, err)
				return
			}

			if cfg.requireActive && !tenant.Active {
				cfg.errorHandler(w, r, ErrInactiveTenant)
				return
			}

			// Cache failures are logged but don't block requests
			if err := cfg.cache.Set(r.Context(), identifier, tenant); err != nil {
				cfg.logger.WarnContext(r.Context(), "failed to cache tenant",
					"tenant_id", identifier,
					"error", err)
			}

			ctx := WithTenant(r.Context(), tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireTenant ensures a tenant is present in context, useful for protecting tenant-only routes.
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
