package tenant_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/tenant"
)

func TestWithTenant(t *testing.T) {
	t.Parallel()

	t.Run("adds tenant to context", func(t *testing.T) {
		t.Parallel()

		testTenant := createTestTenant("acme", true)
		ctx := tenant.WithTenant(context.Background(), testTenant)

		retrieved, ok := tenant.FromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, testTenant, retrieved)
	})

	t.Run("overwrites existing tenant in context", func(t *testing.T) {
		t.Parallel()

		tenant1 := createTestTenant("acme", true)
		tenant2 := createTestTenant("globex", true)

		ctx := tenant.WithTenant(context.Background(), tenant1)
		ctx = tenant.WithTenant(ctx, tenant2)

		retrieved, ok := tenant.FromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, tenant2, retrieved)
	})
}

func TestFromContext(t *testing.T) {
	t.Parallel()

	t.Run("retrieves tenant from context", func(t *testing.T) {
		t.Parallel()

		testTenant := createTestTenant("acme", true)
		ctx := tenant.WithTenant(context.Background(), testTenant)

		retrieved, ok := tenant.FromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, testTenant, retrieved)
	})

	t.Run("returns nil and false for empty context", func(t *testing.T) {
		t.Parallel()

		retrieved, ok := tenant.FromContext(context.Background())
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("returns false for wrong type in context", func(t *testing.T) {
		t.Parallel()

		// Use a different key type to simulate wrong data
		type wrongKey struct{}
		ctx := context.WithValue(context.Background(), wrongKey{}, "not a tenant")

		retrieved, ok := tenant.FromContext(ctx)
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})
}

func TestIDFromContext(t *testing.T) {
	t.Parallel()

	t.Run("retrieves tenant ID from context", func(t *testing.T) {
		t.Parallel()

		testTenant := createTestTenant("acme", true)
		ctx := tenant.WithTenant(context.Background(), testTenant)

		id, ok := tenant.IDFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, testTenant.ID, id)
	})

	t.Run("returns zero UUID and false for empty context", func(t *testing.T) {
		t.Parallel()

		id, ok := tenant.IDFromContext(context.Background())
		assert.False(t, ok)
		assert.Equal(t, uuid.UUID{}, id)
	})

	t.Run("returns zero UUID and false for nil tenant", func(t *testing.T) {
		t.Parallel()

		ctx := tenant.WithTenant(context.Background(), nil)

		id, ok := tenant.IDFromContext(ctx)
		assert.False(t, ok)
		assert.Equal(t, uuid.UUID{}, id)
	})
}

func TestMustFromContext(t *testing.T) {
	t.Parallel()

	t.Run("retrieves tenant from context", func(t *testing.T) {
		t.Parallel()

		testTenant := createTestTenant("acme", true)
		ctx := tenant.WithTenant(context.Background(), testTenant)

		retrieved := tenant.MustFromContext(ctx)
		assert.Equal(t, testTenant, retrieved)
	})

	t.Run("panics when no tenant in context", func(t *testing.T) {
		t.Parallel()

		assert.PanicsWithValue(t, "tenant: no tenant in context", func() {
			tenant.MustFromContext(context.Background())
		})
	})

	t.Run("panics when tenant is nil", func(t *testing.T) {
		t.Parallel()

		ctx := tenant.WithTenant(context.Background(), nil)

		assert.PanicsWithValue(t, "tenant: no tenant in context", func() {
			tenant.MustFromContext(ctx)
		})
	})
}

func TestContext_Propagation(t *testing.T) {
	t.Parallel()

	t.Run("tenant propagates through context chain", func(t *testing.T) {
		t.Parallel()

		testTenant := createTestTenant("acme", true)

		// Create a chain of contexts
		ctx := context.Background()
		ctx = context.WithValue(ctx, "key1", "value1")
		ctx = tenant.WithTenant(ctx, testTenant)
		ctx = context.WithValue(ctx, "key2", "value2")

		// Verify tenant is still accessible
		retrieved, ok := tenant.FromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, testTenant, retrieved)

		// Verify other context values are preserved
		assert.Equal(t, "value1", ctx.Value("key1"))
		assert.Equal(t, "value2", ctx.Value("key2"))
	})

	t.Run("cancelled context still returns tenant", func(t *testing.T) {
		t.Parallel()

		testTenant := createTestTenant("acme", true)

		ctx, cancel := context.WithCancel(context.Background())
		ctx = tenant.WithTenant(ctx, testTenant)

		// Cancel the context
		cancel()

		// Tenant should still be retrievable
		retrieved, ok := tenant.FromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, testTenant, retrieved)
	})
}
