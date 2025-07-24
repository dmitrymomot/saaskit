package session

import (
	"time"

	"github.com/dmitrymomot/saaskit/pkg/cookie"
)

// Option is a functional option for configuring the Manager
type Option func(*Manager)

// WithStore sets a custom session store
func WithStore(store Store) Option {
	return func(m *Manager) {
		m.store = store
	}
}

// WithTransport sets a custom session transport
func WithTransport(transport Transport) Option {
	return func(m *Manager) {
		m.transport = transport
	}
}

// WithConfig sets custom configuration
func WithConfig(config Config) Option {
	return func(m *Manager) {
		m.config = config
	}
}

// WithCookieName sets the session cookie name
func WithCookieName(name string) Option {
	return func(m *Manager) {
		m.config.CookieName = name
	}
}

// WithIdleTimeout sets the idle timeout for sessions
func WithIdleTimeout(anon, auth time.Duration) Option {
	return func(m *Manager) {
		m.config.AnonIdleTimeout = anon
		m.config.AuthIdleTimeout = auth
	}
}

// WithMaxLifetime sets the maximum lifetime for sessions
func WithMaxLifetime(anon, auth time.Duration) Option {
	return func(m *Manager) {
		m.config.AnonMaxLifetime = anon
		m.config.AuthMaxLifetime = auth
	}
}

// WithActivityUpdateThreshold sets the minimum time between activity updates
func WithActivityUpdateThreshold(threshold time.Duration) Option {
	return func(m *Manager) {
		m.config.ActivityUpdateThreshold = threshold
	}
}

// WithCleanupInterval sets the cleanup interval for expired sessions
func WithCleanupInterval(interval time.Duration) Option {
	return func(m *Manager) {
		m.config.CleanupInterval = interval
	}
}

// WithFingerprint sets the fingerprint function
func WithFingerprint(fn FingerprintFunc) Option {
	return func(m *Manager) {
		m.fingerprintFunc = fn
	}
}

// WithCookieManager sets the cookie manager for the default cookie transport
func WithCookieManager(cookieMgr *cookie.Manager, opts ...cookie.Option) Option {
	return func(m *Manager) {
		m.cookieManager = cookieMgr
		m.cookieOptions = opts
	}
}
