package subscription

// Resource represents a countable tenant resource type.
type Resource string

const (
	ResourceUsers        Resource = "users"
	ResourceProjects     Resource = "projects"
	ResourceAPIKeys      Resource = "api_keys"
	ResourceWebhooks     Resource = "webhooks"
	ResourceEmails       Resource = "emails"
	ResourceTickets      Resource = "tickets"
	ResourceTeamMembers  Resource = "team_members"
	ResourceEnvironments Resource = "environments"
	ResourceReferrals    Resource = "referrals"
	ResourceCampaigns    Resource = "campaigns"
	ResourceStorage      Resource = "storage"   // Measured in GB
	ResourceBandwidth    Resource = "bandwidth" // Measured in GB
	ResourceDomains      Resource = "domains"
)

const (
	// Unlimited indicates no limit for a resource (-1 chosen for SQL compatibility)
	Unlimited int64 = -1
)

// Feature represents a plan-specific capability that can be enabled/disabled.
type Feature string

const (
	FeatureAI                Feature = "ai"
	FeatureSSO               Feature = "sso"
	FeatureAPI               Feature = "api"
	FeatureWebhooks          Feature = "webhooks"
	FeatureWhiteLabel        Feature = "white_label"
	FeatureAnalytics         Feature = "analytics"
	FeaturePrioritySupport   Feature = "priority_support"
	FeatureCustomDomain      Feature = "custom_domain"
	FeatureTeamCollaboration Feature = "team_collaboration"
	FeatureExport            Feature = "export"
	FeatureIntegrations      Feature = "integrations"
	FeatureAuditLog          Feature = "audit_log"
)

// UsageInfo contains the current usage and limit for a resource.
type UsageInfo struct {
	Current int64
	Limit   int64
}

// Money represents a monetary amount in the smallest currency unit.
// For example, $10.99 USD would be Amount: 1099, Currency: "USD".
type Money struct {
	Amount   int64  // Amount in smallest currency unit (cents for USD)
	Currency string // ISO 4217 currency code
}

// BillingInterval represents the billing frequency for a subscription plan.
type BillingInterval string

const (
	BillingIntervalNone    BillingInterval = "none" // Free plans with no billing
	BillingIntervalMonthly BillingInterval = "monthly"
	BillingIntervalAnnual  BillingInterval = "annual"
)

// SubscriptionStatus represents the current state of a subscription.
type SubscriptionStatus string

const (
	StatusTrialing  SubscriptionStatus = "trialing"
	StatusActive    SubscriptionStatus = "active"
	StatusPastDue   SubscriptionStatus = "past_due"
	StatusCancelled SubscriptionStatus = "cancelled"
	StatusExpired   SubscriptionStatus = "expired"
)

// CheckoutOptions contains options for creating a checkout session.
type CheckoutOptions struct {
	Email      string // Pre-fill billing email if known
	SuccessURL string // Redirect after successful payment
	CancelURL  string // Redirect if customer cancels
}
