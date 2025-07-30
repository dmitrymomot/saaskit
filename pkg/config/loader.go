package config

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// configCache provides a type-safe way to store and retrieve configuration
// instances using generics
type configCache struct {
	mu     sync.RWMutex
	values map[string]any
	onces  map[string]*sync.Once
}

var (
	// globalCache is the singleton instance for caching configurations
	globalCache = &configCache{
		values: make(map[string]any),
		onces:  make(map[string]*sync.Once),
	}

	defaultEnvLoaded sync.Once
)

// Load loads environment variables into the provided configuration struct.
// It ensures that each unique configuration type is only loaded once
// throughout the application lifecycle.
//
// The function first attempts to load the default .env file if it hasn't been loaded yet,
// then parses environment variables into a struct based on field tags.
// If loading fails, an appropriate error will be returned.
// Once a configuration type is successfully loaded, subsequent calls for the same
// type will return the cached version.
//
// Example:
//
//	type DatabaseConfig struct {
//		Host     string `env:"DB_HOST" envDefault:"localhost"`
//		Port     int    `env:"DB_PORT" envDefault:"5432"`
//		Username string `env:"DB_USER,required"`
//		Password string `env:"DB_PASS,required"`
//	}
//
//	var dbConfig DatabaseConfig
//	err := config.Load(&dbConfig)
//	if err != nil {
//		// Handle error
//	}
func Load[T any](v *T) error {
	defaultEnvLoaded.Do(func() {
		// Ignore errors - the .env file might not exist and that's ok
		_ = godotenv.Load()
	})
	if v == nil {
		return ErrNilPointer
	}

	typeName := getTypeName[T]()

	globalCache.mu.RLock()
	if cached, ok := globalCache.values[typeName]; ok {
		*v = cached.(T)
		globalCache.mu.RUnlock()
		return nil
	}
	globalCache.mu.RUnlock()

	globalCache.mu.Lock()
	once, exists := globalCache.onces[typeName]
	if !exists {
		once = new(sync.Once)
		globalCache.onces[typeName] = once
	}
	globalCache.mu.Unlock()

	var err error

	// Use sync.Once to ensure the config is parsed only once per type
	once.Do(func() {
		if parseErr := env.Parse(v); parseErr != nil {
			err = errors.Join(ErrParsingConfig, parseErr)
			return
		}

		globalCache.mu.Lock()
		globalCache.values[typeName] = *v // Store a copy to avoid external modifications
		globalCache.mu.Unlock()
	})

	if err != nil {
		return err
	}

	// Ensure the value is loaded from cache for concurrent requests
	globalCache.mu.RLock()
	if cached, ok := globalCache.values[typeName]; ok {
		*v = cached.(T)
		globalCache.mu.RUnlock()
		return nil
	}
	globalCache.mu.RUnlock()

	return ErrConfigNotLoaded
}

// MustLoad works like Load but panics if configuration loading fails.
// This is useful for configurations that are required for the application to start.
//
// Example:
//
//	var dbConfig DatabaseConfig
//	config.MustLoad(&dbConfig)
func MustLoad[T any](v *T) {
	if err := Load(v); err != nil {
		panic(fmt.Sprintf("Failed to load required configuration: %v", err))
	}
}

// getTypeName returns a string identifier for the generic type T
func getTypeName[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		// Handle interface types
		return fmt.Sprintf("%T", *new(T))
	}
	return t.String()
}
