// Package httpserver provides a lightweight wrapper around net/http that adds
// graceful shutdown, configurable server timeouts, health-check handlers, and
// structured logging via slog.
//
// The core type is Server which embeds *http.Server behaviour and augments it
// with:
//
//   - Graceful Shutdown – Run blocks until the context is cancelled or an
//     interrupt/TERM signal is received and then shuts the server down using
//     http.Server.Shutdown with a configurable deadline.
//
//   - Functional Options – Construction is done through New or NewFromConfig
//     together with Option helpers such as WithAddr, WithReadTimeout and
//     WithLogger. This keeps the API stable while allowing incremental
//     features.
//
//   - Hooks – WithStartHook and WithStopHook let callers execute side-effects
//     around the server life-cycle.
//
//   - Health Checks – HealthCheckHandler returns an http.HandlerFunc that can
//     be mounted as both liveness and readiness probes.
//
// # Architecture
//
// A Server holds an internal immutable *config generated from the supplied
// Option values. Once Run is called the underlying *http.Server instance is
// initialised (or the one provided by WithServer is reused) and started in its
// own goroutine. A signal listener waits for os.Interrupt or syscall.SIGTERM
// and invokes graceful shutdown. All public errors are wrapped with ErrStart
// and ErrShutdown sentinel errors so they can be inspected with errors.Is.
//
// # Usage
//
//	import (
//		"context"
//		"log/slog"
//		"net/http"
//
//		"github.com/go-chi/chi/v5"
//		"github.com/dmitrymomot/saaskit/pkg/httpserver"
//	)
//
//	func main() {
//		r := chi.NewRouter()
//		r.Get("/healthz", httpserver.HealthCheckHandler(context.Background(), slog.Default()))
//
//		srv := httpserver.New(
//			httpserver.WithAddr(":8080"),
//			httpserver.WithShutdownTimeout(10*time.Second),
//		)
//
//		if err := srv.Run(context.Background(), r); err != nil {
//			slog.Error("server stopped", "err", err)
//		}
//	}
//
// # Errors
//
// Run wraps all listen errors with ErrStart, while Shutdown wraps underlying
// shutdown errors with ErrShutdown. Use errors.Is to distinguish them.
//
// See the README for more detailed examples.
package httpserver
