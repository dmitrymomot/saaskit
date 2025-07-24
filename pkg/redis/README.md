# Redis Package

A lightweight Redis client wrapper with connection management, healthchecks, and Fiber storage implementation.

## Installation

```bash
go get github.com/dmitrymomot/saaskit/pkg/redis
```

## Overview

The `redis` package provides a simplified, production-ready Redis client interface with robust connection management, environment-based configuration, health monitoring, and a ready-to-use Fiber storage implementation. The package is thread-safe and built on top of the official go-redis client.

## Features

- Simple connection API with automatic retry logic and configurable timeouts
- Environment variable based configuration using struct tags
- Built-in health check function for service monitoring
- Complete Storage implementation compatible with Fiber sessions and caching
- Comprehensive error types for better error handling
- Thread-safe implementation for concurrent usage
- Support for any redis.UniversalClient implementation (Client, ClusterClient, etc.)

## Usage

### Basic Connection

```go
import (
    "context"
    "github.com/dmitrymomot/saaskit/pkg/redis"
)

// Connect with configuration
client, err := redis.Connect(context.Background(), redis.Config{
    ConnectionURL: "redis://localhost:6379/0",
    RetryAttempts: 3,
    RetryInterval: 5 * time.Second,
    ConnectTimeout: 30 * time.Second,
})
if err != nil {
    // Handle connection error
    switch {
    case errors.Is(err, redis.ErrFailedToParseRedisConnString):
        // Handle invalid connection string
    case errors.Is(err, redis.ErrRedisNotReady):
        // Handle connection timeout
    default:
        // Handle other errors
    }
}
defer client.Close()

// Use the Redis client
err = client.Set(context.Background(), "key", "value", time.Hour).Err()
// Handle error if needed
```

### Loading Config from Environment

```go
import (
    "context"
    "github.com/dmitrymomot/saaskit/pkg/config"
    "github.com/dmitrymomot/saaskit/pkg/redis"
)

// Load from environment variables (REDIS_URL, REDIS_RETRY_ATTEMPTS, etc.)
cfg, err := config.Load[redis.Config]()
if err != nil {
    // Handle config loading error
}

// Connect using the loaded config
client, err := redis.Connect(context.Background(), cfg)
if err != nil {
    // Handle connection error
}
defer client.Close()
```

### Health Checking

```go
import (
    "context"
    "net/http"
    "github.com/dmitrymomot/saaskit/pkg/redis"
)

func setupHealthCheck(client redis.UniversalClient) http.HandlerFunc {
    // Create a health check function
    healthCheck := redis.Healthcheck(client)

    // Use in HTTP health endpoint
    return func(w http.ResponseWriter, r *http.Request) {
        if err := healthCheck(r.Context()); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Write([]byte("Redis unhealthy"))
            return
        }
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Redis healthy"))
    }
}
```

### Using with Fiber

```go
import (
    "context"
    "github.com/dmitrymomot/saaskit/pkg/redis"
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/session"
)

func setupFiberSession(client redis.UniversalClient) *session.Store {
    // Create Redis storage for Fiber
    storage := redis.NewStorage(client)

    // Create session store with Redis storage
    return session.New(session.Config{
        Storage: storage,
    })
}

// Usage in a Fiber route
app.Get("/", func(c *fiber.Ctx) error {
    store := c.Locals("store").(*session.Store)
    sess, err := store.Get(c)
    if err != nil {
        return err
    }

    // Get or set session values
    visits := sess.Get("visits")
    if visits == nil {
        sess.Set("visits", 1)
    } else {
        sess.Set("visits", visits.(int)+1)
    }

    if err := sess.Save(); err != nil {
        return err
    }

    return c.SendString(fmt.Sprintf("You have visited %d times", sess.Get("visits")))
})
```

## Best Practices

1. **Connection Management**:
    - Use the `defer client.Close()` pattern to ensure Redis connections are properly released
    - Set reasonable timeout values for your specific application needs
    - Configure retry attempts based on your application's resilience requirements

2. **Configuration**:
    - Use environment variables for configuration in production environments
    - Override connection timeout in high-traffic applications to prevent cascading failures

3. **Error Handling**:
    - Always check for specific error types using `errors.Is()` rather than string matching
    - Implement proper fallback mechanisms when Redis is temporarily unavailable

4. **Storage Usage**:
    - Set appropriate expiration times for cache or session data
    - Be cautious with the `Reset()` method as it clears the entire Redis database

## API Reference

### Configuration

```go
type Config struct {
    ConnectionURL  string        // Redis connection URL (env: REDIS_URL, default: "redis://localhost:6379/0")
    RetryAttempts  int           // Number of connection retry attempts (env: REDIS_RETRY_ATTEMPTS, default: 3)
    RetryInterval  time.Duration // Interval between retry attempts (env: REDIS_RETRY_INTERVAL, default: 5s)
    ConnectTimeout time.Duration // Connection timeout (env: REDIS_CONNECT_TIMEOUT, default: 30s)
}
```

### Functions

```go
func Connect(ctx context.Context, cfg Config) (*redis.Client, error)
```

Establishes a connection to Redis with retry logic and timeout handling.

```go
func Healthcheck(client redis.UniversalClient) func(context.Context) error
```

Creates a health check function that verifies Redis connection status.

```go
func NewStorage(redisClient redis.UniversalClient) *Storage
```

Creates a new Storage instance that implements Fiber's storage interface.

### Storage Methods

```go
func (s *Storage) Get(key string) ([]byte, error)
```

Retrieves a value from Redis by its key.

```go
func (s *Storage) Set(key string, val []byte, exp time.Duration) error
```

Stores a key-value pair in Redis with an optional expiration time.

```go
func (s *Storage) Delete(key string) error
```

Removes a key and its value from Redis.

```go
func (s *Storage) Reset() error
```

Removes all keys from the Redis database (FLUSHDB).

```go
func (s *Storage) Close() error
```

Terminates the connection to the Redis server.

```go
func (s *Storage) Conn() redis.UniversalClient
```

Returns the underlying Redis client for direct access.

```go
func (s *Storage) Keys() ([][]byte, error)
```

Returns all keys in the Redis database using SCAN.

### Error Types

```go
var ErrFailedToParseRedisConnString = errors.New("failed to parse redis connection string")
var ErrRedisNotReady = errors.New("redis did not become ready within the given time period")
var ErrEmptyConnectionURL = errors.New("empty redis connection URL")
var ErrHealthcheckFailed = errors.New("redis healthcheck failed")
```
