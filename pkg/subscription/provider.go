package subscription

import (
	"context"
	"time"
)

// BillingProvider defines the minimal interface for payment provider integrations.
// This abstraction allows support for different providers (Paddle, Stripe, Lemonsqueezy)
// while avoiding vendor lock-in. Provider handles all payment complexity through
// hosted checkouts and customer portals, eliminating PCI compliance concerns.
//
// Implementations should use official provider SDKs and handle provider-specific
// quirks internally (e.g., Paddle's customer ID mapping, Stripe's metadata fields).
type BillingProvider interface {
	// CreateCheckoutLink creates a hosted checkout session
	CreateCheckoutLink(ctx context.Context, req CheckoutRequest) (*CheckoutLink, error)

	// GetCustomerPortalLink returns a temporary link to the customer portal
	// where users can update payment methods, cancel, or change plans.
	// The provider implementation decides which fields to use (e.g., Paddle uses TenantID as customer ID)
	GetCustomerPortalLink(ctx context.Context, subscription *Subscription) (*PortalLink, error)

	// ParseWebhook validates and parses incoming webhook data.
	// Must validate signature to prevent webhook spoofing attacks.
	// Returns normalized event type and raw provider data
	ParseWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error)
}

// CheckoutRequest contains data needed to create a checkout session.
type CheckoutRequest struct {
	PriceID    string // provider's price/plan identifier
	CustomerID string // your internal user/tenant ID
	Email      string // optional billing email
	SuccessURL string // redirect after successful payment
	CancelURL  string // redirect if customer cancels
}

// CheckoutLink represents a hosted checkout session.
type CheckoutLink struct {
	URL       string    // hosted checkout URL
	SessionID string    // provider's session identifier
	ExpiresAt time.Time // link expiration
}

// PortalLink represents a customer portal session with optional action-specific URLs.
type PortalLink struct {
	URL              string    // general portal URL (always populated)
	CancelURL        string    // optional: direct to cancellation flow
	UpdatePaymentURL string    // optional: direct to payment method update
	ExpiresAt        time.Time // link expiration (usually 24 hours)
}

// WebhookEvent represents a normalized webhook event from the billing provider.
type WebhookEvent struct {
	Type           EventType      // normalized event type
	ProviderEvent  string         // original provider event name
	SubscriptionID string         // provider's subscription ID
	CustomerID     string         // your user ID from metadata
	Status         string         // subscription status
	PlanID         string         // the plan/price they subscribed to
	Raw            map[string]any // full webhook data
}

// EventType represents the normalized billing event type.
// Each provider implementation maps their specific events to these types.
type EventType string

const (
	EventSubscriptionCreated   EventType = "subscription_created"
	EventSubscriptionUpdated   EventType = "subscription_updated"
	EventSubscriptionCancelled EventType = "subscription_cancelled"
	EventSubscriptionResumed   EventType = "subscription_resumed"

	EventPaymentSucceeded EventType = "payment_succeeded"
	EventPaymentFailed    EventType = "payment_failed"
)
