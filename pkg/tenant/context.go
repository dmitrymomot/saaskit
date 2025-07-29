package tenant

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// contextKey is a private type to prevent collisions with other context keys.
type contextKey struct{}

// WithTenant adds a tenant to the context.
func WithTenant(ctx context.Context, tenant *Tenant) context.Context {
	return context.WithValue(ctx, contextKey{}, tenant)
}

// FromContext retrieves the tenant from the context.
// Returns nil, false if no tenant is found.
func FromContext(ctx context.Context) (*Tenant, bool) {
	tenant, ok := ctx.Value(contextKey{}).(*Tenant)
	return tenant, ok
}

// IDFromContext retrieves just the tenant ID from the context.
// Returns zero UUID and false if no tenant is found.
func IDFromContext(ctx context.Context) (uuid.UUID, bool) {
	tenant, ok := FromContext(ctx)
	if !ok || tenant == nil {
		return uuid.UUID{}, false
	}
	return tenant.ID, true
}

// MustFromContext retrieves the tenant from the context.
// Panics if no tenant is found. Use this only in handlers
// that absolutely require a tenant to function.
func MustFromContext(ctx context.Context) *Tenant {
	tenant, ok := FromContext(ctx)
	if !ok || tenant == nil {
		panic("tenant: no tenant in context")
	}
	return tenant
}

// LoggerExtractor returns a ContextExtractor for the logger that extracts tenant ID from context
func LoggerExtractor() func(ctx context.Context) (slog.Attr, bool) {
	return func(ctx context.Context) (slog.Attr, bool) {
		if id, ok := IDFromContext(ctx); ok {
			return slog.String("tenant_id", id.String()), true
		}
		return slog.Attr{}, false
	}
}
