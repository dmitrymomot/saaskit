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
	EncryptionKey string `env:"TOTP_ENCRYPTION_KEY,required"`
}

// LoadConfig loads the TOTP configuration from environment variables.
// It returns a Config object and an error if required environment variables are missing or invalid.
// The function uses sync.Once to ensure configuration is loaded only once per process.
func LoadConfig() (Config, error) {
	configLoadFunc := func() (Config, error) {
		var cfg Config
		if err := env.Parse(&cfg); err != nil {
			return Config{}, err
		}
		// Validate required fields beyond env tag validation
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
