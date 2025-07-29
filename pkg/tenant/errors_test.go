package tenant_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/tenant"
)

func TestErrors(t *testing.T) {
	t.Parallel()

	t.Run("error messages are descriptive", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "tenant not found", tenant.ErrTenantNotFound.Error())
		assert.Equal(t, "invalid tenant identifier", tenant.ErrInvalidIdentifier.Error())
		assert.Equal(t, "no tenant in context", tenant.ErrNoTenantInContext.Error())
		assert.Equal(t, "tenant is inactive", tenant.ErrInactiveTenant.Error())
	})

	t.Run("errors can be compared with errors.Is", func(t *testing.T) {
		t.Parallel()

		// Wrap errors
		wrapped := errors.Join(tenant.ErrTenantNotFound, errors.New("additional context"))

		assert.ErrorIs(t, wrapped, tenant.ErrTenantNotFound)
		assert.True(t, errors.Is(wrapped, tenant.ErrTenantNotFound))
	})

	t.Run("errors are distinct", func(t *testing.T) {
		t.Parallel()

		// Each error should be unique
		assert.NotErrorIs(t, tenant.ErrTenantNotFound, tenant.ErrInvalidIdentifier)
		assert.NotErrorIs(t, tenant.ErrNoTenantInContext, tenant.ErrInactiveTenant)
	})

	t.Run("errors work with error wrapping", func(t *testing.T) {
		t.Parallel()

		// Simulate wrapping in provider implementation
		providerErr := errors.Join(errors.New("database error"), tenant.ErrTenantNotFound)

		assert.ErrorIs(t, providerErr, tenant.ErrTenantNotFound)
		assert.Contains(t, providerErr.Error(), "database error")
		assert.Contains(t, providerErr.Error(), "tenant not found")
	})
}
