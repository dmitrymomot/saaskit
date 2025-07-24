# Config Package

A type-safe configuration loader for Go applications using environment variables.

## Overview

The `config` package provides a type-safe way to load configurations from environment variables with Go generics support. It implements a thread-safe singleton pattern ensuring each configuration type is loaded only once during application lifecycle. The package supports automatic `.env` file loading, custom environment file paths, and provides comprehensive error handling.

## Features

- Type-safe configuration loading with Go generics
- Thread-safe singleton implementation for each config type
- Automatic and custom path `.env` file loading
- Support for default values and required fields validation
- Environment variable expansion in configuration values
- Comprehensive error handling with specific error types
- Testing utilities for configuration management

## Usage

### Basic Example

```go
import (
    "fmt"
    "log"

    "github.com/dmitrymomot/saaskit/pkg/config"
)

type DatabaseConfig struct {
    Host     string `env:"DB_HOST" envDefault:"localhost"`
    Port     int    `env:"DB_PORT" envDefault:"5432"`
    Username string `env:"DB_USER,required"`
    Password string `env:"DB_PASS,required"`
}

func main() {
    // Create and load the configuration
    var dbConfig DatabaseConfig
    err := config.Load(&dbConfig)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Use the configuration
    fmt.Printf("Database connection: %s:%d\n", dbConfig.Host, dbConfig.Port)
}
```

### Loading From Custom Paths

```go
import (
    "log"

    "github.com/dmitrymomot/saaskit/pkg/config"
)

type AppConfig struct {
    APIKey    string `env:"API_KEY,required"`
    Debug     bool   `env:"DEBUG" envDefault:"false"`
    CacheTTL  int    `env:"CACHE_TTL" envDefault:"3600"`
}

func main() {
    // Load environment variables from a custom path
    err := config.LoadEnv("./config/.env.development")
    if err != nil {
        log.Fatalf("Failed to load environment file: %v", err)
    }

    // Or load from multiple files (later files override earlier ones)
    // err := config.LoadEnv("./config/.env.base", "./config/.env.local")

    var appConfig AppConfig
    err = config.Load(&appConfig)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
}
```

### Multiple Configuration Types

```go
// Server configuration
type ServerConfig struct {
    Port     int    `env:"SERVER_PORT" envDefault:"8080"`
    Host     string `env:"SERVER_HOST" envDefault:"0.0.0.0"`
    LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}

// Authentication configuration
type AuthConfig struct {
    JWTSecret     string `env:"JWT_SECRET,required"`
    TokenLifetime int    `env:"TOKEN_LIFETIME" envDefault:"3600"`
}

// Load different configurations independently
var serverCfg ServerConfig
err := config.Load(&serverCfg)
// serverCfg.Port now contains SERVER_PORT value or default (8080)

var authCfg AuthConfig
err = config.Load(&authCfg)
// authCfg.JWTSecret now contains JWT_SECRET value (required)
```

### Error Handling

```go
import (
    "errors"
    "log"

    "github.com/dmitrymomot/saaskit/pkg/config"
)

func loadConfig() {
    var myConfig MyConfig
    err := config.Load(&myConfig)
    if err != nil {
        switch {
        case errors.Is(err, config.ErrParsingConfig):
            // Handle parsing error (missing required field, invalid format)
        case errors.Is(err, config.ErrConfigNotLoaded):
            // Handle not loaded error
        case errors.Is(err, config.ErrNilPointer):
            // Handle nil pointer error
        default:
            // Handle other errors
        }
    }
}
```

## Best Practices

1. **Configuration Structure**:
    - Define separate configuration structs for different components
    - Group related settings within logical configuration types
    - Use clear, descriptive field and environment variable names

2. **Error Handling**:
    - Use `Load` for configurations that might fail at runtime
    - Use `MustLoad` only for configurations that are essential for startup
    - Check for specific error types when handling configuration errors

3. **Environment Variables**:
    - Use a consistent naming convention for environment variables
    - Prefix variables with component names to avoid collisions
    - Store sensitive information only in environment variables, not in code

4. **Default Values**:
    - Provide sensible defaults for non-critical configuration
    - Mark truly required fields with the `required` tag option

5. **Environment Files**:
    - Use different .env files for different environments (dev, test, prod)
    - Load environment-specific files before component-specific ones
    - Structure files from general to specific (base → environment → local)
    - Always call `LoadEnv` before any `Load` calls that depend on those variables

6. **Testing**:
    - Reset the cache between tests with `ResetCache()` when testing with different environment variables
    - Use `ForceReloadConfig()` to reload a specific configuration type after changing environment variables
    - Verify configuration state with `IsConfigLoaded[ConfigType]()`
    - Set up clean environment variables with `t.Setenv()` in your tests

## API Reference

### Functions

```go
func LoadEnv(filenames ...string) error
```

Loads environment variables from one or more .env files. If no paths are provided, attempts to load the default .env file from the current directory. Files are loaded in the order provided, with variables in later files taking precedence over earlier ones.

```go
func MustLoadEnv(filenames ...string)
```

Like LoadEnv but panics if loading fails. Useful when environment files are essential for the application to start.

```go
func Load[T any](v *T) error
```

Loads environment variables into the provided configuration struct pointer of type T. Ensures each configuration type is only loaded once and subsequent calls return the cached instance. Returns an error if parsing fails or a nil pointer is provided.

```go
func MustLoad[T any](v *T)
```

Like Load but panics if configuration loading fails. Useful for configurations that are required for the application to start.

```go
func ResetCache()
```

Clears all cached configuration instances. This is primarily useful in testing scenarios where environment variables change between test cases.

```go
func ForceReloadConfig[T any](v *T) error
```

Forces a reload of the specified configuration type, ignoring any previously cached instances. This is useful in testing when environment variables have changed.

```go
func IsConfigLoaded[T any]() bool
```

Returns true if the specified configuration type has already been loaded and cached.

### Environment Variable Tags

The package supports the following field tags:

```go
type Config struct {
    // Basic with default
    Port int `env:"PORT" envDefault:"8080"`

    // Required field
    APIKey string `env:"API_KEY,required"`

    // Lists with custom separator
    Hosts []string `env:"HOSTS" envSeparator:":"`

    // Duration parsing
    Timeout time.Duration `env:"TIMEOUT" envDefault:"30s"`

    // Environment variable expansion
    TempDir string `env:"TEMP_DIR,expand" envDefault:"${HOME}/tmp"`
}
```

### Error Types

```go
var ErrParsingConfig = errors.New("failed to parse environment variables into config")
var ErrInvalidConfigType = errors.New("invalid config type")
var ErrConfigNotLoaded = errors.New("configuration has not been loaded")
var ErrNilPointer = errors.New("nil pointer provided to config loader")
```
