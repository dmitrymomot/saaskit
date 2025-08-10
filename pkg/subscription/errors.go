package subscription

import "errors"

var (
	ErrPlanNotFound             = errors.New("subscription plan not found")
	ErrPlanIDNotFound           = errors.New("subscription plan ID not found")
	ErrPlanIDNotInContext       = errors.New("subscription plan ID not found in context")
	ErrInvalidPlanConfiguration = errors.New("invalid subscription plan configuration")

	ErrLimitExceeded        = errors.New("subscription limit exceeded")
	ErrInvalidResource      = errors.New("invalid subscription resource")
	ErrNoCounterRegistered  = errors.New("no usage counter registered for resource")
	ErrDowngradeNotPossible = errors.New("subscription downgrade not possible")

	ErrTrialExpired      = errors.New("subscription trial has expired")
	ErrTrialNotAvailable = errors.New("subscription trial not available")

	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists")
	ErrInvalidSubscriptionState  = errors.New("invalid subscription state")
	ErrProviderError             = errors.New("subscription provider error")

	ErrFailedToLoadPlans          = errors.New("failed to load subscription plans")
	ErrFailedToCountResourceUsage = errors.New("failed to count resource usage")

	// Provider-specific errors
	ErrMissingAPIKey              = errors.New("billing provider API key is required")
	ErrMissingWebhookSecret       = errors.New("billing provider webhook secret is required")
	ErrInvalidProviderEnvironment = errors.New("invalid billing provider environment")
	ErrWebhookVerificationFailed  = errors.New("webhook signature verification failed")
	ErrNoCheckoutURL              = errors.New("no checkout URL returned from provider")
	ErrNoPortalURL                = errors.New("no portal URL returned from provider")
	ErrMissingProviderCustomerID  = errors.New("provider customer ID not available")
	ErrMissingProviderSubID       = errors.New("subscription provider ID is required for customer portal access")
	ErrMissingTenantID            = errors.New("tenant ID is required")
	ErrMissingPriceID             = errors.New("price ID is required")

	// Webhook processing errors
	ErrMissingTenantIDInWebhook = errors.New("missing tenant ID in webhook event")
	ErrWebhookVerification      = errors.New("webhook verification error")
	ErrFailedToReadRequestBody  = errors.New("failed to read request body")
	ErrFailedToParseWebhook     = errors.New("failed to parse webhook payload")

	// Provider operation errors
	ErrFailedToCreatePaddleClient  = errors.New("failed to create paddle client")
	ErrFailedToCreateTransaction   = errors.New("failed to create paddle transaction")
	ErrFailedToCreatePortalSession = errors.New("failed to create paddle customer portal session")
	ErrNoPortalForFreePlan         = errors.New("no customer portal available for free plans")

	// Subscription operation errors
	ErrFailedToSaveSubscription         = errors.New("failed to save subscription")
	ErrFailedToUpdateSubscription       = errors.New("failed to update subscription")
	ErrFailedToCancelSubscription       = errors.New("failed to cancel subscription")
	ErrFailedToUpdateSubscriptionStatus = errors.New("failed to update subscription status")

	// Configuration errors
	ErrPlanIDMismatch    = errors.New("plan ID mismatch in configuration")
	ErrNegativeTrialDays = errors.New("plan has negative trial days")
)
