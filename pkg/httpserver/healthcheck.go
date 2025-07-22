package httpserver

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/dmitrymomot/saaskit/pkg/logger"
)

// HealthCheckHandler returns a HTTP handler that can be used for both
// liveness and readiness probes.
//
//   - Liveness: when no dependency functions are supplied the handler simply
//     returns 200 OK with body "ALIVE".
//   - Readiness: when one or more dependency functions are supplied each
//     function is executed; if they all succeed the handler returns 200 OK
//     with body "READY". If any of them return an error the handler returns
//     500 Internal Server Error with body "NOT_READY".
func HealthCheckHandler(ctx context.Context, log *slog.Logger, funcs ...func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Liveness probe: no dependency functions supplied.
		if len(funcs) == 0 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ALIVE"))
			return
		}

		// Readiness probe: verify all dependency functions succeed.
		for _, f := range funcs {
			if err := f(ctx); err != nil {
				log.ErrorContext(ctx, "Readiness check failed", logger.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("NOT_READY"))
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	}
}
