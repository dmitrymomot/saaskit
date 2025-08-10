package subscription_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/subscription"
)

// Mock implementations
type mockPlansSource struct {
	mock.Mock
}

func (m *mockPlansSource) Load(ctx context.Context) (map[string]subscription.Plan, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]subscription.Plan), args.Error(1)
}

type mockProvider struct {
	mock.Mock
}

func (m *mockProvider) CreateCheckoutLink(ctx context.Context, req subscription.CheckoutRequest) (*subscription.CheckoutLink, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*subscription.CheckoutLink), args.Error(1)
}

func (m *mockProvider) GetCustomerPortalLink(ctx context.Context, sub *subscription.Subscription) (*subscription.PortalLink, error) {
	args := m.Called(ctx, sub)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*subscription.PortalLink), args.Error(1)
}

func (m *mockProvider) ParseWebhook(r *http.Request) (*subscription.WebhookEvent, error) {
	args := m.Called(r)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*subscription.WebhookEvent), args.Error(1)
}

type mockStore struct {
	mock.Mock
}

func (m *mockStore) Get(ctx context.Context, tenantID uuid.UUID) (*subscription.Subscription, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*subscription.Subscription), args.Error(1)
}

func (m *mockStore) Save(ctx context.Context, sub *subscription.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

// Test helpers
func createTestPlans() map[string]subscription.Plan {
	return map[string]subscription.Plan{
		"free": {
			ID:       "free",
			Name:     "Free",
			Interval: subscription.BillingIntervalNone,
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects:    1,
				subscription.ResourceTeamMembers: 1,
				subscription.ResourceAPIKeys:     0,
			},
			Features: []subscription.Feature{},
		},
		"basic": {
			ID:       "basic",
			Name:     "Basic",
			Interval: subscription.BillingIntervalMonthly,
			Price:    subscription.Money{Amount: 1000, Currency: "USD"},
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects:    10,
				subscription.ResourceTeamMembers: 5,
				subscription.ResourceAPIKeys:     2,
			},
			Features: []subscription.Feature{
				subscription.FeatureAPI,
			},
		},
		"pro": {
			ID:        "pro",
			Name:      "Pro",
			Interval:  subscription.BillingIntervalMonthly,
			Price:     subscription.Money{Amount: 5000, Currency: "USD"},
			TrialDays: 14,
			Limits: map[subscription.Resource]int64{
				subscription.ResourceProjects:    50,
				subscription.ResourceTeamMembers: subscription.Unlimited,
				subscription.ResourceAPIKeys:     10,
				subscription.ResourceWebhooks:    5,
			},
			Features: []subscription.Feature{
				subscription.FeatureAPI,
				subscription.FeatureSSO,
				subscription.FeatureWebhooks,
			},
		},
	}
}

func TestService_CanCreate_LimitValidation(t *testing.T) {
	t.Parallel()

	t.Run("allows creation when under limit", func(t *testing.T) {
		t.Parallel()
		ctx := subscription.SetPlanIDToContext(context.Background(), "basic")
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		svc, err := subscription.NewService(ctx, src, provider, store,
			subscription.WithCounter(subscription.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
				return 5, nil // Current usage is 5, limit is 10
			}),
		)
		require.NoError(t, err)

		err = svc.CanCreate(ctx, tenantID, subscription.ResourceProjects)
		assert.NoError(t, err)

		src.AssertExpectations(t)
	})

	t.Run("blocks creation when at limit", func(t *testing.T) {
		t.Parallel()
		ctx := subscription.SetPlanIDToContext(context.Background(), "basic")
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		svc, err := subscription.NewService(ctx, src, provider, store,
			subscription.WithCounter(subscription.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
				return 10, nil // At limit
			}),
		)
		require.NoError(t, err)

		err = svc.CanCreate(ctx, tenantID, subscription.ResourceProjects)
		assert.ErrorIs(t, err, subscription.ErrLimitExceeded)

		src.AssertExpectations(t)
	})

	t.Run("allows unlimited resources", func(t *testing.T) {
		t.Parallel()
		ctx := subscription.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		svc, err := subscription.NewService(ctx, src, provider, store,
			subscription.WithCounter(subscription.ResourceTeamMembers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
				return 100, nil // High usage but unlimited
			}),
		)
		require.NoError(t, err)

		err = svc.CanCreate(ctx, tenantID, subscription.ResourceTeamMembers)
		assert.NoError(t, err)

		src.AssertExpectations(t)
	})

	t.Run("returns error when no counter registered", func(t *testing.T) {
		t.Parallel()
		ctx := subscription.SetPlanIDToContext(context.Background(), "basic")
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		svc, err := subscription.NewService(ctx, src, provider, store)
		require.NoError(t, err)

		err = svc.CanCreate(ctx, tenantID, subscription.ResourceProjects)
		assert.ErrorIs(t, err, subscription.ErrNoCounterRegistered)

		src.AssertExpectations(t)
	})
}

