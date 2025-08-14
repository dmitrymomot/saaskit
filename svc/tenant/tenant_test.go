package tenant_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/svc/tenant"
)

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

// createTestTenantWithID removed as it's unused
