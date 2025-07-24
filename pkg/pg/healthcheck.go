package pg

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Healthcheck is a function that checks the health of the database.
// It returns an error if the database is not healthy.
func Healthcheck(conn *pgxpool.Pool) func(context.Context) error {
	return func(ctx context.Context) error {
		if err := conn.Ping(ctx); err != nil {
			return errors.Join(ErrHealthcheckFailed, err)
		}
		return nil
	}
}
