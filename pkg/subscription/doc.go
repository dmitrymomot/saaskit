// Package subscription provides comprehensive SaaS subscription management with resource limits,
// feature flags, trial periods, and billing provider integration.
//
// The package implements a flexible subscription system that enforces usage limits,
// controls feature access, and integrates with payment providers (Paddle, Stripe,
// Lemonsqueezy) through a minimal interface. It's designed for solo developers building
// SaaS applications who need a pragmatic approach to subscription management.
//
// # Architecture
//
// The package follows a service-oriented architecture with clear separation of concerns:
//
//   - Service: Main interface providing all subscription operations
//   - Plan: Defines subscription tiers with limits and features
//   - BillingProvider: Abstracts payment provider interactions
//   - SubscriptionStore: Persists subscription data
//   - ResourceCounterFunc: Tracks resource usage
//   - PlansListSource: Loads plan definitions
//
// Resource counting is delegated to the application layer through ResourceCounterFunc
// callbacks, allowing flexible implementation strategies (database aggregates, caching,
// external services). Plans are cached in memory after loading for optimal performance.
//
// # Core Components
//
// The Service interface provides all subscription operations:
//   - CanCreate: Check resource limits before creation
//   - GetUsage: Get current usage and limits
//   - HasFeature: Check feature availability
//   - CreateCheckoutLink: Generate payment links
//   - HandleWebhook: Process provider events
//
// Plans define subscription tiers with:
//   - Resource limits (users, projects, storage, etc.)
//   - Feature flags (AI, SSO, API access, etc.)
//   - Trial periods and pricing
//   - Billing intervals (monthly, annual, none for free)
//
// # Quick Start
//
// Create a subscription service with plans, provider, and storage:
//
//	import "github.com/dmitrymomot/saaskit/pkg/subscription"
//
//	// Define subscription plans
//	plans := []subscription.Plan{
//		{
//			ID:       "free",
//			Name:     "Free Tier",
//			Interval: subscription.BillingIntervalNone,
//			Limits: map[subscription.Resource]int64{
//				subscription.ResourceUsers:    1,
//				subscription.ResourceProjects: 3,
//			},
//		},
//		{
//			ID:       "price_pro_monthly", // Must match provider's price ID
//			Name:     "Professional",
//			Interval: subscription.BillingIntervalMonthly,
//			Price:    subscription.Money{Amount: 9900, Currency: "USD"},
//			Limits: map[subscription.Resource]int64{
//				subscription.ResourceUsers:    subscription.Unlimited,
//				subscription.ResourceProjects: subscription.Unlimited,
//			},
//			Features: []subscription.Feature{
//				subscription.FeatureAI,
//				subscription.FeatureSSO,
//			},
//			TrialDays: 14,
//		},
//	}
//
//	// Setup counter functions
//	userCounter := func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
//		// Count users for this tenant from your database
//		return db.CountUsers(ctx, tenantID)
//	}
//
//	projectCounter := func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
//		// Count projects for this tenant
//		return db.CountProjects(ctx, tenantID)
//	}
//
//	// Create service
//	svc, err := subscription.NewService(
//		ctx,
//		subscription.NewInMemSource(plans...),
//		provider, // Your BillingProvider implementation
//		store,    // Your SubscriptionStore implementation
//		subscription.WithCounter(subscription.ResourceUsers, userCounter),
//		subscription.WithCounter(subscription.ResourceProjects, projectCounter),
//	)
//
// # Paddle Integration
//
// The package includes a complete Paddle implementation. Set up Paddle provider:
//
//	import "github.com/dmitrymomot/saaskit/pkg/subscription"
//
//	// Configure Paddle
//	paddleConfig := subscription.PaddleConfig{
//		APIKey:        "your-paddle-api-key",
//		WebhookSecret: "your-paddle-webhook-secret",
//		Environment:   "sandbox", // or "production"
//	}
//
//	// Create Paddle provider
//	provider, err := subscription.NewPaddleProvider(paddleConfig)
//	if err != nil {
//		log.Fatal("Failed to create Paddle provider:", err)
//	}
//
//	// Use in service creation
//	svc, err := subscription.NewService(ctx, planSource, provider, store)
//
// # Resource Management
//
// Enforce resource limits before allowing resource creation:
//
//	// Before creating a user
//	err := svc.CanCreate(ctx, tenantID, subscription.ResourceUsers)
//	if errors.Is(err, subscription.ErrLimitExceeded) {
//		// Show upgrade prompt or reject request
//		return fmt.Errorf("user limit reached, please upgrade your plan")
//	}
//
//	// Create the user...
//	user := createUser(...)
//
//	// Get current usage and limits
//	used, limit, err := svc.GetUsage(ctx, tenantID, subscription.ResourceProjects)
//	if err != nil {
//		// Handle error
//	}
//	fmt.Printf("Using %d of %d projects", used, limit)
//
//	// Get usage as percentage for UI progress bars
//	percentage := svc.GetUsagePercentage(ctx, tenantID, subscription.ResourceStorage)
//	// Returns 0-100 for normal limits, -1 for unlimited
//
// Counter functions must be fast as they're called frequently. Consider:
//   - Database indexes on tenant_id columns
//   - Cached counts with periodic refresh
//   - Eventual consistency for non-critical resources
//
// # Feature Control
//
// Enable/disable features based on subscription plan:
//
//	// Check if AI features are available
//	if svc.HasFeature(ctx, tenantID, subscription.FeatureAI) {
//		// Enable AI-powered features
//		result := processWithAI(input)
//	} else {
//		// Use basic processing
//		result := processBasic(input)
//	}
//
//	// Feature checks are fail-closed for security
//	// Returns false on any error to prevent unauthorized access
//
// Built-in features include:
//   - FeatureAI: AI-powered capabilities
//   - FeatureSSO: Single Sign-On integration
//   - FeatureAPI: API access
//   - FeatureWebhooks: Webhook functionality
//   - FeatureAnalytics: Advanced analytics
//   - And more...
//
// # Plan Management
//
// Set plan context for dynamic plan resolution:
//
//	import "github.com/dmitrymomot/saaskit/pkg/subscription"
//
//	// Set plan ID in request context
//	ctx = subscription.SetPlanIDToContext(ctx, "pro_monthly")
//
//	// Service will use this plan for all operations
//	canCreate := svc.CanCreate(ctx, tenantID, subscription.ResourceUsers)
//
// Alternative: Custom plan resolver from database:
//
//	dbResolver := func(ctx context.Context, tenantID uuid.UUID) (string, error) {
//		return db.GetTenantPlanID(ctx, tenantID)
//	}
//
//	svc, err := subscription.NewService(
//		ctx, planSource, provider, store,
//		subscription.WithPlanIDResolver(dbResolver),
//	)
//
// # Checkout and Billing
//
// Create checkout sessions for plan upgrades:
//
//	// Create checkout link
//	link, err := svc.CreateCheckoutLink(ctx, tenantID, "price_pro_monthly",
//		subscription.CheckoutOptions{
//			Email:      user.Email,        // Pre-fill billing email
//			SuccessURL: "https://app.com/success",
//			CancelURL:  "https://app.com/cancel",
//		},
//	)
//	if err != nil {
//		// Handle error
//	}
//
//	// Redirect user to hosted checkout
//	http.Redirect(w, r, link.URL, http.StatusSeeOther)
//
//	// Get customer portal link for plan management
//	portal, err := svc.GetCustomerPortalLink(ctx, tenantID)
//	if err != nil {
//		// Handle error (free plans return error as they have no portal)
//	}
//	// portal.URL - general portal
//	// portal.CancelURL - direct to cancellation (if available)
//	// portal.UpdatePaymentURL - direct to payment update (if available)
//
// Free plans bypass payment processing and activate immediately.
//
// # Webhook Processing
//
// Process billing provider webhooks to sync subscription state:
//
//	func webhookHandler(w http.ResponseWriter, r *http.Request) {
//		body, err := io.ReadAll(r.Body)
//		if err != nil {
//			http.Error(w, "Invalid request", http.StatusBadRequest)
//			return
//		}
//
//		signature := r.Header.Get("Paddle-Signature") // or appropriate header
//
//		// Process webhook
//		err = svc.HandleWebhook(r.Context(), body, signature)
//		if err != nil {
//			log.Printf("Webhook error: %v", err)
//			http.Error(w, "Webhook processing failed", http.StatusBadRequest)
//			return
//		}
//
//		w.WriteHeader(http.StatusOK)
//	}
//
// Webhook events automatically update subscription status, plan changes,
// and trial states in your SubscriptionStore implementation.
//
// # Trial Management
//
// Plans can include trial periods that are automatically managed:
//
//	// Check if trial is active
//	sub, err := svc.GetSubscription(ctx, tenantID)
//	if err != nil {
//		// Handle error
//	}
//
//	if sub.IsTrialing() {
//		daysLeft := sub.TrialDaysRemaining()
//		fmt.Printf("Trial expires in %d days", daysLeft)
//
//		// Show trial warning when close to expiration
//		if daysLeft <= 3 {
//			showTrialWarning()
//		}
//	}
//
//	// Verify trial status before allowing access
//	err = svc.CheckTrial(ctx, tenantID, sub.CreatedAt)
//	if errors.Is(err, subscription.ErrTrialExpired) {
//		// Block access and show upgrade prompt
//		showUpgradePrompt()
//		return
//	}
//
// # Error Handling
//
// The package defines specific errors for different scenarios:
//
//	switch {
//	case errors.Is(err, subscription.ErrLimitExceeded):
//		// Resource limit reached - show upgrade prompt
//		showUpgradePrompt(resource)
//
//	case errors.Is(err, subscription.ErrTrialExpired):
//		// Trial period ended - require plan selection
//		redirectToPlans()
//
//	case errors.Is(err, subscription.ErrPlanNotFound):
//		// Invalid plan ID - configuration error
//		log.Error("Invalid plan configuration")
//
//	case errors.Is(err, subscription.ErrNoCounterRegistered):
//		// Counter not registered for resource - configuration error
//		log.Error("Resource counter not configured")
//
//	case errors.Is(err, subscription.ErrSubscriptionNotFound):
//		// No subscription exists - redirect to plan selection
//		redirectToPlans()
//	}
//
// # Performance Considerations
//
// For optimal performance:
//   - Resource counters are called frequently - optimize with indexes and caching
//   - Plans are cached in memory - plan changes require service restart
//   - Feature checks are fail-closed - returns false on errors for security
//   - Database queries should use tenant_id indexes
//   - Consider read replicas for counter queries
//   - Cache frequently accessed usage data
//
// # Storage Implementation
//
// Implement SubscriptionStore for your database:
//
//	type MySubscriptionStore struct {
//		db *sql.DB
//	}
//
//	func (s *MySubscriptionStore) Get(ctx context.Context, tenantID uuid.UUID) (*subscription.Subscription, error) {
//		// Query subscription by tenant_id
//		// Return subscription.ErrSubscriptionNotFound if not found
//	}
//
//	func (s *MySubscriptionStore) Save(ctx context.Context, sub *subscription.Subscription) error {
//		// Insert or update subscription using tenant_id as primary key
//	}
//
// The store interface is minimal to support various database systems and ORMs.
package subscription
