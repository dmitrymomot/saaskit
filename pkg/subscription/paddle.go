package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	paddle "github.com/PaddleHQ/paddle-go-sdk/v4"
	"github.com/google/uuid"
)

// PaddleConfig holds configuration for Paddle billing provider.
type PaddleConfig struct {
	APIKey        string `env:"PADDLE_API_KEY,required"`
	WebhookSecret string `env:"PADDLE_WEBHOOK_SECRET,required"`
	Environment   string `env:"PADDLE_ENVIRONMENT" envDefault:"production"`
}

// Validate checks if the configuration is valid.
func (c PaddleConfig) Validate() error {
	if c.APIKey == "" {
		return ErrMissingAPIKey
	}
	if c.WebhookSecret == "" {
		return ErrMissingWebhookSecret
	}
	env := strings.ToLower(c.Environment)
	if env != "" && env != "sandbox" && env != "production" {
		return ErrInvalidProviderEnvironment
	}
	return nil
}

// PaddleProvider implements BillingProvider for Paddle.
type PaddleProvider struct {
	client   *paddle.SDK
	verifier *paddle.WebhookVerifier
	config   PaddleConfig
}

// NewPaddleProvider creates a new Paddle billing provider.
func NewPaddleProvider(config PaddleConfig) (*PaddleProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	var client *paddle.SDK
	var err error

	switch strings.ToLower(config.Environment) {
	case "sandbox":
		client, err = paddle.NewSandbox(config.APIKey)
	case "production", "":
		client, err = paddle.New(config.APIKey)
	default:
		return nil, ErrInvalidProviderEnvironment
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create paddle client: %w", err)
	}

	verifier := paddle.NewWebhookVerifier(config.WebhookSecret)

	return &PaddleProvider{
		client:   client,
		verifier: verifier,
		config:   config,
	}, nil
}

// CreateCheckoutLink creates a hosted checkout session in Paddle.
func (p *PaddleProvider) CreateCheckoutLink(ctx context.Context, req CheckoutRequest) (*CheckoutLink, error) {
	if req.PriceID == "" {
		return nil, ErrMissingPriceID
	}
	if req.TenantID == uuid.Nil {
		return nil, ErrMissingTenantID
	}

	// Create transaction item from catalog
	item := paddle.NewCreateTransactionItemsTransactionItemFromCatalog(&paddle.TransactionItemFromCatalog{
		PriceID:  req.PriceID,
		Quantity: 1,
	})

	// Create transaction request with custom data
	transactionReq := &paddle.CreateTransactionRequest{
		Items: []paddle.CreateTransactionItems{*item},
		CustomData: paddle.CustomData{
			"tenant_id": req.TenantID.String(),
		},
	}

	// Add customer email if provided
	if req.Email != "" {
		// Store email in custom_data for reference
		transactionReq.CustomData["email"] = req.Email
	}

	// Add checkout configuration
	if req.SuccessURL != "" {
		transactionReq.Checkout = &paddle.TransactionCheckout{
			URL: paddle.PtrTo(req.SuccessURL),
		}
	}

	// Create the transaction
	transaction, err := p.client.CreateTransaction(ctx, transactionReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create paddle transaction: %w", err)
	}

	// Extract checkout URL
	var checkoutURL string
	if transaction.Checkout != nil && transaction.Checkout.URL != nil {
		checkoutURL = *transaction.Checkout.URL
	} else {
		return nil, ErrNoCheckoutURL
	}

	return &CheckoutLink{
		URL:       checkoutURL,
		SessionID: transaction.ID,
		ExpiresAt: time.Now().Add(DefaultCheckoutExpiry),
	}, nil
}

// GetCustomerPortalLink returns a link to Paddle's customer portal.
func (p *PaddleProvider) GetCustomerPortalLink(ctx context.Context, subscription *Subscription) (*PortalLink, error) {
	if subscription == nil {
		return nil, ErrSubscriptionNotFound
	}
	if subscription.ProviderSubID == "" {
		return nil, fmt.Errorf("subscription provider ID is required")
	}
	if subscription.ProviderCustomerID == "" {
		return nil, ErrMissingProviderCustomerID
	}

	// Create a customer portal session request
	portalSessionReq := &paddle.CreateCustomerPortalSessionRequest{
		CustomerID:      subscription.ProviderCustomerID,
		SubscriptionIDs: []string{subscription.ProviderSubID},
	}

	portalSession, err := p.client.CreateCustomerPortalSession(ctx, portalSessionReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create paddle customer portal session: %w", err)
	}

	// Start with the general overview URL for the customer portal
	portalLink := &PortalLink{
		ExpiresAt: time.Now().Add(DefaultPortalExpiry),
	}

	// Set the general portal URL
	if portalSession.URLs.General.Overview != "" {
		portalLink.URL = portalSession.URLs.General.Overview
	}

	// Look for subscription-specific action URLs
	if len(portalSession.URLs.Subscriptions) > 0 {
		// Find the URLs for our specific subscription
		for _, subURL := range portalSession.URLs.Subscriptions {
			if subURL.ID == subscription.ProviderSubID {
				// Set cancel URL if available
				if subURL.CancelSubscription != "" {
					portalLink.CancelURL = subURL.CancelSubscription
				}
				// Set update payment method URL if available
				if subURL.UpdateSubscriptionPaymentMethod != "" {
					portalLink.UpdatePaymentURL = subURL.UpdateSubscriptionPaymentMethod
				}
				break
			}
		}
	}

	// Ensure we have at least the general URL
	if portalLink.URL == "" {
		return nil, ErrNoPortalURL
	}

	return portalLink, nil
}

