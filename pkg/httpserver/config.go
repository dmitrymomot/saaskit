package httpserver

import "time"

type Config struct {
	Addr            string        `env:"HTTP_ADDR" envDefault:":8080"`          // Default port 8080 for container deployment compatibility
	ReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"30s"`    // 30s prevents slow client DoS attacks while allowing reasonable upload times
	WriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"30s"`   // 30s balances large response delivery with resource protection
	IdleTimeout     time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"120s"`   // 120s reduces connection churn for persistent clients
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" envDefault:"5s"` // 5s allows most requests to complete during graceful shutdown
}

// NewFromConfig creates a new Server from the provided Config.
// Only non-zero values from the config are applied.
func NewFromConfig(cfg Config, opts ...Option) *Server {
	configOpts := make([]Option, 0, 5)

	if cfg.Addr != "" {
		configOpts = append(configOpts, WithAddr(cfg.Addr))
	}
	if cfg.ReadTimeout > 0 {
		configOpts = append(configOpts, WithReadTimeout(cfg.ReadTimeout))
	}
	if cfg.WriteTimeout > 0 {
		configOpts = append(configOpts, WithWriteTimeout(cfg.WriteTimeout))
	}
	if cfg.IdleTimeout > 0 {
		configOpts = append(configOpts, WithIdleTimeout(cfg.IdleTimeout))
	}
	if cfg.ShutdownTimeout > 0 {
		configOpts = append(configOpts, WithShutdownTimeout(cfg.ShutdownTimeout))
	}

	// Append any additional options provided
	configOpts = append(configOpts, opts...)

	return New(configOpts...)
}
