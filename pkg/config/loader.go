package config

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"reflect"
	"sync"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv" // For manual .env file loading
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

	// defaultEnvLoaded tracks if the default .env file has been loaded
	defaultEnvLoaded sync.Once
)

// LoadEnv loads environment variables from one or more .env files.
// If no paths are provided, it attempts to load the default .env file
// from the current directory.
//
// Files are loaded in the order provided. Variables in files loaded later
// take precedence over variables in files loaded earlier.
//
// Example:
//
//	// Load from custom .env file
//	err := config.LoadEnv("./config/.env")
//	if err != nil {
//	    log.Fatalf("Error loading env from custom path: %v", err)
//	}
//
//	// Load from multiple .env files
//	err := config.LoadEnv("./config/.env.base", "./config/.env.local")
func LoadEnv(filenames ...string) error {
	// Reset singleton mechanisms to allow reloading configurations
	globalCache.mu.Lock()
	globalCache.values = make(map[string]any)
	globalCache.onces = make(map[string]*sync.Once)
	defaultEnvLoaded = sync.Once{}
	globalCache.mu.Unlock()

	if len(filenames) == 0 {
		// If no paths specified, attempt to load from default .env
		return godotenv.Load()
	}

	// First load all environment variables from all files
	var envMap = make(map[string]string)
	for _, filename := range filenames {
		fileEnv, err := godotenv.Read(filename)
		if err != nil {
			return err
		}

		// Merge with precedence to later files
		maps.Copy(envMap, fileEnv)
	}

	// Now set all the environment variables
	for key, value := range envMap {
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}

	return nil
}

// MustLoadEnv works like LoadEnv but panics if loading fails.
// This is useful when environment variables are essential for the application to start.
//
// Example:
//
//	// Load from custom .env file, panic on failure
//	config.MustLoadEnv("./config/.env")
func MustLoadEnv(filenames ...string) {
	if err := LoadEnv(filenames...); err != nil {
		panic(fmt.Sprintf("Failed to load environment file(s): %v", err))
	}
}

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
// To load from a custom path, use LoadEnv() before calling Load().
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
//	// Optional: Load from custom path
//	config.LoadEnv("./config/.env")
//
//	var dbConfig DatabaseConfig
//	err := config.Load(&dbConfig)
//	if err != nil {
//		// Handle error
//	}
func Load[T any](v *T) error {
	// Try to load the default .env file once if no custom file has been explicitly loaded
	defaultEnvLoaded.Do(func() {
		// Ignore errors - the .env file might not exist and that's ok
		_ = godotenv.Load()
	})
	if v == nil {
		return ErrNilPointer
	}

	typeName := getTypeName[T]()

	// Try to retrieve from cache first with a read lock
	globalCache.mu.RLock()
	if cached, ok := globalCache.values[typeName]; ok {
		*v = cached.(T)
		globalCache.mu.RUnlock()
		return nil
	}
	globalCache.mu.RUnlock()

	// Get or create the once instance for this type
	globalCache.mu.Lock()
	once, exists := globalCache.onces[typeName]
	if !exists {
		once = new(sync.Once)
		globalCache.onces[typeName] = once
	}
	globalCache.mu.Unlock()

	// Error to be captured from the sync.Once execution
	var err error

	// Use sync.Once to ensure the config is parsed only once
	once.Do(func() {
		// Parse environment variables into the provided instance
		if parseErr := env.Parse(v); parseErr != nil {
			err = errors.Join(ErrParsingConfig, parseErr)
			return
		}

		// Store the successfully parsed config in the cache
		globalCache.mu.Lock()
		globalCache.values[typeName] = *v // Store a copy
		globalCache.mu.Unlock()
	})

	if err != nil {
		return err
	}

	// If we didn't hit the once.Do or there was no error,
	// ensure the value is loaded from cache
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

// The following functions are provided primarily for testing purposes and
// should generally not be used in production code.

// ResetCache clears all cached configuration instances.
// This is primarily useful in testing scenarios where
// environment variables change between test cases.
func ResetCache() {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	// Clear all cached values and once instances
	globalCache.values = make(map[string]any)
	globalCache.onces = make(map[string]*sync.Once)

	// Reset the default env loaded state
	defaultEnvLoaded = sync.Once{}
}

// ForceReloadConfig forces a reload of the specified configuration type,
// ignoring any previously cached instances. This is useful in testing
// when environment variables have changed.
func ForceReloadConfig[T any](v *T) error {
	if v == nil {
		return ErrNilPointer
	}

	typeName := getTypeName[T]()

	// Reset the once for this type
	globalCache.mu.Lock()
	globalCache.onces[typeName] = new(sync.Once)
	delete(globalCache.values, typeName)
	globalCache.mu.Unlock()

	// Now load the configuration
	return Load(v)
}

// IsConfigLoaded returns true if the specified configuration type has
// already been loaded and cached.
func IsConfigLoaded[T any]() bool {
	typeName := getTypeName[T]()

	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()

	_, exists := globalCache.values[typeName]
	return exists
}
