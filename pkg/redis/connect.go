package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Connect establishes a connection to a Redis server using the provided configuration.
// It attempts to connect multiple times based on the RetryAttempts config value,
// with a delay between attempts specified by RetryInterval.
//
// Parameters:
//   - ctx: Context for controlling the connection timeout and cancellation
//   - cfg: Configuration for the Redis connection including connection URL, timeouts, and retry settings
//
// Returns:
//   - *redis.Client: A connected Redis client if successful
//   - error: ErrFailedToParseRedisConnString if the connection URL is invalid
//     ErrRedisNotReady if all connection attempts fail
func Connect(ctx context.Context, cfg Config) (*redis.Client, error) {
	// Create a new context with a timeout for the connection
	ctx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	// Parse the redis connection string
	redisConnOpt, err := redis.ParseURL(cfg.ConnectionURL)
	if err != nil {
		return nil, errors.Join(ErrFailedToParseRedisConnString, err)
	}

	// Retry to connect to the redis server
	for range cfg.RetryAttempts {
		// Create a new redis client
		redisClient := redis.NewClient(redisConnOpt)

		// Check if the redis connection is established before proceeding with the application.
		if err := redisClient.Ping(ctx).Err(); err == nil {
			return redisClient, nil
		}

		// Close the failed client
		_ = redisClient.Close()

		// Check if context is done before waiting
		select {
		case <-ctx.Done():
			return nil, errors.Join(ErrRedisNotReady, ctx.Err())
		default:
			// Wait for the next retry interval
			time.Sleep(cfg.RetryInterval)
		}
	}

	return nil, ErrRedisNotReady
}
