# OpenSearch Package

A lightweight wrapper for the official OpenSearch Go client with type-safe configuration.

## Installation

```bash
go get github.com/dmitrymomot/saaskit/pkg/opensearch
```

## Overview

The `opensearch` package provides a simple, type-safe interface to OpenSearch with environment-based configuration, automatic health checking, and standardized error handling. It is thread-safe and designed to be used in concurrent applications.

## Features

- Type-safe configuration with environment variable support
- Built-in health check on connection
- Comprehensive error handling with specific error types
- Simplified client initialization
- Thread-safe implementation for concurrent use
- Context-aware operations for cancellation support

## Usage

### Basic Connection

```go
import (
    "context"
    "github.com/dmitrymomot/saaskit/pkg/opensearch"
)

// Create an OpenSearch client with direct configuration
client, err := opensearch.New(context.Background(), opensearch.Config{
    Addresses: []string{"https://localhost:9200"},
    Username:  "admin",
    Password:  "admin",
})
if err != nil {
    // Handle error
    // Returns opensearch.ErrConnectionFailed or opensearch.ErrHealthcheckFailed
}

// Use the client
info, err := client.Info()
// Returns info about the OpenSearch cluster
```

### Environment-Based Configuration

```go
import (
    "context"
    "github.com/dmitrymomot/saaskit/pkg/config"
    "github.com/dmitrymomot/saaskit/pkg/opensearch"
)

// Load OpenSearch config from environment variables
cfg, err := config.Load[opensearch.Config]()
if err != nil {
    // Handle error
}

// Create client with loaded config
client, err := opensearch.New(context.Background(), cfg)
if err != nil {
    // Handle error
}
```

### Health Check Integration

```go
import (
    "context"
    "net/http"
    "errors"
    "github.com/dmitrymomot/saaskit/pkg/opensearch"
)

// Create a healthcheck function for an existing client
healthCheck := opensearch.Healthcheck(client)

// Use in HTTP handler
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    if err := healthCheck(r.Context()); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("OpenSearch unavailable"))
        return
    }
    w.Write([]byte("OpenSearch healthy"))
})

// Error Handling Example
if err := healthCheck(context.Background()); err != nil {
    switch {
    case errors.Is(err, opensearch.ErrHealthcheckFailed):
        // Handle health check failure
    default:
        // Handle unknown errors
    }
}
```

## Best Practices

1. **Secure Credential Storage**:
    - Store OpenSearch credentials in environment variables
    - Never hardcode sensitive information in your code

2. **Health Monitoring**:
    - Implement health checks in your service status endpoints
    - Set appropriate timeouts for health check context

3. **Performance Considerations**:
    - Configure retries appropriately based on your network reliability
    - Use context with timeouts for operations that may take long

4. **Error Handling**:
    - Use errors.Is() to check for specific error types
    - Properly wrap errors to maintain context

5. **Context Usage**:
    - Always pass appropriate context to control operation lifetimes
    - Use context cancellation for graceful shutdown

## API Reference

### Configuration

```go
type Config struct {
    Addresses    []string `env:"OPENSEARCH_ADDRESSES,required"`
    Username     string   `env:"OPENSEARCH_USERNAME,notEmpty"`
    Password     string   `env:"OPENSEARCH_PASSWORD,notEmpty"`
    MaxRetries   int      `env:"OPENSEARCH_MAX_RETRIES" default:"3"`
    DisableRetry bool     `env:"OPENSEARCH_DISABLE_RETRY" default:"false"`
}
```

### Functions

```go
func New(ctx context.Context, cfg Config) (*opensearch.Client, error)
```

Creates a new OpenSearch client with automatic health check. Returns an OpenSearch client or an error if the connection or health check fails.

```go
func Healthcheck(client *opensearch.Client) func(context.Context) error
```

Returns a function that checks the health of the OpenSearch connection. The returned function accepts a context and returns an error if the health check fails.

### Error Types

```go
var ErrConnectionFailed = errors.New("opensearch connection failed")
var ErrHealthcheckFailed = errors.New("opensearch healthcheck failed")
```
