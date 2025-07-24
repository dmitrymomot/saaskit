# PostgreSQL Package

A high-performance PostgreSQL database wrapper with connection pooling, migrations, and health checks.

## Overview

The `pg` package provides a robust PostgreSQL client using the pgx driver. It handles connection management with retry capabilities, database migrations using goose, health checking, and comprehensive error handling. This package is thread-safe, context-aware, and designed for production use in concurrent applications.

## Features

- Connection pooling with configurable parameters and automatic retries
- Environment-based configuration with sensible defaults
- Database migrations powered by goose with structured logging
- Built-in health check functionality for monitoring
- Specialized error detection functions for common PostgreSQL error scenarios
- Context-aware operations for proper timeout and cancellation handling
- Thread-safe implementation for concurrent database access

## Usage

### Basic Connection

```go
import (
    "context"
    "log"
    "github.com/dmitrymomot/saaskit/pkg/pg"
)

func main() {
    // Create a PostgreSQL connection with context
    db, err := pg.Connect(context.Background(), pg.Config{
        ConnectionString: "postgres://user:password@localhost:5432/dbname",
        MaxOpenConns:     20,
        MaxIdleConns:     10,
        RetryAttempts:    5,
        RetryInterval:    time.Second * 3,
    })
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    // Database connection is ready for use
}
```

### Environment-Based Configuration

```go
import (
    "context"
    "log"
    "github.com/dmitrymomot/saaskit/pkg/config"
    "github.com/dmitrymomot/saaskit/pkg/pg"
)

func main() {
    // Load PostgreSQL config from environment variables
    var cfg pg.Config
    if err := config.Load(&cfg); err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    // Connect with loaded config
    db, err := pg.Connect(context.Background(), cfg)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()
}
```

### Database Migrations

```go
import (
    "context"
    "log/slog"
    "github.com/dmitrymomot/saaskit/pkg/pg"
)

func main() {
    // Create logger for migration output
    logger := slog.Default()

    // Configure database with migrations path
    cfg := pg.Config{
        ConnectionString: "postgres://user:password@localhost:5432/dbname",
        MigrationsPath:   "./migrations", // Path to migration files
        MigrationsTable:  "schema_migrations",
    }

    // Connect to database
    db, err := pg.Connect(context.Background(), cfg)
    if err != nil {
        logger.Error("Failed to connect to database", "error", err)
        return
    }
    defer db.Close()

    // Run migrations to latest version
    ctx := context.Background()
    if err := pg.Migrate(ctx, db, cfg, logger); err != nil {
        logger.Error("Migration failed", "error", err)
        return
    }

    logger.Info("Migrations completed successfully")
}
```

### Health Checking

```go
import (
    "context"
    "net/http"
    "github.com/dmitrymomot/saaskit/pkg/pg"
)

func setupHealthcheck(db *pgxpool.Pool) http.HandlerFunc {
    // Create a health check function
    healthCheck := pg.Healthcheck(db)

    // Return HTTP handler
    return func(w http.ResponseWriter, r *http.Request) {
        if err := healthCheck(r.Context()); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Write([]byte("Database unavailable"))
            return
        }
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Database healthy"))
    }
}
```

### Error Handling

```go
import (
    "errors"
    "github.com/dmitrymomot/saaskit/pkg/pg"
)

func handleDatabaseError(err error) {
    // Check for specific PostgreSQL error types
    switch {
    case pg.IsNotFoundError(err):
        // Handle record not found
    case pg.IsDuplicateKeyError(err):
        // Handle duplicate key constraint violation
    case pg.IsForeignKeyViolationError(err):
        // Handle foreign key constraint violation
    case pg.IsTxClosedError(err):
        // Handle closed transaction error
    case errors.Is(err, pg.ErrFailedToOpenDBConnection):
        // Handle connection failure
    case errors.Is(err, pg.ErrFailedToApplyMigrations):
        // Handle migration failure
    default:
        // Handle other database errors
    }
}
```

