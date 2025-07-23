package session_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/cookie"
	"github.com/dmitrymomot/saaskit/pkg/session"
)

func setupManager(t *testing.T) *session.Manager {
	cookieMgr, err := cookie.New([]string{"test-secret-key-that-is-long-enough"})
	require.NoError(t, err)

	return session.New(
		session.WithCookieManager(cookieMgr),
		session.WithConfig(session.Config{
			CookieName:              "test-sid",
			AnonIdleTimeout:         30 * time.Minute,
			AnonMaxLifetime:         24 * time.Hour,
			AuthIdleTimeout:         2 * time.Hour,
			AuthMaxLifetime:         30 * 24 * time.Hour,
			ActivityUpdateThreshold: 5 * time.Minute,
			CleanupInterval:         0, // Disable cleanup for tests
		}),
	)
}

func TestManager_Ensure(t *testing.T) {
	manager := setupManager(t)
	ctx := context.Background()

	t.Run("creates new session", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)

		sess, err := manager.Ensure(ctx, w, r)
		assert.NoError(t, err)
		assert.NotNil(t, sess)
		assert.False(t, sess.IsAuthenticated())
		assert.NotEmpty(t, sess.Token)

		// Check cookie was set
		cookies := w.Result().Cookies()
		assert.Len(t, cookies, 1)
		assert.Equal(t, "test-sid", cookies[0].Name)
	})

	t.Run("returns existing valid session", func(t *testing.T) {
		// First request creates session
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequest("GET", "/", nil)
		sess1, err := manager.Ensure(ctx, w1, r1)
		require.NoError(t, err)

		// Second request with cookie should return same session
		r2 := httptest.NewRequest("GET", "/", nil)
		for _, c := range w1.Result().Cookies() {
			r2.AddCookie(c)
		}
		w2 := httptest.NewRecorder()

		sess2, err := manager.Ensure(ctx, w2, r2)
		assert.NoError(t, err)
		assert.Equal(t, sess1.ID, sess2.ID)
		assert.Equal(t, sess1.Token, sess2.Token)
	})

	t.Run("creates new session for invalid cookie", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{
			Name:  "test-sid",
			Value: "invalid-token",
		})

		sess, err := manager.Ensure(ctx, w, r)
		assert.NoError(t, err)
		assert.NotNil(t, sess)

		// Should have new token
		cookies := w.Result().Cookies()
		assert.Len(t, cookies, 1)
	})
}

func TestManager_Get(t *testing.T) {
	manager := setupManager(t)
	ctx := context.Background()

	t.Run("returns existing session", func(t *testing.T) {
		// Create session first
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequest("GET", "/", nil)
		sess1, err := manager.Ensure(ctx, w1, r1)
		require.NoError(t, err)

		// Get session
		r2 := httptest.NewRequest("GET", "/", nil)
		for _, c := range w1.Result().Cookies() {
			r2.AddCookie(c)
		}

		sess2, err := manager.Get(ctx, r2)
		assert.NoError(t, err)
		assert.Equal(t, sess1.ID, sess2.ID)
	})

	t.Run("returns error for no session", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		_, err := manager.Get(ctx, r)
		assert.ErrorIs(t, err, session.ErrSessionNotFound)
	})

	t.Run("returns error for invalid session", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{
			Name:  "test-sid",
			Value: "invalid-token",
		})

		_, err := manager.Get(ctx, r)
		assert.Error(t, err)
	})
}

func TestManager_Authenticate(t *testing.T) {
	manager := setupManager(t)
	ctx := context.Background()
	userID := uuid.New()

	t.Run("upgrades anonymous session", func(t *testing.T) {
		// Create anonymous session
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequest("GET", "/", nil)
		sess1, err := manager.Ensure(ctx, w1, r1)
		require.NoError(t, err)
		require.False(t, sess1.IsAuthenticated())

		// Authenticate
		r2 := httptest.NewRequest("POST", "/login", nil)
		for _, c := range w1.Result().Cookies() {
			r2.AddCookie(c)
		}
		w2 := httptest.NewRecorder()

		err = manager.Authenticate(ctx, w2, r2, userID)
		assert.NoError(t, err)

		// Get authenticated session
		r3 := httptest.NewRequest("GET", "/", nil)
		for _, c := range w2.Result().Cookies() {
			r3.AddCookie(c)
		}

		sess3, err := manager.Get(ctx, r3)
		assert.NoError(t, err)
		assert.True(t, sess3.IsAuthenticated())
		assert.Equal(t, userID, *sess3.UserID)
		// Token should be rotated
		assert.NotEqual(t, sess1.Token, sess3.Token)
	})

	t.Run("creates new authenticated session", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/login", nil)

		err := manager.Authenticate(ctx, w, r, userID)
		assert.NoError(t, err)

		// Check session was created
		cookies := w.Result().Cookies()
		assert.Len(t, cookies, 1)

		// Get the session
		r2 := httptest.NewRequest("GET", "/", nil)
		for _, c := range cookies {
			r2.AddCookie(c)
		}

		sess, err := manager.Get(ctx, r2)
		assert.NoError(t, err)
		assert.True(t, sess.IsAuthenticated())
		assert.Equal(t, userID, *sess.UserID)
	})
}