func TestService_GetUsagePercentage(t *testing.T) {
	t.Parallel()

	t.Run("calculates percentage correctly", func(t *testing.T) {
		t.Parallel()
		ctx := subscription.SetPlanIDToContext(context.Background(), "basic")
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		svc, err := subscription.NewService(ctx, src, provider, store,
			subscription.WithCounter(subscription.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
				return 7, nil // 7 out of 10 = 70%
			}),
		)
		require.NoError(t, err)

		percentage := svc.GetUsagePercentage(ctx, tenantID, subscription.ResourceProjects)
		assert.Equal(t, 70, percentage)

		src.AssertExpectations(t)
	})

	t.Run("returns -1 for unlimited resources", func(t *testing.T) {
		t.Parallel()
		ctx := subscription.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		svc, err := subscription.NewService(ctx, src, provider, store,
			subscription.WithCounter(subscription.ResourceTeamMembers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
				return 100, nil
			}),
		)
		require.NoError(t, err)

		percentage := svc.GetUsagePercentage(ctx, tenantID, subscription.ResourceTeamMembers)
		assert.Equal(t, -1, percentage)

		src.AssertExpectations(t)
	})

	t.Run("returns 100 when limit is zero", func(t *testing.T) {
		t.Parallel()
		ctx := subscription.SetPlanIDToContext(context.Background(), "free")
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		svc, err := subscription.NewService(ctx, src, provider, store,
			subscription.WithCounter(subscription.ResourceAPIKeys, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
				return 0, nil
			}),
		)
		require.NoError(t, err)

		percentage := svc.GetUsagePercentage(ctx, tenantID, subscription.ResourceAPIKeys)
		assert.Equal(t, 100, percentage)

		src.AssertExpectations(t)
	})

	t.Run("caps at 100 percent", func(t *testing.T) {
		t.Parallel()
		ctx := subscription.SetPlanIDToContext(context.Background(), "basic")
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		svc, err := subscription.NewService(ctx, src, provider, store,
			subscription.WithCounter(subscription.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
				return 15, nil // Over limit
			}),
		)
		require.NoError(t, err)

		percentage := svc.GetUsagePercentage(ctx, tenantID, subscription.ResourceProjects)
		assert.Equal(t, 100, percentage)

		src.AssertExpectations(t)
	})
}

func TestService_CanDowngrade(t *testing.T) {
	t.Parallel()

	t.Run("allows downgrade when usage within target limits", func(t *testing.T) {
		t.Parallel()
		ctx := subscription.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		svc, err := subscription.NewService(ctx, src, provider, store,
			subscription.WithCounter(subscription.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
				return 5, nil // Within basic limit of 10
			}),
		)
		require.NoError(t, err)

		err = svc.CanDowngrade(ctx, tenantID, "basic")
		assert.NoError(t, err)

		src.AssertExpectations(t)
	})

	t.Run("blocks downgrade when over target limits", func(t *testing.T) {
		t.Parallel()
		ctx := subscription.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		svc, err := subscription.NewService(ctx, src, provider, store,
			subscription.WithCounter(subscription.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
				return 25, nil // Over basic limit of 10
			}),
		)
		require.NoError(t, err)

		err = svc.CanDowngrade(ctx, tenantID, "basic")
		assert.ErrorIs(t, err, subscription.ErrDowngradeNotPossible)

		src.AssertExpectations(t)
	})

	t.Run("blocks downgrade from unlimited to limited", func(t *testing.T) {
		t.Parallel()
		ctx := subscription.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		svc, err := subscription.NewService(ctx, src, provider, store,
			subscription.WithCounter(subscription.ResourceTeamMembers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
				return 10, nil // Over basic limit of 5
			}),
		)
		require.NoError(t, err)

		err = svc.CanDowngrade(ctx, tenantID, "basic")
		assert.ErrorIs(t, err, subscription.ErrDowngradeNotPossible)

		src.AssertExpectations(t)
	})
}

