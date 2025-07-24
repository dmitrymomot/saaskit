# Logger Package

A context-aware logging utility built on Go's `log/slog`. It exposes a single
`New` function that accepts functional options so you can configure output
format, level, and context extraction in a concise way.

## Overview

The package wraps `slog` handlers with a decorator that extracts values from
`context.Context`. A single `New` factory configures the logger using functional
options so you can easily select development or production defaults and attach
context extractors.

## Features

- JSON or text output formats
- `WithTextFormatter` and `WithJSONFormatter` convenience options
- Panics on unsupported log format
- Development and production helpers via options
- Context extractors for request IDs or user information
- Easy integration with `slog` via `SetAsDefault`
- Helper functions for common attributes like user and workspace IDs

## Usage

Create a development logger:

```go
log := logger.New(
    logger.WithDevelopment("my-service"),
)
logger.SetAsDefault(log)
```

Use text output explicitly:

```go
log := logger.New(logger.WithTextFormatter())
```

Use JSON output explicitly:

```go
log := logger.New(logger.WithJSONFormatter())
```

Add context values automatically:

```go
type ctxKey string
var requestIDKey ctxKey = "req-id"

log := logger.New(
    logger.WithProduction("api-service"),
    logger.WithContextValue("request_id", requestIDKey),
)

ctx := context.WithValue(context.Background(), requestIDKey, "123")
log.InfoContext(ctx, "processed request")
```

### Attribute Helpers

Use helper functions to keep attribute names consistent:

```go
log.Info("user login",
    logger.UserID(userID),
    logger.WorkspaceID(workspaceID),
    logger.Role("admin"),
    logger.RequestID(reqID),
    logger.Error(err),
)
```

Additional helpers include `logger.RequestID`, `logger.Error`, and `logger.Errors` for grouping multiple errors.

#### Grouping Helpers

Group related attributes under a single key or log multiple errors at once:

```go
err1 := errors.New("db error")
err2 := errors.New("cache error")

log.Error("startup failed", logger.Errors(err1, err2),
    logger.Group("request", slog.String("id", "42")))
```

## TODO

- Middleware examples for HTTP frameworks
