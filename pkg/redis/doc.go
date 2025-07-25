// Package redis provides convenient helpers for connecting to a Redis server
// and working with it inside SaaSKit based applications.
//
// The package wraps the excellent go-redis client and adds:
//
//   - Robust `Connect` which retries the connection using the supplied
//     configuration.
//   - A thin `Storage` key-value wrapper that satisfies various cache / session
//     interfaces (e.g. Fiber storage).
//   - Health-check helpers to integrate Redis into HTTP or GRPC liveness /
//     readiness probes.
//
// Configuration is described by the `Config` struct whose fields can be
// populated from environment variables via github.com/caarlos0/env.
//
// # Usage
//
// Import the package:
//
//	import "github.com/dmitrymomot/saaskit/pkg/redis"
//
// Create configuration (most projects rely on env parsing):
//
//	cfg := redis.Config{
//	    ConnectionURL:  "redis://localhost:6379/0",
//	    RetryAttempts:  3,
//	    RetryInterval:  5 * time.Second,
//	    ConnectTimeout: 30 * time.Second,
//	}
//
// Connect with auto-retry:
//
//	ctx := context.Background()
//	client, err := redis.Connect(ctx, cfg)
//	if err != nil {
//	    // handle error, probably terminate the application
//	}
//	defer client.Close()
//
// Wrap client with the Storage helper if you need a simple key/value store:
//
//	store := redis.NewStorage(client)
//	if err := store.Set("foo", []byte("bar"), 0); err != nil {
//	    log.Fatal(err)
//	}
//
// Register a health-check in your observability stack:
//
//	checker := redis.Healthcheck(client)
//	if err := checker(ctx); err != nil {
//	    // redis is not healthy
//	}
//
// # Errors
//
// The package defines several sentinel errors (e.g. ErrRedisNotReady) that wrap
// the underlying go-redis errors using errors.Join. This makes it easy to
// compare and unwrap.
//
// # See Also
//
//   - https://github.com/redis/go-redis – underlying driver
//   - Fiber storage interface – github.com/gofiber/storage
package redis
