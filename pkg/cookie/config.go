package cookie

import (
	"net/http"
	"strings"
)

const defaultOptionCapacity = 6 // Default capacity for options slice to avoid reallocation

type Config struct {
	Secrets  string        `env:"COOKIE_SECRETS" envDefault:""`
	Path     string        `env:"COOKIE_PATH" envDefault:"/"`
	Domain   string        `env:"COOKIE_DOMAIN" envDefault:""`
	MaxAge   int           `env:"COOKIE_MAX_AGE" envDefault:"0"`
	Secure   bool          `env:"COOKIE_SECURE" envDefault:"false"`
	HttpOnly bool          `env:"COOKIE_HTTP_ONLY" envDefault:"true"`
	SameSite http.SameSite `env:"COOKIE_SAME_SITE" envDefault:"2"` // SameSiteLaxMode (2) provides CSRF protection while maintaining functionality
}

func DefaultConfig() Config {
	return Config{
		Secrets:  "",
		Path:     "/",
		Domain:   "",
		MaxAge:   0,
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// parseSecrets splits comma-separated secrets for key rotation support.
// Empty strings are filtered out to prevent cryptographic vulnerabilities.
func (c Config) parseSecrets() []string {
	if c.Secrets == "" {
		return nil
	}

	parts := strings.Split(c.Secrets, ",")
	secrets := make([]string, 0, len(parts))

	for _, s := range parts {
		s = strings.TrimSpace(s)
		if s != "" {
			secrets = append(secrets, s)
		}
	}

	return secrets
}

// NewFromConfig creates a Manager from configuration.
// Only non-zero config values override defaults to preserve secure settings.
func NewFromConfig(cfg Config, opts ...Option) (*Manager, error) {
	secrets := cfg.parseSecrets()

	configOpts := make([]Option, 0, defaultOptionCapacity)

	if cfg.Path != "" {
		configOpts = append(configOpts, WithPath(cfg.Path))
	}
	if cfg.Domain != "" {
		configOpts = append(configOpts, WithDomain(cfg.Domain))
	}
	if cfg.MaxAge != 0 {
		configOpts = append(configOpts, WithMaxAge(cfg.MaxAge))
	}
	if cfg.Secure {
		configOpts = append(configOpts, WithSecure(cfg.Secure))
	}
	if cfg.HttpOnly {
		configOpts = append(configOpts, WithHTTPOnly(cfg.HttpOnly))
	}
	if cfg.SameSite != 0 {
		configOpts = append(configOpts, WithSameSite(cfg.SameSite))
	}

	configOpts = append(configOpts, opts...)

	return New(secrets, configOpts...)
}
