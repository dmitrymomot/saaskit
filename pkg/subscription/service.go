package subscription

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

// Service defines the public interface for subscription management.
type Service interface {
	// Limits and features
	CanCreate(ctx context.Context, tenantID uuid.UUID, res Resource) error
	GetUsage(ctx context.Context, tenantID uuid.UUID, res Resource) (used, limit int64, err error)
	GetUsageSafe(ctx context.Context, tenantID uuid.UUID, res Resource) (used, limit int64)
	HasFeature(ctx context.Context, tenantID uuid.UUID, feature Feature) bool
	CheckTrial(ctx context.Context, tenantID uuid.UUID, startedAt time.Time) error
	VerifyPlan(ctx context.Context, planID string) error
	GetUsagePercentage(ctx context.Context, tenantID uuid.UUID, res Resource) int
	CanDowngrade(ctx context.Context, tenantID uuid.UUID, targetPlanID string) error
	GetAllUsage(ctx context.Context, tenantID uuid.UUID) (map[Resource]UsageInfo, error)

	// Subscription management
	GetSubscription(ctx context.Context, tenantID uuid.UUID) (*Subscription, error)

	// Billing provider interactions
	CreateCheckoutLink(ctx context.Context, tenantID uuid.UUID, planID string, opts CheckoutOptions) (*CheckoutLink, error)
	GetCustomerPortalLink(ctx context.Context, tenantID uuid.UUID) (*PortalLink, error)
	HandleWebhook(ctx context.Context, payload []byte, signature string) error
}

// PlansListSource defines how plans are loaded into the subscription service.
type PlansListSource interface {
	Load(ctx context.Context) (map[string]Plan, error)
}

// PlanIDResolver resolves a plan ID for a given tenant.
type PlanIDResolver func(ctx context.Context, tenantID uuid.UUID) (string, error)

// ResourceCounterFunc returns the current usage for a tenant resource.
// Must be fast and ideally cached as it's called on every resource creation attempt.
// Consider implementing counters with database aggregates or cached values.
type ResourceCounterFunc func(ctx context.Context, tenantID uuid.UUID) (int64, error)

type service struct {
	plans          map[string]Plan
	counters       map[Resource]ResourceCounterFunc
	planIDResolver PlanIDResolver
	provider       BillingProvider
	store          SubscriptionStore
}

