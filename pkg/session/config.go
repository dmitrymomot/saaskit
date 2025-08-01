package session

import "time"

// Config holds session configuration
type Config struct {
	// CookieName is the name of the session cookie (default: "sid")
	CookieName string `env:"SESSION_COOKIE_NAME" envDefault:"sid"`

	AnonIdleTimeout time.Duration `env:"SESSION_ANON_IDLE_TIMEOUT" envDefault:"30m"`
	AnonMaxLifetime time.Duration `env:"SESSION_ANON_MAX_LIFETIME" envDefault:"24h"`

	AuthIdleTimeout time.Duration `env:"SESSION_AUTH_IDLE_TIMEOUT" envDefault:"2h"`
	AuthMaxLifetime time.Duration `env:"SESSION_AUTH_MAX_LIFETIME" envDefault:"720h"`

	// ActivityUpdateThreshold is the minimum time between activity updates
	ActivityUpdateThreshold time.Duration `env:"SESSION_ACTIVITY_UPDATE_THRESHOLD" envDefault:"5m"`

	// CleanupInterval for expired sessions (0 to disable)
	CleanupInterval time.Duration `env:"SESSION_CLEANUP_INTERVAL" envDefault:"5m"`

	// SecureCookies enables the Secure flag on session cookies (recommended for production)
	SecureCookies bool `env:"SESSION_SECURE_COOKIES" envDefault:"false"`
}

// DefaultConfig returns default session configuration
func DefaultConfig() Config {
	return Config{
		CookieName:              "sid",
		AnonIdleTimeout:         30 * time.Minute,
		AnonMaxLifetime:         24 * time.Hour,
		AuthIdleTimeout:         2 * time.Hour,
		AuthMaxLifetime:         30 * 24 * time.Hour,
		ActivityUpdateThreshold: 5 * time.Minute,
		CleanupInterval:         5 * time.Minute,
		SecureCookies:           false,
	}
}

// GetTimeouts returns idle and max lifetime based on session state
func (c Config) GetTimeouts(isAuthenticated bool) (idle, max time.Duration) {
	if isAuthenticated {
		return c.AuthIdleTimeout, c.AuthMaxLifetime
	}
	return c.AnonIdleTimeout, c.AnonMaxLifetime
}

// NewFromConfig creates a new Manager from the provided Config.
// Requires Store via options. Cookie manager required for default cookie transport.
func NewFromConfig(cfg Config, opts ...Option) *Manager {
	configOpts := []Option{
		WithConfig(cfg),
	}

	configOpts = append(configOpts, opts...)

	return New(configOpts...)
}
