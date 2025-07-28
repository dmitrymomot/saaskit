package tenant

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Tenant represents a tenant in the system with minimal information
// needed for request-scoped operations and UI display.
type Tenant struct {
	ID        uuid.UUID `json:"id"`
	Subdomain string    `json:"subdomain"`
	Name      string    `json:"name"`
	Logo      string    `json:"logo_url"`
	PlanID    string    `json:"plan_id"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// Provider loads tenant information from a data source.
// Implementations should handle various identifier formats
// (UUID, subdomain, etc.) based on application needs.
type Provider interface {
	// GetByIdentifier retrieves a tenant using any unique identifier.
	// The identifier could be a UUID, subdomain, or any other unique field.
	// Returns ErrTenantNotFound if no tenant matches the identifier.
	GetByIdentifier(ctx context.Context, identifier string) (*Tenant, error)
}
