package subscription

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	paddle "github.com/PaddleHQ/paddle-go-sdk/v4"
)

// PaddleConfig holds configuration for Paddle billing provider.
type PaddleConfig struct {
	APIKey        string `env:"PADDLE_API_KEY,required"`
	WebhookSecret string `env:"PADDLE_WEBHOOK_SECRET,required"`
	Environment   string `env:"PADDLE_ENVIRONMENT" envDefault:"production"`
}

// PaddleProvider implements BillingProvider for Paddle.
type PaddleProvider struct {
	client   *paddle.SDK
	verifier *paddle.WebhookVerifier
	config   PaddleConfig
}

// NewPaddleProvider creates a new Paddle billing provider.
func NewPaddleProvider(config PaddleConfig) (*PaddleProvider, error) {
	if config.APIKey == "" {
		return nil, errors.New("paddle API key is required")
	}
	if config.WebhookSecret == "" {
		return nil, errors.New("paddle webhook secret is required")
	}

	var client *paddle.SDK
	var err error

	switch strings.ToLower(config.Environment) {
	case "sandbox":
		client, err = paddle.NewSandbox(config.APIKey)
	case "production", "":
		client, err = paddle.New(config.APIKey)
	default:
		return nil, fmt.Errorf("invalid paddle environment: %s", config.Environment)
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
		return nil, errors.New("price ID is required")
	}
	if req.CustomerID == "" {
		return nil, errors.New("customer ID is required")
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
			"customer_id": req.CustomerID,
		},
	}

	// Add customer email if provided
	if req.Email != "" {
		// In Paddle, CustomerID is the Paddle customer ID, not email
		// Email should be set through customer creation or update
		// For now, we'll add it to custom data
		transactionReq.CustomData["email"] = req.Email
	}

	// Add checkout configuration
	if req.SuccessURL != "" {
		transactionReq.Checkout = &paddle.TransactionCheckout{
			URL: paddle.PtrTo(req.SuccessURL),
		}
	}

	// Create the transaction
	transaction, err := p.client.TransactionsClient.CreateTransaction(ctx, transactionReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create paddle transaction: %w", err)
	}

	// Extract checkout URL
	var checkoutURL string
	if transaction.Checkout != nil && transaction.Checkout.URL != nil {
		checkoutURL = *transaction.Checkout.URL
	} else {
		return nil, errors.New("no checkout URL returned from paddle")
	}

	return &CheckoutLink{
		URL:       checkoutURL,
		SessionID: transaction.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour), // Paddle checkout links typically expire in 24 hours
	}, nil
}

// GetCustomerPortalLink returns a link to Paddle's customer portal.
func (p *PaddleProvider) GetCustomerPortalLink(ctx context.Context, subscription *Subscription) (*PortalLink, error) {
	if subscription == nil {
		return nil, errors.New("subscription is required")
	}
	if subscription.ProviderSubID == "" {
		return nil, errors.New("subscription provider ID is required")
	}

	// For Paddle, we need the actual Paddle customer ID (ctm_xxx)
	// This should be stored somewhere - for now we'll use the TenantID as a string
	customerID := subscription.TenantID.String()

	// Create a customer portal session request
	portalSessionReq := &paddle.CreateCustomerPortalSessionRequest{
		CustomerID:      customerID, // This should be the Paddle customer ID (ctm_xxx)
		SubscriptionIDs: []string{subscription.ProviderSubID},
	}

	portalSession, err := p.client.CustomerPortalSessionsClient.CreateCustomerPortalSession(ctx, portalSessionReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create paddle customer portal session: %w", err)
	}

	// Start with the general overview URL for the customer portal
	portalLink := &PortalLink{
		ExpiresAt: time.Now().Add(24 * time.Hour), // Portal links typically expire in 24 hours
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
		return nil, errors.New("no portal URL returned from paddle")
	}

	return portalLink, nil
}

// ParseWebhook validates and parses incoming webhook data from Paddle.
// Note: This method signature differs from the interface to match Paddle SDK requirements.
// You should create an HTTP request with the webhook payload and signature header before calling this.
func (p *PaddleProvider) ParseWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error) {
	// Create an HTTP request for verification
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/webhook", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request for verification: %w", err)
	}
	req.Header.Set("Paddle-Signature", signature)

	// Verify the webhook signature
	valid, err := p.verifier.Verify(req)
	if err != nil {
		return nil, fmt.Errorf("webhook verification error: %w", err)
	}
	if !valid {
		return nil, errors.New("webhook signature verification failed")
	}

	// Parse the webhook payload
	var paddleEvent struct {
		EventID    string         `json:"event_id"`
		EventType  string         `json:"event_type"`
		OccurredAt string         `json:"occurred_at"`
		Data       map[string]any `json:"data"`
	}

	if err := json.Unmarshal(payload, &paddleEvent); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Extract relevant information based on event type
	event := &WebhookEvent{
		Type:          mapPaddleEventType(paddleEvent.EventType),
		ProviderEvent: paddleEvent.EventType,
		Raw:           paddleEvent.Data,
	}

	// Different event types have different data structures
	// Handle subscription events
	if strings.HasPrefix(paddleEvent.EventType, "subscription.") {
		// Extract subscription ID
		if subID, ok := paddleEvent.Data["id"].(string); ok {
			event.SubscriptionID = subID
		}

		// Extract status
		if status, ok := paddleEvent.Data["status"].(string); ok {
			event.Status = status
		}

		// Extract customer ID from custom data
		if customData, ok := paddleEvent.Data["custom_data"].(map[string]any); ok {
			if customerID, ok := customData["customer_id"].(string); ok {
				event.CustomerID = customerID
			}
		}

		// Extract plan/price ID from items
		if items, ok := paddleEvent.Data["items"].([]any); ok && len(items) > 0 {
			if item, ok := items[0].(map[string]any); ok {
				if price, ok := item["price"].(map[string]any); ok {
					if priceID, ok := price["id"].(string); ok {
						event.PlanID = priceID
					}
				}
			}
		}
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

		// Extract status
		if status, ok := paddleEvent.Data["status"].(string); ok {
			event.Status = status
		}

		// Extract customer ID from custom data
		if customData, ok := paddleEvent.Data["custom_data"].(map[string]any); ok {
			if customerID, ok := customData["customer_id"].(string); ok {
				event.CustomerID = customerID
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

// ParseWebhookRequest is an alternative method that accepts an http.Request directly.
// This can be used when you have the full HTTP request available.
func (p *PaddleProvider) ParseWebhookRequest(req *http.Request) (*WebhookEvent, error) {
	// Verify the webhook signature
	valid, err := p.verifier.Verify(req)
	if err != nil {
		return nil, fmt.Errorf("webhook verification error: %w", err)
	}
	if !valid {
		return nil, errors.New("webhook signature verification failed")
	}

	// Read the request body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	req.Body = io.NopCloser(bytes.NewReader(body)) // Reset body for potential reuse

	// Get signature from header
	signature := req.Header.Get("Paddle-Signature")

	// Parse using the main method
	return p.ParseWebhook(req.Context(), body, signature)
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

