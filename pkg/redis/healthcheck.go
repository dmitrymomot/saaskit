package redis

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

// Healthcheck is a function that checks the health of the database.
// It returns an error if the database is not healthy.
func Healthcheck(client redis.UniversalClient) func(context.Context) error {
	return func(ctx context.Context) error {
		if _, err := client.Ping(ctx).Result(); err != nil {
			return errors.Join(ErrHealthcheckFailed, err)
		}
		return nil
	}
}
