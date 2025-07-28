package tenant_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/tenant"
)

// mockProvider implements tenant.Provider for testing
type mockProvider struct {
	mu      sync.RWMutex
	tenants map[string]*tenant.Tenant
	err     error
	calls   int
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		tenants: make(map[string]*tenant.Tenant),
	}
}

func (m *mockProvider) GetByIdentifier(ctx context.Context, identifier string) (*tenant.Tenant, error) {
	m.mu.Lock()
	m.calls++
	err := m.err
	m.mu.Unlock()

	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	t, ok := m.tenants[identifier]
	m.mu.RUnlock()

	if !ok {
		return nil, tenant.ErrTenantNotFound
	}
	return t, nil
}

func (m *mockProvider) addTenant(t *tenant.Tenant) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tenants[t.ID.String()] = t
	m.tenants[t.Subdomain] = t
}

func (m *mockProvider) getCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.calls
}

func (m *mockProvider) getCallCount() int {
	return m.getCalls()
}

func TestTenant_Structure(t *testing.T) {
	t.Parallel()

	t.Run("tenant has all required fields", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		tenantID := uuid.New()

		testTenant := &tenant.Tenant{
			ID:        tenantID,
			Subdomain: "acme",
			Name:      "ACME Corp",
			Logo:      "https://example.com/logo.png",
			PlanID:    "pro",
			Active:    true,
			CreatedAt: now,
		}

		assert.Equal(t, tenantID, testTenant.ID)
		assert.Equal(t, "acme", testTenant.Subdomain)
		assert.Equal(t, "ACME Corp", testTenant.Name)
		assert.Equal(t, "https://example.com/logo.png", testTenant.Logo)
		assert.Equal(t, "pro", testTenant.PlanID)
		assert.True(t, testTenant.Active)
		assert.Equal(t, now, testTenant.CreatedAt)
	})
}

func TestProvider_MockImplementation(t *testing.T) {
	t.Parallel()

	t.Run("returns tenant by UUID", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		testTenant := createTestTenant("acme", true)
		provider.addTenant(testTenant)

		result, err := provider.GetByIdentifier(context.Background(), testTenant.ID.String())
		require.NoError(t, err)
		assert.Equal(t, testTenant, result)
	})

	t.Run("returns tenant by subdomain", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		testTenant := createTestTenant("acme", true)
		provider.addTenant(testTenant)

		result, err := provider.GetByIdentifier(context.Background(), "acme")
		require.NoError(t, err)
		assert.Equal(t, testTenant, result)
	})

	t.Run("returns ErrTenantNotFound for missing tenant", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()

		_, err := provider.GetByIdentifier(context.Background(), "nonexistent")
		assert.ErrorIs(t, err, tenant.ErrTenantNotFound)
	})

	t.Run("returns custom error when set", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		customErr := errors.New("database error")
		provider.err = customErr

		_, err := provider.GetByIdentifier(context.Background(), "any")
		assert.ErrorIs(t, err, customErr)
	})

	t.Run("tracks call count", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()

		_, _ = provider.GetByIdentifier(context.Background(), "test")
		_, _ = provider.GetByIdentifier(context.Background(), "test")

		assert.Equal(t, 2, provider.getCalls())
	})
}

// Helper function to create test tenants
func createTestTenant(subdomain string, active bool) *tenant.Tenant {
	return &tenant.Tenant{
		ID:        uuid.New(),
		Subdomain: subdomain,
		Name:      subdomain + " Corp",
		Logo:      "https://example.com/" + subdomain + ".png",
		PlanID:    "standard",
		Active:    active,
		CreatedAt: time.Now(),
	}
}

// Helper function to create test tenant with specific ID
func createTestTenantWithID(id uuid.UUID, subdomain string, active bool) *tenant.Tenant {
	t := createTestTenant(subdomain, active)
	t.ID = id
	return t
}
