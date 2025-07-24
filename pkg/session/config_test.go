package session_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/cookie"
	"github.com/dmitrymomot/saaskit/pkg/session"
)

func TestNewFromConfig(t *testing.T) {
	cookieMgr, err := cookie.New([]string{"test-secret-key-that-is-long-enough"})
	require.NoError(t, err)

	cfg := session.Config{
		CookieName:              "test-session",
		AnonIdleTimeout:         15 * time.Minute,
		AnonMaxLifetime:         12 * time.Hour,
		AuthIdleTimeout:         1 * time.Hour,
		AuthMaxLifetime:         7 * 24 * time.Hour,
		ActivityUpdateThreshold: 2 * time.Minute,
		CleanupInterval:         10 * time.Minute,
	}

	manager := session.NewFromConfig(cfg,
		session.WithCookieManager(cookieMgr),
	)

	// Test that configuration was applied
	assert.NotNil(t, manager)

	// Create a session to verify cookie name
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	_, err = manager.Ensure(r.Context(), w, r)
	assert.NoError(t, err)

	// Check cookie name
	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, "test-session", cookies[0].Name)
}

func TestDefaultConfig(t *testing.T) {
	cfg := session.DefaultConfig()

	assert.Equal(t, "sid", cfg.CookieName)
	assert.Equal(t, 30*time.Minute, cfg.AnonIdleTimeout)
	assert.Equal(t, 24*time.Hour, cfg.AnonMaxLifetime)
	assert.Equal(t, 2*time.Hour, cfg.AuthIdleTimeout)
	assert.Equal(t, 30*24*time.Hour, cfg.AuthMaxLifetime)
	assert.Equal(t, 5*time.Minute, cfg.ActivityUpdateThreshold)
	assert.Equal(t, 5*time.Minute, cfg.CleanupInterval)
}

func TestConfig_GetTimeouts(t *testing.T) {
	cfg := session.Config{
		AnonIdleTimeout: 10 * time.Minute,
		AnonMaxLifetime: 1 * time.Hour,
		AuthIdleTimeout: 30 * time.Minute,
		AuthMaxLifetime: 24 * time.Hour,
	}

	t.Run("anonymous", func(t *testing.T) {
		idle, max := cfg.GetTimeouts(false)
		assert.Equal(t, 10*time.Minute, idle)
		assert.Equal(t, 1*time.Hour, max)
	})

	t.Run("authenticated", func(t *testing.T) {
		idle, max := cfg.GetTimeouts(true)
		assert.Equal(t, 30*time.Minute, idle)
		assert.Equal(t, 24*time.Hour, max)
	})
}
