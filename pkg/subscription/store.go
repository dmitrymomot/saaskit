package subscription

import (
	"context"

	"github.com/google/uuid"
)

// SubscriptionStore defines the interface for subscription persistence.
// Each tenant has exactly one subscription, so TenantID serves as the primary key.
type SubscriptionStore interface {
	// Get retrieves a subscription by tenant ID.
	// Returns ErrSubscriptionNotFound if no subscription exists.
	Get(ctx context.Context, tenantID uuid.UUID) (*Subscription, error)

	// Save creates or updates a subscription.
	// The implementation should use TenantID to determine if it's an update.
	Save(ctx context.Context, subscription *Subscription) error
}