// ParseWebhook validates and parses incoming webhook data from HTTP request.
func (p *PaddleProvider) ParseWebhook(req *http.Request) (*WebhookEvent, error) {
	// Verify the webhook signature
	valid, err := p.verifier.Verify(req)
	if err != nil {
		return nil, fmt.Errorf("webhook verification error: %w", err)
	}
	if !valid {
		return nil, ErrWebhookVerificationFailed
	}

	// Read the request body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	// Parse the webhook payload
	var paddleEvent paddleWebhookEvent
	if err := json.Unmarshal(body, &paddleEvent); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	return p.extractWebhookData(paddleEvent)
}

// paddleWebhookEvent represents the structure of a Paddle webhook event.
type paddleWebhookEvent struct {
	EventID    string         `json:"event_id"`
	EventType  string         `json:"event_type"`
	OccurredAt string         `json:"occurred_at"`
	Data       map[string]any `json:"data"`
}

// extractWebhookData extracts relevant data from a Paddle webhook event.
func (p *PaddleProvider) extractWebhookData(paddleEvent paddleWebhookEvent) (*WebhookEvent, error) {
	event := &WebhookEvent{
		Type:          mapPaddleEventType(paddleEvent.EventType),
		ProviderEvent: paddleEvent.EventType,
		Raw:           paddleEvent.Data,
	}

	// Handle subscription events
	if strings.HasPrefix(paddleEvent.EventType, "subscription.") {
		// Extract provider's customer ID
		if custID, ok := paddleEvent.Data["customer_id"].(string); ok {
			event.CustomerID = custID
		}

		// Extract and map status
		if status, ok := paddleEvent.Data["status"].(string); ok {
			event.Status = string(mapPaddleStatus(status))
		}

		// Extract tenant ID from custom_data
		if customData, ok := paddleEvent.Data["custom_data"].(map[string]any); ok {
			if tenantIDStr, ok := customData["tenant_id"].(string); ok {
				if tenantID, err := uuid.Parse(tenantIDStr); err == nil {
					event.TenantID = tenantID
				}
			}
		}

		// Extract subscription ID
		if subID, ok := paddleEvent.Data["id"].(string); ok {
			event.SubscriptionID = subID
		}

		// Extract plan/price ID from items
		event.PlanID = extractPriceIDFromItems(paddleEvent.Data["items"])
	}

	// Handle transaction events
	if strings.HasPrefix(paddleEvent.EventType, "transaction.") {
		// Extract transaction ID as subscription ID for transaction events
		if txnID, ok := paddleEvent.Data["id"].(string); ok {
			event.SubscriptionID = txnID
		}

		// Extract subscription ID if this transaction is related to a subscription
		if subID, ok := paddleEvent.Data["subscription_id"].(string); ok {
			event.SubscriptionID = subID
		}

		// Extract provider's customer ID
		if custID, ok := paddleEvent.Data["customer_id"].(string); ok {
			event.CustomerID = custID
		}

		// Extract and map status
		if status, ok := paddleEvent.Data["status"].(string); ok {
			event.Status = string(mapPaddleStatus(status))
		}

		// Extract tenant ID from custom_data
		if customData, ok := paddleEvent.Data["custom_data"].(map[string]any); ok {
			if tenantIDStr, ok := customData["tenant_id"].(string); ok {
				if tenantID, err := uuid.Parse(tenantIDStr); err == nil {
					event.TenantID = tenantID
				}
			}
		}

		// Extract plan/price ID from items
		if items, ok := paddleEvent.Data["items"].([]any); ok && len(items) > 0 {
			if item, ok := items[0].(map[string]any); ok {
				if priceID, ok := item["price_id"].(string); ok {
					event.PlanID = priceID
				}
			}
		}
	}

	return event, nil
}

// extractPriceIDFromItems extracts the price ID from webhook items array.
func extractPriceIDFromItems(items any) string {
	itemsArray, ok := items.([]any)
	if !ok || len(itemsArray) == 0 {
		return ""
	}

	if item, ok := itemsArray[0].(map[string]any); ok {
		// Try to get price ID from nested price object
		if price, ok := item["price"].(map[string]any); ok {
			if priceID, ok := price["id"].(string); ok {
				return priceID
			}
		}
		// Or directly from price_id field
		if priceID, ok := item["price_id"].(string); ok {
			return priceID
		}
	}
	return ""
}

// mapPaddleEventType maps Paddle event types to internal EventType.
func mapPaddleEventType(paddleEvent string) EventType {
	switch paddleEvent {
	case "transaction.completed":
		return EventSubscriptionCreated
	case "subscription.created":
		return EventSubscriptionCreated
	case "subscription.updated":
		return EventSubscriptionUpdated
	case "subscription.canceled":
		return EventSubscriptionCancelled
	case "subscription.resumed":
		return EventSubscriptionResumed
	case "transaction.payment_succeeded":
		return EventPaymentSucceeded
	case "transaction.payment_failed":
		return EventPaymentFailed
	default:
		// Return the original event as EventType for unmapped events
		return EventType(paddleEvent)
	}
}

// mapPaddleStatus maps Paddle subscription status to internal SubscriptionStatus.
func mapPaddleStatus(paddleStatus string) SubscriptionStatus {
	switch strings.ToLower(paddleStatus) {
	case "trialing":
		return StatusTrialing
	case "active":
		return StatusActive
	case "past_due":
		return StatusPastDue
	case "canceled", "cancelled":
		return StatusCancelled
	case "expired":
		return StatusExpired
	default:
		// Return as-is for unknown statuses
		return SubscriptionStatus(paddleStatus)
	}
}
