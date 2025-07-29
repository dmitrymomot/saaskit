package rbac_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/rbac"
)

func TestAuthorizer_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	roles := getTestRoles()
	source := rbac.NewInMemRoleSource(roles)
	auth, err := rbac.NewAuthorizer(ctx, source)
	require.NoError(t, err)

	t.Run("concurrent_can_checks", func(t *testing.T) {
		t.Parallel()

		const numGoroutines = 100
		const numOperations = 1000

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()

				for j := 0; j < numOperations; j++ {
					// Test different permission checks
					switch j % 4 {
					case 0:
						err := auth.Can("editor", "content.write")
						assert.NoError(t, err)
					case 1:
						err := auth.Can("editor", "content.read")
						assert.NoError(t, err)
					case 2:
						err := auth.Can("admin", "admin.users.create")
						assert.NoError(t, err)
					case 3:
						err := auth.Can("viewer", "content.write")
						assert.Error(t, err)
					}
				}
			}(i)
		}

		wg.Wait()
	})

	t.Run("concurrent_canany_checks", func(t *testing.T) {
		t.Parallel()

		const numGoroutines = 50
		const numOperations = 500

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		permissions := []string{"content.read", "content.write", "admin.users.create"}

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()

				for j := 0; j < numOperations; j++ {
					err := auth.CanAny("editor", permissions...)
					assert.NoError(t, err)
				}
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent_canall_checks", func(t *testing.T) {
		t.Parallel()

		const numGoroutines = 50
		const numOperations = 500

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		editorPermissions := []string{"content.read", "content.write"}
		adminPermissions := []string{"content.read", "admin.users.create"}

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()

				for j := 0; j < numOperations; j++ {
					if id%2 == 0 {
						err := auth.CanAll("editor", editorPermissions...)
						assert.NoError(t, err)
					} else {
						err := auth.CanAll("admin", adminPermissions...)
						assert.NoError(t, err)
					}
				}
			}(i)
		}

		wg.Wait()
	})

	t.Run("concurrent_context_checks", func(t *testing.T) {
		t.Parallel()

		const numGoroutines = 50
		const numOperations = 500

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()

				ctx := rbac.SetRoleToContext(context.Background(), "editor")

				for j := 0; j < numOperations; j++ {
					err := auth.CanFromContext(ctx, "content.write")
					assert.NoError(t, err)
				}
			}()
		}

		wg.Wait()
	})
}

// Stress test with race detector
func TestAuthorizer_RaceConditions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	roles := getTestRoles()
	source := rbac.NewInMemRoleSource(roles)
	auth, err := rbac.NewAuthorizer(ctx, source)
	require.NoError(t, err)

	const numGoroutines = 20
	const numOperations = 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Mix of different operations running concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				switch (id + j) % 6 {
				case 0:
					_ = auth.Can("admin", "admin.users.delete")
				case 1:
					_ = auth.CanAny("editor", "content.read", "content.write")
				case 2:
					_ = auth.CanAll("admin", "content.read", "admin.users.create")
				case 3:
					ctx := rbac.SetRoleToContext(context.Background(), "viewer")
					_ = auth.CanFromContext(ctx, "content.read")
				case 4:
					_ = auth.VerifyRole("editor")
				case 5:
					_ = auth.GetRoles()
				}
			}
		}(i)
	}

	wg.Wait()
}
