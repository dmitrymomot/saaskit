package httpserver

import "time"

type Config struct {
	Addr            string        `env:"HTTP_ADDR" envDefault:":8080"`          // Addr is the address the server listens on.
	ReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"30s"`    // ReadTimeout is the maximum duration for reading the entire request.
	WriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"30s"`   // WriteTimeout is the maximum duration before timing out writes of the response.
	IdleTimeout     time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"120s"`   // IdleTimeout is the maximum amount of time to wait for the next request when keep-alives are enabled.
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" envDefault:"5s"` // ShutdownTimeout is the time allowed for graceful shutdown.
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
