package pg

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Healthcheck returns a closure that validates database connectivity for health endpoints.
// Uses closure pattern to inject the connection dependency while maintaining
// compatibility with standard health check interfaces that expect func(context.Context) error.
func Healthcheck(conn *pgxpool.Pool) func(context.Context) error {
	return func(ctx context.Context) error {
		if err := conn.Ping(ctx); err != nil {
			return errors.Join(ErrHealthcheckFailed, err)
		}
		return nil
	}
}
