package tenant

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// contextKey prevents collisions with other packages using context values
type contextKey struct{}

func WithTenant(ctx context.Context, tenant *Tenant) context.Context {
	return context.WithValue(ctx, contextKey{}, tenant)
}

func FromContext(ctx context.Context) (*Tenant, bool) {
	tenant, ok := ctx.Value(contextKey{}).(*Tenant)
	return tenant, ok
}

// IDFromContext provides fast access to tenant ID without exposing full tenant data
func IDFromContext(ctx context.Context) (uuid.UUID, bool) {
	tenant, ok := FromContext(ctx)
	if !ok || tenant == nil {
		return uuid.UUID{}, false
	}
	return tenant.ID, true
}

// MustFromContext panics if no tenant is found. Use only in handlers
// that absolutely require a tenant to function.
func MustFromContext(ctx context.Context) *Tenant {
	tenant, ok := FromContext(ctx)
	if !ok || tenant == nil {
		panic("tenant: no tenant in context")
	}
	return tenant
}

// LoggerExtractor returns a function that enriches log records with tenant ID
func LoggerExtractor() func(ctx context.Context) (slog.Attr, bool) {
	return func(ctx context.Context) (slog.Attr, bool) {
		if id, ok := IDFromContext(ctx); ok {
			return slog.String("tenant_id", id.String()), true
		}
		return slog.Attr{}, false
	}
}