func TestService_CheckTrial(t *testing.T) {
	t.Parallel()

	t.Run("trial active when within period", func(t *testing.T) {
		t.Parallel()
		ctx := subscription.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		svc, err := subscription.NewService(ctx, src, provider, store)
		require.NoError(t, err)

		startedAt := time.Now().AddDate(0, 0, -7) // Started 7 days ago
		err = svc.CheckTrial(ctx, tenantID, startedAt)
		assert.NoError(t, err)

		src.AssertExpectations(t)
	})

	t.Run("trial expired when past period", func(t *testing.T) {
		t.Parallel()
		ctx := subscription.SetPlanIDToContext(context.Background(), "pro")
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		svc, err := subscription.NewService(ctx, src, provider, store)
		require.NoError(t, err)

		startedAt := time.Now().AddDate(0, 0, -20) // Started 20 days ago
		err = svc.CheckTrial(ctx, tenantID, startedAt)
		assert.ErrorIs(t, err, subscription.ErrTrialExpired)

		src.AssertExpectations(t)
	})

	t.Run("no trial for plans without trial days", func(t *testing.T) {
		t.Parallel()
		ctx := subscription.SetPlanIDToContext(context.Background(), "basic")
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		svc, err := subscription.NewService(ctx, src, provider, store)
		require.NoError(t, err)

		err = svc.CheckTrial(ctx, tenantID, time.Now())
		assert.ErrorIs(t, err, subscription.ErrTrialNotAvailable)

		src.AssertExpectations(t)
	})
}

func TestService_CreateCheckoutLink_FreePlan(t *testing.T) {
	t.Parallel()

	t.Run("free plan bypasses provider and creates subscription directly", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		// Free plan should not exist yet
		store.On("Get", ctx, tenantID).Return(nil, subscription.ErrSubscriptionNotFound)

		// Should save the free subscription
		store.On("Save", ctx, mock.MatchedBy(func(sub *subscription.Subscription) bool {
			return sub.TenantID == tenantID &&
				sub.PlanID == "free" &&
				sub.Status == subscription.StatusActive &&
				sub.ProviderSubID == ""
		})).Return(nil)

		svc, err := subscription.NewService(ctx, src, provider, store)
		require.NoError(t, err)

		opts := subscription.CheckoutOptions{
			SuccessURL: "https://example.com/success",
			CancelURL:  "https://example.com/cancel",
		}

		link, err := svc.CreateCheckoutLink(ctx, tenantID, "free", opts)
		require.NoError(t, err)
		assert.Equal(t, opts.SuccessURL, link.URL)
		assert.Empty(t, link.SessionID)

		src.AssertExpectations(t)
		store.AssertExpectations(t)
		provider.AssertExpectations(t)
	})
}

func TestService_CreateCheckoutLink_PaidPlan(t *testing.T) {
	t.Parallel()

	t.Run("paid plan delegates to provider", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		store.On("Get", ctx, tenantID).Return(nil, subscription.ErrSubscriptionNotFound)

		checkoutReq := subscription.CheckoutRequest{
			PriceID:    "basic",
			TenantID:   tenantID,
			Email:      "user@example.com",
			SuccessURL: "https://example.com/success",
			CancelURL:  "https://example.com/cancel",
		}

		expectedLink := &subscription.CheckoutLink{
			URL:       "https://provider.com/checkout/123",
			SessionID: "cs_123",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		provider.On("CreateCheckoutLink", ctx, checkoutReq).Return(expectedLink, nil)

		svc, err := subscription.NewService(ctx, src, provider, store)
		require.NoError(t, err)

		opts := subscription.CheckoutOptions{
			Email:      checkoutReq.Email,
			SuccessURL: checkoutReq.SuccessURL,
			CancelURL:  checkoutReq.CancelURL,
		}

		link, err := svc.CreateCheckoutLink(ctx, tenantID, "basic", opts)
		require.NoError(t, err)
		assert.Equal(t, expectedLink.URL, link.URL)
		assert.Equal(t, expectedLink.SessionID, link.SessionID)

		src.AssertExpectations(t)
		store.AssertExpectations(t)
		provider.AssertExpectations(t)
	})
}

func TestService_CreateCheckoutLink_DuplicatePrevention(t *testing.T) {
	t.Parallel()

	t.Run("prevents duplicate subscriptions", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		// Already has a subscription
		existingSub := &subscription.Subscription{
			TenantID: tenantID,
			PlanID:   "basic",
			Status:   subscription.StatusActive,
		}
		store.On("Get", ctx, tenantID).Return(existingSub, nil)

		svc, err := subscription.NewService(ctx, src, provider, store)
		require.NoError(t, err)

		opts := subscription.CheckoutOptions{
			SuccessURL: "https://example.com/success",
			CancelURL:  "https://example.com/cancel",
		}

		_, err = svc.CreateCheckoutLink(ctx, tenantID, "pro", opts)
		assert.ErrorIs(t, err, subscription.ErrSubscriptionAlreadyExists)

		src.AssertExpectations(t)
		store.AssertExpectations(t)
		provider.AssertExpectations(t)
	})
}

