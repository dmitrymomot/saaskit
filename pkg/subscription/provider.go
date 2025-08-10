package subscription

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Common durations for all billing providers
const (
	DefaultCheckoutExpiry = 24 * time.Hour // Default checkout link expiration
	DefaultPortalExpiry   = 24 * time.Hour // Default portal link expiration
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

	// ParseWebhook validates and parses incoming webhook data from HTTP request.
	// Must validate signature headers to prevent webhook spoofing attacks.
	// Each provider looks for their specific signature headers (e.g., Paddle-Signature).
	// Returns normalized event type and raw provider data.
	ParseWebhook(r *http.Request) (*WebhookEvent, error)
}

// CheckoutRequest contains data needed to create a checkout session.
type CheckoutRequest struct {
	PriceID    string    // provider's price/plan identifier
	TenantID   uuid.UUID // your internal tenant ID
	Email      string    // optional billing email
	SuccessURL string    // redirect after successful payment
	CancelURL  string    // redirect if customer cancels
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
	TenantID       uuid.UUID      // your internal tenant ID (from custom_data)
	CustomerID     string         // provider's customer ID (ctm_xxx, cus_xxx, etc)
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