## Best Practices

1. **Connection Management**:
    - Always close connections with `defer db.Close()` to prevent leaks
    - Configure connection pool sizes based on your application's workload
    - Use context with timeouts for operations that might take long

2. **Error Handling**:
    - Use the provided error helper functions for precise error classification
    - Check for specific errors like duplicate keys and foreign key violations
    - Use `errors.Is()` for package-defined errors and `errors.As()` for PostgreSQL errors

3. **Migrations**:
    - Apply migrations at application startup to ensure schema consistency
    - Keep migration files idempotent and forward-only
    - Use transaction-based migrations to prevent partial application

4. **Health Monitoring**:
    - Implement the health check in your monitoring and readiness probes
    - Configure appropriate health check periods (`HealthCheckPeriod`)
    - Add timeout to health check context to prevent blocking

## API Reference

### Configuration

```go
type Config struct {
    ConnectionString  string        `env:"PG_CONN_URL,required"`                   // Connection string to the database
    MaxOpenConns      int32         `env:"PG_MAX_OPEN_CONNS" envDefault:"10"`      // Maximum number of open connections
    MaxIdleConns      int32         `env:"PG_MAX_IDLE_CONNS" envDefault:"5"`       // Maximum number of idle connections
    HealthCheckPeriod time.Duration `env:"PG_HEALTHCHECK_PERIOD" envDefault:"1m"`  // Period between health checks
    MaxConnIdleTime   time.Duration `env:"PG_MAX_CONN_IDLE_TIME" envDefault:"10m"` // Maximum idle time for a connection
    MaxConnLifetime   time.Duration `env:"PG_MAX_CONN_LIFETIME" envDefault:"30m"`  // Maximum lifetime for a connection
    RetryAttempts     int           `env:"PG_RETRY_ATTEMPTS" envDefault:"3"`       // Number of retry attempts
    RetryInterval     time.Duration `env:"PG_RETRY_INTERVAL" envDefault:"5s"`      // Interval between retry attempts
    MigrationsPath    string        `env:"PG_MIGRATIONS_PATH" envDefault:"db/migrations"`      // Path to migration files
    MigrationsTable   string        `env:"PG_MIGRATIONS_TABLE" envDefault:"schema_migrations"` // Table for migration history
}
```

### Functions

```go
func Connect(ctx context.Context, cfg Config) (*pgxpool.Pool, error)
```

Opens a new PostgreSQL connection pool with the provided configuration and retry logic.

```go
func Migrate(ctx context.Context, pool *pgxpool.Pool, cfg Config, log logger) error
```

Runs database migrations to the latest version using goose.

```go
func Healthcheck(conn *pgxpool.Pool) func(context.Context) error
```

Creates a health check function that verifies database connectivity.

### Error Detection Functions

```go
func IsNotFoundError(err error) bool
```

Checks if the error indicates that a record was not found.

```go
func IsDuplicateKeyError(err error) bool
```

Checks if the error indicates a duplicate key constraint violation.

```go
func IsForeignKeyViolationError(err error) bool
```

Checks if the error indicates a foreign key constraint violation.

```go
func IsTxClosedError(err error) bool
```

Checks if the error indicates that a transaction is already closed.

### Error Types

```go
var ErrFailedToOpenDBConnection = errors.New("failed to open db connection")
var ErrEmptyConnectionString = errors.New("empty postgres connection string, use DATABASE_URL env var")
var ErrHealthcheckFailed = errors.New("healthcheck failed, connection is not available")
var ErrFailedToParseDBConfig = errors.New("failed to parse db config")
var ErrFailedToApplyMigrations = errors.New("failed to apply migrations")
var ErrMigrationsDirNotFound = errors.New("migrations directory not found")
var ErrMigrationPathNotProvided = errors.New("migration path not provided")
```
