package httpserver

import (
	"log/slog"
	"net/http"
	"time"
)

// Option configures the HTTP server.
type Option func(*config)

// WithAddr sets the address the server listens on.
func WithAddr(addr string) Option {
	if addr == "" {
		panic("WithAddr: addr cannot be empty")
	}
	return func(c *config) { c.addr = addr }
}

// WithReadTimeout sets the maximum duration for reading the entire request.
func WithReadTimeout(d time.Duration) Option {
	if d <= 0 {
		panic("WithReadTimeout: duration must be > 0")
	}
	return func(c *config) { c.readTimeout = d }
}

// WithWriteTimeout sets the maximum duration before timing out writes of the response.
func WithWriteTimeout(d time.Duration) Option {
	if d <= 0 {
		panic("WithWriteTimeout: duration must be > 0")
	}
	return func(c *config) { c.writeTimeout = d }
}

// WithIdleTimeout sets the maximum amount of time to wait for the next request when keep-alives are enabled.
func WithIdleTimeout(d time.Duration) Option {
	if d <= 0 {
		panic("WithIdleTimeout: duration must be > 0")
	}
	return func(c *config) { c.idleTimeout = d }
}

// WithShutdownTimeout sets the time allowed for graceful shutdown.
func WithShutdownTimeout(d time.Duration) Option {
	if d <= 0 {
		panic("WithShutdownTimeout: duration must be > 0")
	}
	return func(c *config) { c.shutdownTimeout = d }
}

// WithServer uses the provided http.Server instance. The server's Handler and
// timeout fields may be modified; values already set take precedence over
// package defaults.
func WithServer(srv *http.Server) Option {
	if srv == nil {
		panic("WithServer: nil server")
	}
	return func(c *config) { c.server = srv }
}

// WithLogger supplies an external slog.Logger instance. If nil, a noop logger is used.
func WithLogger(l *slog.Logger) Option {
	return func(c *config) { c.logger = l }
}

// WithStartHook registers a callback that runs when the server begins listening.
func WithStartHook(h func(*slog.Logger)) Option {
	if h == nil {
		panic("WithStartHook: nil hook")
	}
	return func(c *config) {
		c.startHooks = append(c.startHooks, h)
	}
}

// WithStopHook registers a callback that runs after the server shuts down.
func WithStopHook(h func(*slog.Logger)) Option {
	if h == nil {
		panic("WithStopHook: nil hook")
	}
	return func(c *config) {
		c.stopHooks = append(c.stopHooks, h)
	}
}