// NewService creates a new Service with the given dependencies.
// Panics if required parameters (src, provider, store) are nil to fail fast during initialization.
// This prevents runtime errors from misconfigured services.
// Use ServiceOption functions to configure optional settings like custom plan ID resolver.
func NewService(ctx context.Context, src PlansListSource, provider BillingProvider, store SubscriptionStore, opts ...ServiceOption) (Service, error) {
	if src == nil {
		panic("subscription: PlansListSource is required")
	}
	if provider == nil {
		panic("subscription: BillingProvider is required")
	}
	if store == nil {
		panic("subscription: SubscriptionStore is required")
	}

	plans, err := src.Load(ctx)
	if err != nil {
		return nil, errors.Join(ErrFailedToLoadPlans, err)
	}

	if err := validatePlans(plans); err != nil {
		return nil, err
	}

	s := &service{
		plans:          plans,
		counters:       make(map[Resource]ResourceCounterFunc),
		planIDResolver: PlanIDContextResolver,
		provider:       provider,
		store:          store,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// CanCreate checks if a tenant can create a new resource instance.
func (s *service) CanCreate(ctx context.Context, tenantID uuid.UUID, res Resource) error {
	planID, err := s.planIDResolver(ctx, tenantID)
	if err != nil {
		return err
	}

	plan, exists := s.plans[planID]
	if !exists {
		return ErrPlanNotFound
	}

	limit, exists := plan.Limits[res]
	if !exists {
		return ErrInvalidResource
	}

	if limit == Unlimited {
		return nil
	}

	counter, exists := s.counters[res]
	if !exists {
		return ErrNoCounterRegistered
	}

	current, err := counter(ctx, tenantID)
	if err != nil {
		return errors.Join(ErrFailedToCountResourceUsage, err)
	}

	if current >= limit {
		return ErrLimitExceeded
	}

	return nil
}

// GetUsage returns the current usage and limit for a resource in a tenant.
func (s *service) GetUsage(ctx context.Context, tenantID uuid.UUID, res Resource) (used, limit int64, err error) {
	planID, err := s.planIDResolver(ctx, tenantID)
	if err != nil {
		return 0, 0, err
	}

	plan, exists := s.plans[planID]
	if !exists {
		return 0, 0, ErrPlanNotFound
	}

	resourceLimit, exists := plan.Limits[res]
	if !exists {
		return 0, 0, ErrInvalidResource
	}

	counter, exists := s.counters[res]
	if !exists {
		return 0, 0, ErrNoCounterRegistered
	}

	current, err := counter(ctx, tenantID)
	if err != nil {
		return 0, 0, errors.Join(ErrFailedToCountResourceUsage, err)
	}

	return current, resourceLimit, nil
}

// GetUsageSafe is a convenience wrapper for UI dashboards that need to display usage
// without error handling. Returns zero values on any error to prevent UI crashes.
func (s *service) GetUsageSafe(ctx context.Context, tenantID uuid.UUID, res Resource) (used, limit int64) {
	used, limit, _ = s.GetUsage(ctx, tenantID, res)
	return used, limit
}

// HasFeature checks if a feature is available for the tenant's current plan.
// Returns false on any error to fail closed for security-sensitive features.
func (s *service) HasFeature(ctx context.Context, tenantID uuid.UUID, feature Feature) bool {
	planID, err := s.planIDResolver(ctx, tenantID)
	if err != nil {
		return false
	}

	plan, exists := s.plans[planID]
	if !exists {
		return false
	}

	return slices.Contains(plan.Features, feature)
}

// CheckTrial determines if a tenant's trial period is active for a specific plan.
func (s *service) CheckTrial(ctx context.Context, tenantID uuid.UUID, startedAt time.Time) error {
	planID, err := s.planIDResolver(ctx, tenantID)
	if err != nil {
		return err
	}

	plan, exists := s.plans[planID]
	if !exists {
		return ErrPlanNotFound
	}

	if plan.TrialDays == 0 {
		return ErrTrialNotAvailable
	}

	if !plan.IsTrialActive(startedAt) {
		return ErrTrialExpired
	}

	return nil
}

// VerifyPlan checks if a plan ID is valid.
func (s *service) VerifyPlan(ctx context.Context, planID string) error {
	if _, exists := s.plans[planID]; !exists {
		return ErrPlanNotFound
	}
	return nil
}

// GetUsagePercentage returns usage as percentage (0-100, or -1 for unlimited).
// Caps at 100% to prevent UI display issues. Returns 0 on errors.
func (s *service) GetUsagePercentage(ctx context.Context, tenantID uuid.UUID, res Resource) int {
	used, limit, err := s.GetUsage(ctx, tenantID, res)
	if err != nil {
		return 0
	}

	if limit == Unlimited {
		return -1
	}

	if limit == 0 {
		return 100
	}

	return min(int((used*100)/limit), 100)
}

// CanDowngrade checks if downgrade is possible given current usage.
func (s *service) CanDowngrade(ctx context.Context, tenantID uuid.UUID, targetPlanID string) error {
	targetPlan, exists := s.plans[targetPlanID]
	if !exists {
		return ErrPlanNotFound
	}

	currentPlanID, err := s.planIDResolver(ctx, tenantID)
	if err != nil {
		return err
	}

	currentPlan, exists := s.plans[currentPlanID]
	if !exists {
		return ErrPlanNotFound
	}

	for resource, targetLimit := range targetPlan.Limits {
		if targetLimit == Unlimited {
			continue
		}

		currentLimit, hasResource := currentPlan.Limits[resource]
		if !hasResource {
			continue
		}

		// Only verify current usage when limit is being reduced
		// to prevent data loss scenarios
		if currentLimit != targetLimit && (currentLimit == Unlimited || currentLimit > targetLimit) {
			counter, exists := s.counters[resource]
			if !exists {
				continue
			}

			currentUsage, err := counter(ctx, tenantID)
			if err != nil {
				return errors.Join(ErrFailedToCountResourceUsage, err)
			}

			if currentUsage > targetLimit {
				return ErrDowngradeNotPossible
			}
		}
	}

	return nil
}

// GetAllUsage returns all resource usage for a tenant.
func (s *service) GetAllUsage(ctx context.Context, tenantID uuid.UUID) (map[Resource]UsageInfo, error) {
	planID, err := s.planIDResolver(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	plan, exists := s.plans[planID]
	if !exists {
		return nil, ErrPlanNotFound
	}

	result := make(map[Resource]UsageInfo, len(plan.Limits))

	for resource, limit := range plan.Limits {
		usage := UsageInfo{
			Current: 0,
			Limit:   limit,
		}

		if counter, exists := s.counters[resource]; exists {
			if current, err := counter(ctx, tenantID); err == nil {
				usage.Current = current
			}
		}

		result[resource] = usage
	}

	return result, nil
}

// CreateCheckoutLink generates a checkout link for a tenant to subscribe to a plan.
func (s *service) CreateCheckoutLink(ctx context.Context, tenantID uuid.UUID, planID string, opts CheckoutOptions) (*CheckoutLink, error) {
	plan, exists := s.plans[planID]
	if !exists {
		return nil, ErrPlanNotFound
	}

	// Prevent duplicate subscriptions for the same tenant
	if _, err := s.store.Get(ctx, tenantID); err == nil {
		return nil, ErrSubscriptionAlreadyExists
	} else if !errors.Is(err, ErrSubscriptionNotFound) {
		return nil, err
	}

	// Free plans bypass payment provider entirely for instant activation
	if plan.Interval == BillingIntervalNone {
		now := time.Now().UTC()
		subscription := &Subscription{
			TenantID:      tenantID,
			PlanID:        planID,
			Status:        StatusActive,
			ProviderSubID: "",
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if err := s.store.Save(ctx, subscription); err != nil {
			return nil, fmt.Errorf("failed to save free plan subscription: %w", err)
		}

		// Redirect to success URL immediately since no payment needed
		return &CheckoutLink{
			URL:       opts.SuccessURL,
			SessionID: "",
			ExpiresAt: time.Now().Add(5 * time.Minute),
		}, nil
	}

	// Delegate to payment provider for paid plans
	return s.provider.CreateCheckoutLink(ctx, CheckoutRequest{
		PriceID:    plan.ID, // Plan.ID must match provider's price ID
		CustomerID: tenantID.String(),
		Email:      opts.Email,
		SuccessURL: opts.SuccessURL,
		CancelURL:  opts.CancelURL,
	})
}

// GetSubscription retrieves a tenant's subscription.
func (s *service) GetSubscription(ctx context.Context, tenantID uuid.UUID) (*Subscription, error) {
	return s.store.Get(ctx, tenantID)
}

// GetCustomerPortalLink returns a link to the customer portal where users can manage their subscription.
func (s *service) GetCustomerPortalLink(ctx context.Context, tenantID uuid.UUID) (*PortalLink, error) {
	subscription, err := s.store.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Free plans have no provider subscription to manage
	if subscription.ProviderSubID == "" {
		return nil, fmt.Errorf("no customer portal available for free plans")
	}

	// Provider implementation determines which subscription fields to use
	// (e.g., Paddle uses TenantID, Stripe uses ProviderSubID)
	return s.provider.GetCustomerPortalLink(ctx, subscription)
}

// HandleWebhook processes incoming webhook events from the billing provider.
func (s *service) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	event, err := s.provider.ParseWebhook(ctx, payload, signature)
	if err != nil {
		return err
	}

	// Customer ID must be a valid UUID matching our tenant ID format
	tenantID, err := uuid.Parse(event.CustomerID)
	if err != nil {
		return fmt.Errorf("invalid tenant ID in webhook: %w", err)
	}

	switch event.Type {
	case EventSubscriptionCreated:
		now := time.Now().UTC()
		subscription := &Subscription{
			TenantID:      tenantID,
			PlanID:        event.PlanID,
			Status:        SubscriptionStatus(event.Status),
			ProviderSubID: event.SubscriptionID,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		// Set trial end date based on plan configuration
		// Provider should already set status to "trialing" if applicable
		if plan, exists := s.plans[event.PlanID]; exists && plan.TrialDays > 0 {
			if subscription.Status == StatusTrialing {
				trialEnd := plan.TrialEndsAt(now)
				subscription.TrialEndsAt = &trialEnd
			}
		}

		if err := s.store.Save(ctx, subscription); err != nil {
			return fmt.Errorf("failed to save subscription: %w", err)
		}

	case EventSubscriptionUpdated:
		subscription, err := s.store.Get(ctx, tenantID)
		if err != nil {
			return fmt.Errorf("subscription not found for tenant %s: %w", tenantID, err)
		}

		subscription.PlanID = event.PlanID
		subscription.Status = SubscriptionStatus(event.Status)
		subscription.UpdatedAt = time.Now().UTC()

		if err := s.store.Save(ctx, subscription); err != nil {
			return fmt.Errorf("failed to update subscription: %w", err)
		}

	case EventSubscriptionCancelled:
		subscription, err := s.store.Get(ctx, tenantID)
		if err != nil {
			return fmt.Errorf("subscription not found for tenant %s: %w", tenantID, err)
		}

		now := time.Now().UTC()
		subscription.Status = StatusCancelled
		subscription.CancelledAt = &now
		subscription.UpdatedAt = now

		if err := s.store.Save(ctx, subscription); err != nil {
			return fmt.Errorf("failed to cancel subscription: %w", err)
		}

	case EventPaymentFailed:
		subscription, err := s.store.Get(ctx, tenantID)
		if err == nil {
			subscription.Status = StatusPastDue
			subscription.UpdatedAt = time.Now().UTC()

			if err := s.store.Save(ctx, subscription); err != nil {
				return fmt.Errorf("failed to update subscription status: %w", err)
			}
		} else if !errors.Is(err, ErrSubscriptionNotFound) {
			return fmt.Errorf("failed to get subscription: %w", err)
		}
	}

	return nil
}

// validatePlans ensures plan configurations are internally consistent.
// Catches common configuration errors early to prevent runtime issues.
func validatePlans(plans map[string]Plan) error {
	for planID, plan := range plans {
		if plan.ID != planID {
			return errors.Join(ErrInvalidPlanConfiguration,
				fmt.Errorf("plan ID mismatch: map key %s != plan.ID %s", planID, plan.ID))
		}

		if plan.TrialDays < 0 {
			return errors.Join(ErrInvalidPlanConfiguration,
				fmt.Errorf("plan %s has negative trial days: %d", planID, plan.TrialDays))
		}
	}
	return nil
}
