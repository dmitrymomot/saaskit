package redis

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

// Healthcheck returns a function that pings Redis for health monitoring.
func Healthcheck(client redis.UniversalClient) func(context.Context) error {
	return func(ctx context.Context) error {
		if _, err := client.Ping(ctx).Result(); err != nil {
			return errors.Join(ErrHealthcheckFailed, err)
		}
		return nil
	}
}