func TestService_HandleWebhook_SubscriptionCreated(t *testing.T) {
	t.Parallel()

	t.Run("creates subscription with trial", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		payload := []byte(`{"event": "subscription.created"}`)
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
		req.Header.Set("Paddle-Signature", "valid_signature")

		event := &subscription.WebhookEvent{
			Type:           subscription.EventSubscriptionCreated,
			TenantID:       tenantID,
			CustomerID:     "cus_test123",
			SubscriptionID: "sub_123",
			PlanID:         "pro",
			Status:         string(subscription.StatusTrialing),
		}

		provider.On("ParseWebhook", req).Return(event, nil)

		store.On("Save", ctx, mock.MatchedBy(func(sub *subscription.Subscription) bool {
			return sub.TenantID == tenantID &&
				sub.PlanID == "pro" &&
				sub.Status == subscription.StatusTrialing &&
				sub.ProviderSubID == "sub_123" &&
				sub.TrialEndsAt != nil
		})).Return(nil)

		svc, err := subscription.NewService(ctx, src, provider, store)
		require.NoError(t, err)

		err = svc.HandleWebhook(req)
		assert.NoError(t, err)

		src.AssertExpectations(t)
		store.AssertExpectations(t)
		provider.AssertExpectations(t)
	})
}

func TestService_HandleWebhook_SubscriptionUpdated(t *testing.T) {
	t.Parallel()

	t.Run("updates existing subscription status", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		payload := []byte(`{"event": "subscription.updated"}`)
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
		req.Header.Set("Paddle-Signature", "valid_signature")

		event := &subscription.WebhookEvent{
			Type:           subscription.EventSubscriptionUpdated,
			TenantID:       tenantID,
			CustomerID:     "cus_test123",
			SubscriptionID: "sub_123",
			PlanID:         "pro",
			Status:         string(subscription.StatusActive),
		}

		existingSub := &subscription.Subscription{
			TenantID:      tenantID,
			PlanID:        "basic",
			Status:        subscription.StatusTrialing,
			ProviderSubID: "sub_123",
		}

		provider.On("ParseWebhook", req).Return(event, nil)
		store.On("Get", ctx, tenantID).Return(existingSub, nil)

		store.On("Save", ctx, mock.MatchedBy(func(sub *subscription.Subscription) bool {
			return sub.TenantID == tenantID &&
				sub.PlanID == "pro" &&
				sub.Status == subscription.StatusActive &&
				sub.ProviderSubID == "sub_123"
		})).Return(nil)

		svc, err := subscription.NewService(ctx, src, provider, store)
		require.NoError(t, err)

		err = svc.HandleWebhook(req)
		assert.NoError(t, err)

		src.AssertExpectations(t)
		store.AssertExpectations(t)
		provider.AssertExpectations(t)
	})
}

func TestService_HandleWebhook_PaymentFailed(t *testing.T) {
	t.Parallel()

	t.Run("sets subscription to past due on payment failure", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		tenantID := uuid.New()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		payload := []byte(`{"event": "payment.failed"}`)
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
		req.Header.Set("Paddle-Signature", "valid_signature")

		event := &subscription.WebhookEvent{
			Type:     subscription.EventPaymentFailed,
			TenantID: tenantID,
		}

		existingSub := &subscription.Subscription{
			TenantID:      tenantID,
			PlanID:        "pro",
			Status:        subscription.StatusActive,
			ProviderSubID: "sub_123",
		}

		provider.On("ParseWebhook", req).Return(event, nil)
		store.On("Get", ctx, tenantID).Return(existingSub, nil)

		store.On("Save", ctx, mock.MatchedBy(func(sub *subscription.Subscription) bool {
			return sub.Status == subscription.StatusPastDue
		})).Return(nil)

		svc, err := subscription.NewService(ctx, src, provider, store)
		require.NoError(t, err)

		err = svc.HandleWebhook(req)
		assert.NoError(t, err)

		src.AssertExpectations(t)
		store.AssertExpectations(t)
		provider.AssertExpectations(t)
	})
}

func TestService_HandleWebhook_InvalidSignature(t *testing.T) {
	t.Parallel()

	t.Run("rejects webhook with invalid signature", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		src := &mockPlansSource{}
		provider := &mockProvider{}
		store := &mockStore{}

		plans := createTestPlans()
		src.On("Load", mock.Anything).Return(plans, nil)

		payload := []byte(`{"event": "subscription.created"}`)
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
		req.Header.Set("Paddle-Signature", "invalid_signature")

		provider.On("ParseWebhook", req).Return(nil, errors.New("invalid signature"))

		svc, err := subscription.NewService(ctx, src, provider, store)
		require.NoError(t, err)

		err = svc.HandleWebhook(req)
		assert.Error(t, err)

		src.AssertExpectations(t)
		store.AssertExpectations(t)
		provider.AssertExpectations(t)
	})
}
