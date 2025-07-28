package tenant_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/tenant"
)

func TestNoOpCache(t *testing.T) {
	t.Parallel()

	t.Run("always returns cache miss", func(t *testing.T) {
		t.Parallel()

		cache := &tenant.NoOpCache{}
		testTenant := createTestTenant("acme", true)

		// Set should succeed but do nothing
		err := cache.Set(context.Background(), "key1", testTenant)
		assert.NoError(t, err)

		// Get should always return cache miss
		retrieved, ok := cache.Get(context.Background(), "key1")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("delete always succeeds", func(t *testing.T) {
		t.Parallel()

		cache := &tenant.NoOpCache{}

		// Delete should always succeed
		err := cache.Delete(context.Background(), "nonexistent")
		assert.NoError(t, err)
	})
}

// createTestTenant function is defined in tenant_test.go