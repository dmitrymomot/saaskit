# MongoDB Package

A lightweight wrapper for MongoDB with connection management, environment-based configuration, and health monitoring.

## Installation

```bash
go get github.com/dmitrymomot/saaskit/pkg/mongo
```

## Overview

The `mongo` package provides a simple, robust interface to MongoDB in Go applications. It offers type-safe configuration, connection management with retry capabilities, and built-in health checks. This package is designed to simplify MongoDB integration in your applications while promoting best practices for connection handling and monitoring.

## Features

- Type-safe configuration with environment variable support
- Connection management with configurable retry logic
- Connection pooling with adjustable settings
- Built-in health check functionality for service monitoring
- Support for MongoDB Driver v2
- Thread-safe operations for concurrent use
- Simple database and collection access patterns

## Usage

### Basic Connection

```go
import (
    "context"
    "log"

    "github.com/dmitrymomot/saaskit/pkg/mongo"
)

// Create a MongoDB client with direct configuration
client, err := mongo.New(context.Background(), mongo.Config{
    ConnectionURL: "mongodb://localhost:27017",
})
if err != nil {
    // Handle connection error
    switch {
    case errors.Is(err, mongo.ErrFailedToConnectToMongo):
        log.Fatalf("Could not connect to MongoDB: %v", err)
    default:
        log.Fatalf("Unexpected error: %v", err)
    }
}
// Ensure proper cleanup
defer client.Disconnect(context.Background())

// Use the client
collection := client.Database("mydb").Collection("mycollection")
// Continue with MongoDB operations...
```

### Environment-Based Configuration

```go
import (
    "context"
    "log"

    "github.com/dmitrymomot/saaskit/pkg/config"
    "github.com/dmitrymomot/saaskit/pkg/mongo"
)

// Load MongoDB config from environment variables
cfg, err := config.Load[mongo.Config]()
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

// Create client with loaded config
client, err := mongo.New(context.Background(), cfg)
if err != nil {
    log.Fatalf("Failed to connect: %v", err)
}
defer client.Disconnect(context.Background())
```

### Connect to a Specific Database

```go
// Direct connection to a specific database
db, err := mongo.NewWithDatabase(context.Background(), cfg, "mydb")
if err != nil {
    log.Fatalf("Failed to connect to database: %v", err)
}

// Use the database directly
collection := db.Collection("users")
// Continue with collection operations...
```

### Health Check Integration

```go
import (
    "net/http"
)

// Create a MongoDB client
client, _ := mongo.New(context.Background(), cfg)

// Create a healthcheck function
healthCheck := mongo.Healthcheck(client)

// Use in HTTP handler
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    if err := healthCheck(r.Context()); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("MongoDB unavailable"))
        return
    }
    w.Write([]byte("MongoDB healthy"))
})
```

## Best Practices

1. **Connection Management**:
    - Always close connections with `defer client.Disconnect(ctx)`
    - Use appropriate context for operation timeouts

2. **Configuration**:
    - Set realistic connection timeouts based on your environment
    - Configure pool sizes appropriate for your workload
    - Enable retry options for better resilience

3. **Health Monitoring**:
    - Implement regular health checks in your application
    - Use the health check in readiness probes for container orchestration

4. **Error Handling**:
    - Check for specific MongoDB errors using `errors.Is`
    - Implement proper retry logic for transient failures

5. **Performance**:
    - Adjust connection pool settings based on your application's needs
    - Be aware of retry settings and their impact on latency

## API Reference

### Configuration Type

```go
type Config struct {
    ConnectionURL   string        `env:"MONGODB_URL,required"`           // Connection string URI
    ConnectTimeout  time.Duration `env:"MONGODB_CONNECT_TIMEOUT" envDefault:"10s"`
    MaxPoolSize     uint64        `env:"MONGODB_MAX_POOL_SIZE" envDefault:"100"`
    MinPoolSize     uint64        `env:"MONGODB_MIN_POOL_SIZE" envDefault:"1"`
    MaxConnIdleTime time.Duration `env:"MONGODB_MAX_CONN_IDLE_TIME" envDefault:"300s"`
    RetryWrites     bool          `env:"MONGODB_RETRY_WRITES" envDefault:"true"`
    RetryReads      bool          `env:"MONGODB_RETRY_READS" envDefault:"true"`
    RetryAttempts   int           `env:"MONGODB_RETRY_ATTEMPTS" envDefault:"3"`
    RetryInterval   time.Duration `env:"MONGODB_RETRY_INTERVAL" envDefault:"5s"`
}
```

### Functions

```go
func New(ctx context.Context, cfg Config) (*mongo.Client, error)
```

Creates a new MongoDB client with the provided configuration. Returns an error if the connection cannot be established after the configured retry attempts.

```go
func NewWithDatabase(ctx context.Context, cfg Config, database string) (*mongo.Database, error)
```

Creates a new MongoDB client and returns a specific database object. Useful for directly working with a particular database.

```go
func Healthcheck(client *mongo.Client) func(context.Context) error
```

Returns a function that checks the health of the MongoDB connection. The returned function accepts a context and returns an error if the health check fails.

### Error Types

```go
var ErrFailedToConnectToMongo = errors.New("failed to connect to mongo")
var ErrHealthcheckFailed = errors.New("mongo healthcheck failed")
```

## Known Issues

1. The retry logic implementation in the `New` function uses a range loop on an integer which is not the recommended approach in modern Go. Future versions will update this to use a more idiomatic approach.

2. There's currently no dedicated function for properly disconnecting from MongoDB, although you can use the standard `client.Disconnect(ctx)` method from the MongoDB driver.
