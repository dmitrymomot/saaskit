package redis

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Connect establishes a Redis connection with exponential retry logic.
// Fails fast on URL parsing errors, retries on connection failures.
func Connect(ctx context.Context, cfg Config) (redis.UniversalClient, error) {
	// Validate connection URL before attempting to parse
	if cfg.ConnectionURL == "" {
		return nil, ErrEmptyConnectionURL
	}

	// Ensure URL has valid Redis scheme
	if !strings.HasPrefix(cfg.ConnectionURL, "redis://") &&
		!strings.HasPrefix(cfg.ConnectionURL, "rediss://") {
		return nil, ErrFailedToParseRedisConnString
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	redisConnOpt, err := redis.ParseURL(cfg.ConnectionURL)
	if err != nil {
		return nil, errors.Join(ErrFailedToParseRedisConnString, err)
	}

	for range cfg.RetryAttempts {
		redisClient := redis.NewClient(redisConnOpt)

		if err := redisClient.Ping(ctx).Err(); err == nil {
			return redisClient, nil
		}

		_ = redisClient.Close()

		select {
		case <-ctx.Done():
			return nil, errors.Join(ErrRedisNotReady, ctx.Err())
		default:
			time.Sleep(cfg.RetryInterval)
		}
	}

	return nil, ErrRedisNotReady
}