func TestManager_Destroy(t *testing.T) {
	manager := setupManager(t)
	ctx := context.Background()

	t.Run("destroys existing session", func(t *testing.T) {
		// Create session
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequest("GET", "/", nil)
		_, err := manager.Ensure(ctx, w1, r1)
		require.NoError(t, err)

		// Destroy session
		r2 := httptest.NewRequest("POST", "/logout", nil)
		for _, c := range w1.Result().Cookies() {
			r2.AddCookie(c)
		}
		w2 := httptest.NewRecorder()

		err = manager.Destroy(ctx, w2, r2)
		assert.NoError(t, err)

		// Check cookie was cleared
		cookies := w2.Result().Cookies()
		assert.Len(t, cookies, 1)
		assert.Equal(t, "test-sid", cookies[0].Name)
		assert.Equal(t, -1, cookies[0].MaxAge)

		// Session should not exist
		r3 := httptest.NewRequest("GET", "/", nil)
		for _, c := range w1.Result().Cookies() {
			r3.AddCookie(c)
		}
		_, err = manager.Get(ctx, r3)
		assert.Error(t, err)
	})

	t.Run("handles no session gracefully", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/logout", nil)

		err := manager.Destroy(ctx, w, r)
		assert.NoError(t, err)
	})
}

func TestManager_SetAndGetValue(t *testing.T) {
	manager := setupManager(t)
	ctx := context.Background()

	// Create session
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("GET", "/", nil)

	// Set value
	err := manager.Set(ctx, w1, r1, "key1", "value1")
	assert.NoError(t, err)

	// Get value
	r2 := httptest.NewRequest("GET", "/", nil)
	for _, c := range w1.Result().Cookies() {
		r2.AddCookie(c)
	}

	val, ok := manager.GetValue(ctx, r2, "key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	// Non-existent key
	val, ok = manager.GetValue(ctx, r2, "nonexistent")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestManager_Refresh(t *testing.T) {
	manager := setupManager(t)
	ctx := context.Background()

	// Create session
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("GET", "/", nil)
	sess1, err := manager.Ensure(ctx, w1, r1)
	require.NoError(t, err)
	originalExpiry := sess1.ExpiresAt

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Refresh session
	r2 := httptest.NewRequest("GET", "/", nil)
	for _, c := range w1.Result().Cookies() {
		r2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()

	err = manager.Refresh(ctx, w2, r2)
	assert.NoError(t, err)

	// Get refreshed session
	r3 := httptest.NewRequest("GET", "/", nil)
	for _, c := range w2.Result().Cookies() {
		r3.AddCookie(c)
	}

	sess3, err := manager.Get(ctx, r3)
	assert.NoError(t, err)
	assert.True(t, sess3.ExpiresAt.After(originalExpiry))
}

func TestManager_WithFingerprint(t *testing.T) {
	cookieMgr, err := cookie.New([]string{"test-secret-key-that-is-long-enough"})
	require.NoError(t, err)

	fingerprintFunc := func(r *http.Request) string {
		return r.Header.Get("User-Agent")
	}

	manager := session.New(
		session.WithCookieManager(cookieMgr),
		session.WithFingerprint(fingerprintFunc),
	)

	ctx := context.Background()

	t.Run("validates fingerprint", func(t *testing.T) {
		// Create session with fingerprint
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequest("GET", "/", nil)
		r1.Header.Set("User-Agent", "TestBrowser/1.0")

		sess1, err := manager.Ensure(ctx, w1, r1)
		assert.NoError(t, err)
		assert.Equal(t, "TestBrowser/1.0", sess1.Fingerprint)

		// Same fingerprint should work
		r2 := httptest.NewRequest("GET", "/", nil)
		r2.Header.Set("User-Agent", "TestBrowser/1.0")
		for _, c := range w1.Result().Cookies() {
			r2.AddCookie(c)
		}

		sess2, err := manager.Get(ctx, r2)
		assert.NoError(t, err)
		assert.Equal(t, sess1.ID, sess2.ID)

		// Different fingerprint should fail
		r3 := httptest.NewRequest("GET", "/", nil)
		r3.Header.Set("User-Agent", "DifferentBrowser/2.0")
		for _, c := range w1.Result().Cookies() {
			r3.AddCookie(c)
		}

		_, err = manager.Get(ctx, r3)
		assert.ErrorIs(t, err, session.ErrInvalidSession)
	})
}

func TestManager_WithHeaderTransport(t *testing.T) {
	manager := session.New(
		session.WithTransport(session.NewHeaderTransport("X-Session-Token")),
	)

	ctx := context.Background()

	t.Run("uses header transport", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)

		sess, err := manager.Ensure(ctx, w, r)
		assert.NoError(t, err)
		assert.NotEmpty(t, sess.Token)

		// Check header was set
		assert.NotEmpty(t, w.Header().Get("X-Session-Token"))
		assert.Contains(t, w.Header().Get("X-Session-Token"), "Bearer ")

		// Use token from header
		r2 := httptest.NewRequest("GET", "/", nil)
		r2.Header.Set("X-Session-Token", w.Header().Get("X-Session-Token"))

		sess2, err := manager.Get(ctx, r2)
		assert.NoError(t, err)
		assert.Equal(t, sess.ID, sess2.ID)
	})
}

func TestManager_PanicOnNoCookieManager(t *testing.T) {
	assert.Panics(t, func() {
		session.New() // No cookie manager provided
	})
}
