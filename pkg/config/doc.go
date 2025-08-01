// Package config provides a type-safe, generic and cached way to load
// application configuration from environment variables.
//
// It wraps popular libraries `github.com/joho/godotenv` and
// `github.com/caarlos0/env/v11` to deliver a convenient API that:
//
//   - Automatically loads values from `.env` file in the current working directory if present.
//   - Parses the environment into any Go struct using field tags.
//   - Caches each successfully loaded configuration type so it is only parsed
//     once for the lifetime of the process.
//   - Exposes a helper that panics on failure (`MustLoad`) for
//     scenarios where configuration is critical.
//
// # Architecture
//
// Internally the package keeps a singleton `configCache` that stores parsed
// struct copies keyed by their fully-qualified type name. Each key also holds a
// `sync.Once` instance guaranteeing the expensive parsing work is executed at
// most once per configuration type even when accessed from multiple goroutines
// concurrently.
//
// The exported helpers interact with the cache in a thread-safe manner using
// `sync.RWMutex`, while low-level parsing is delegated to `env.Parse`.
//
// # Usage
//
// First, create a struct describing your configuration and annotate its fields
// with `env` tags:
//
//	type DatabaseConfig struct {
//	    Host string `env:"DB_HOST,required"`
//	    Port int    `env:"DB_PORT" envDefault:"5432"`
//	    User string `env:"DB_USER,required"`
//	    Pass string `env:"DB_PASS,required"`
//	}
//
// The package automatically loads the default `.env` file if present, then populates the struct:
//
//	import "github.com/dmitrymomot/saaskit/pkg/config"
//
//	func main() {
//	    var db DatabaseConfig
//	    if err := config.Load(&db); err != nil {
//	        log.Fatalf("parsing env: %v", err)
//	    }
//
//	    // db is now populated and cached for future calls.
//	}
//
// Subsequent calls to `config.Load(&db)` will be served from the in-memory cache
// without re-parsing.
//
// # Error Handling
//
// The package defines sentinel errors that can be compared with `errors.Is`:
//
//   - `ErrParsingConfig`   – failed to parse env vars into struct.
//   - `ErrConfigNotLoaded` – requested config type has not been loaded yet.
//   - `ErrNilPointer`      – nil pointer passed to `Load`/`MustLoad`.
//
// # Performance Considerations
//
// Because each unique configuration struct is parsed only once and stored by
// value, lookups are extremely fast after the initial load. The cache does use
// additional memory proportional to the size of your configs.
//
// # See Also
//
//   - https://github.com/joho/godotenv – .env file loader.
//   - https://github.com/caarlos0/env – environment parser.
package config
