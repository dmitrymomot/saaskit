package session_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/session"
)

func TestMemoryStore_Create(t *testing.T) {
	store := session.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	t.Run("successful create", func(t *testing.T) {
		sess := session.NewSession("token1", nil, "", 1*time.Hour)
		err := store.Create(ctx, sess)
		assert.NoError(t, err)

		// Verify it was stored
		retrieved, err := store.Get(ctx, "token1")
		assert.NoError(t, err)
		assert.Equal(t, sess.ID, retrieved.ID)
	})

	t.Run("nil session", func(t *testing.T) {
		err := store.Create(ctx, nil)
		assert.ErrorIs(t, err, session.ErrInvalidSession)
	})

	t.Run("empty token", func(t *testing.T) {
		sess := session.NewSession("", nil, "", 1*time.Hour)
		err := store.Create(ctx, sess)
		assert.ErrorIs(t, err, session.ErrInvalidSession)
	})

	t.Run("data isolation", func(t *testing.T) {
		sess := session.NewSession("token2", nil, "", 1*time.Hour)
		sess.Set("key", "value")

		err := store.Create(ctx, sess)
		require.NoError(t, err)

		// Modify original
		sess.Set("key", "modified")

		// Retrieved should have original value
		retrieved, err := store.Get(ctx, "token2")
		require.NoError(t, err)
		val, _ := retrieved.GetString("key")
		assert.Equal(t, "value", val)
	})
}

func TestMemoryStore_Get(t *testing.T) {
	store := session.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	t.Run("existing session", func(t *testing.T) {
		sess := session.NewSession("token1", nil, "", 1*time.Hour)
		sess.Set("key", "value")

		err := store.Create(ctx, sess)
		require.NoError(t, err)

		retrieved, err := store.Get(ctx, "token1")
		assert.NoError(t, err)
		assert.Equal(t, sess.ID, retrieved.ID)
		val, _ := retrieved.GetString("key")
		assert.Equal(t, "value", val)
	})

	t.Run("non-existent session", func(t *testing.T) {
		_, err := store.Get(ctx, "nonexistent")
		assert.ErrorIs(t, err, session.ErrSessionNotFound)
	})

	t.Run("expired session", func(t *testing.T) {
		sess := session.NewSession("expired", nil, "", 1*time.Hour)
		sess.ExpiresAt = time.Now().Add(-1 * time.Hour)

		err := store.Create(ctx, sess)
		require.NoError(t, err)

		_, err = store.Get(ctx, "expired")
		assert.ErrorIs(t, err, session.ErrSessionExpired)
	})
}

func TestMemoryStore_Update(t *testing.T) {
	store := session.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		sess := session.NewSession("token1", nil, "", 1*time.Hour)
		sess.Set("key", "value1")

		err := store.Create(ctx, sess)
		require.NoError(t, err)

		// Update session
		sess.Set("key", "value2")
		sess.Set("newkey", "newvalue")

		err = store.Update(ctx, sess)
		assert.NoError(t, err)

		// Verify updates
		retrieved, err := store.Get(ctx, "token1")
		require.NoError(t, err)
		val, _ := retrieved.GetString("key")
		assert.Equal(t, "value2", val)
		val, _ = retrieved.GetString("newkey")
		assert.Equal(t, "newvalue", val)
	})

	t.Run("non-existent session", func(t *testing.T) {
		sess := session.NewSession("nonexistent", nil, "", 1*time.Hour)
		err := store.Update(ctx, sess)
		assert.ErrorIs(t, err, session.ErrSessionNotFound)
	})

	t.Run("nil session", func(t *testing.T) {
		err := store.Update(ctx, nil)
		assert.ErrorIs(t, err, session.ErrInvalidSession)
	})
}

func TestMemoryStore_UpdateActivity(t *testing.T) {
	store := session.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		sess := session.NewSession("token1", nil, "", 1*time.Hour)
		originalTime := sess.LastActivityAt

		err := store.Create(ctx, sess)
		require.NoError(t, err)

		// Update activity
		newTime := time.Now().Add(10 * time.Minute)
		err = store.UpdateActivity(ctx, "token1", newTime)
		assert.NoError(t, err)

		// Verify update
		retrieved, err := store.Get(ctx, "token1")
		require.NoError(t, err)
		assert.Equal(t, newTime.Unix(), retrieved.LastActivityAt.Unix())
		assert.NotEqual(t, originalTime.Unix(), retrieved.LastActivityAt.Unix())
	})

	t.Run("non-existent session", func(t *testing.T) {
		err := store.UpdateActivity(ctx, "nonexistent", time.Now())
		assert.ErrorIs(t, err, session.ErrSessionNotFound)
	})
}

func TestMemoryStore_Delete(t *testing.T) {
	store := session.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		sess := session.NewSession("token1", nil, "", 1*time.Hour)

		err := store.Create(ctx, sess)
		require.NoError(t, err)

		// Verify it exists
		_, err = store.Get(ctx, "token1")
		assert.NoError(t, err)

		// Delete it
		err = store.Delete(ctx, "token1")
		assert.NoError(t, err)

		// Verify it's gone
		_, err = store.Get(ctx, "token1")
		assert.ErrorIs(t, err, session.ErrSessionNotFound)
	})

	t.Run("delete non-existent", func(t *testing.T) {
		// Should not error
		err := store.Delete(ctx, "nonexistent")
		assert.NoError(t, err)
	})
}

