package rbac_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/rbac"
)

func TestRoleContext(t *testing.T) {
	t.Run("set and get role", func(t *testing.T) {
		ctx := rbac.SetRoleToContext(context.Background(), "admin")
		role, ok := rbac.GetRoleFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, "admin", role)
	})

	t.Run("get role from empty context", func(t *testing.T) {
		role, ok := rbac.GetRoleFromContext(context.Background())
		assert.False(t, ok)
		assert.Empty(t, role)
	})

	t.Run("override role in context", func(t *testing.T) {
		ctx := rbac.SetRoleToContext(context.Background(), "user")
		ctx = rbac.SetRoleToContext(ctx, "admin")
		role, ok := rbac.GetRoleFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, "admin", role)
	})

	t.Run("context with wrong type", func(t *testing.T) {
		// Simulate wrong type in context
		type wrongKey struct{}
		ctx := context.WithValue(context.Background(), wrongKey{}, 123)
		role, ok := rbac.GetRoleFromContext(ctx)
		assert.False(t, ok)
		assert.Empty(t, role)
	})

	t.Run("empty role string", func(t *testing.T) {
		ctx := rbac.SetRoleToContext(context.Background(), "")
		role, ok := rbac.GetRoleFromContext(ctx)
		assert.True(t, ok)
		assert.Empty(t, role)
	})
}
