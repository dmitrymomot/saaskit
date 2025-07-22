package totp

import (
	"sync"

	"github.com/caarlos0/env/v11"
	_ "github.com/joho/godotenv/autoload" // Load .env file automatically
)

var (
	cfg  Config
	once sync.Once
)

type Config struct {
	EncryptionKey string `env:"TOTP_ENCRYPTION_KEY,required"` // Encryption key for the TOTP secrets
}

// LoadConfig loads the configuration from the yaml file located at the given path.
// It returns a Config object representing the configuration and an error if any.
// The function panics if the configuration file is not found or if an error occurs while reading the file.
// The function also panics if the configuration file is found but it is not a valid yaml file.
func LoadConfig() (Config, error) {
	configLoadFunc := func() (Config, error) {
		var cfg Config
		if err := env.Parse(&cfg); err != nil {
			return Config{}, err
		}
		// Additional validation
		if cfg.EncryptionKey == "" {
			return Config{}, ErrEncryptionKeyNotSet
		}
		return cfg, nil
	}

	var err error
	once.Do(func() {
		cfg, err = configLoadFunc()
	})
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}