func TestMemoryStore_DeleteExpired(t *testing.T) {
	store := session.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	// Create mix of valid and expired sessions
	valid1 := session.NewSession("valid1", nil, "", 1*time.Hour)
	valid2 := session.NewSession("valid2", nil, "", 1*time.Hour)

	expired1 := session.NewSession("expired1", nil, "", 1*time.Hour)
	expired1.ExpiresAt = time.Now().Add(-1 * time.Hour)

	expired2 := session.NewSession("expired2", nil, "", 1*time.Hour)
	expired2.ExpiresAt = time.Now().Add(-2 * time.Hour)

	// Create all sessions
	require.NoError(t, store.Create(ctx, valid1))
	require.NoError(t, store.Create(ctx, valid2))
	require.NoError(t, store.Create(ctx, expired1))
	require.NoError(t, store.Create(ctx, expired2))

	// Delete expired
	err := store.DeleteExpired(ctx)
	assert.NoError(t, err)

	// Valid sessions should still exist
	_, err = store.Get(ctx, "valid1")
	assert.NoError(t, err)
	_, err = store.Get(ctx, "valid2")
	assert.NoError(t, err)

	// Expired sessions should be gone
	_, err = store.Get(ctx, "expired1")
	assert.ErrorIs(t, err, session.ErrSessionNotFound)
	_, err = store.Get(ctx, "expired2")
	assert.ErrorIs(t, err, session.ErrSessionNotFound)
}

func TestMemoryStore_DeleteByUserID(t *testing.T) {
	store := session.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()
	userID1 := uuid.New()
	userID2 := uuid.New()

	// Create sessions for different users
	user1Session1 := session.NewSession("user1-1", &userID1, "", 1*time.Hour)
	user1Session2 := session.NewSession("user1-2", &userID1, "", 1*time.Hour)
	user2Session := session.NewSession("user2-1", &userID2, "", 1*time.Hour)
	anonSession := session.NewSession("anon", nil, "", 1*time.Hour)

	// Create all sessions
	require.NoError(t, store.Create(ctx, user1Session1))
	require.NoError(t, store.Create(ctx, user1Session2))
	require.NoError(t, store.Create(ctx, user2Session))
	require.NoError(t, store.Create(ctx, anonSession))

	// Delete user1's sessions
	err := store.DeleteByUserID(ctx, userID1.String())
	assert.NoError(t, err)

	// User1's sessions should be gone
	_, err = store.Get(ctx, "user1-1")
	assert.ErrorIs(t, err, session.ErrSessionNotFound)
	_, err = store.Get(ctx, "user1-2")
	assert.ErrorIs(t, err, session.ErrSessionNotFound)

	// Other sessions should still exist
	_, err = store.Get(ctx, "user2-1")
	assert.NoError(t, err)
	_, err = store.Get(ctx, "anon")
	assert.NoError(t, err)

	t.Run("invalid user ID", func(t *testing.T) {
		err := store.DeleteByUserID(ctx, "invalid-uuid")
		assert.Error(t, err)
	})
}

func TestMemoryStore_Stats(t *testing.T) {
	store := session.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()
	userID := uuid.New()

	// Create mix of sessions
	auth1 := session.NewSession("auth1", &userID, "", 1*time.Hour)
	auth2 := session.NewSession("auth2", &userID, "", 1*time.Hour)
	anon1 := session.NewSession("anon1", nil, "", 1*time.Hour)
	anon2 := session.NewSession("anon2", nil, "", 1*time.Hour)
	anon3 := session.NewSession("anon3", nil, "", 1*time.Hour)

	require.NoError(t, store.Create(ctx, auth1))
	require.NoError(t, store.Create(ctx, auth2))
	require.NoError(t, store.Create(ctx, anon1))
	require.NoError(t, store.Create(ctx, anon2))
	require.NoError(t, store.Create(ctx, anon3))

	total, authenticated, anonymous := store.Stats()
	assert.Equal(t, 5, total)
	assert.Equal(t, 2, authenticated)
	assert.Equal(t, 3, anonymous)
}

func TestMemoryStore_Cleanup(t *testing.T) {
	// Test with cleanup enabled
	store := session.NewMemoryStore(50 * time.Millisecond)
	defer store.Close()

	ctx := context.Background()

	// Create expired session
	expired := session.NewSession("expired", nil, "", 1*time.Hour)
	expired.ExpiresAt = time.Now().Add(-1 * time.Hour)

	require.NoError(t, store.Create(ctx, expired))

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)

	// Expired session should be gone
	_, err := store.Get(ctx, "expired")
	assert.ErrorIs(t, err, session.ErrSessionNotFound)
}

func TestMemoryStore_Concurrency(t *testing.T) {
	store := session.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	// Create initial session
	sess := session.NewSession("concurrent", nil, "", 1*time.Hour)
	sess.Set("counter", 0)
	require.NoError(t, store.Create(ctx, sess))

	// Run concurrent operations
	done := make(chan bool)
	for range 10 {
		go func() {
			for j := 0; j < 100; j++ {
				// Get
				s, _ := store.Get(ctx, "concurrent")
				if s != nil {
					// Update
					counter, _ := s.GetInt("counter")
					s.Set("counter", counter+1)
					store.Update(ctx, s)
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}

	// Verify session still exists and is valid
	final, err := store.Get(ctx, "concurrent")
	assert.NoError(t, err)
	assert.NotNil(t, final)
}
