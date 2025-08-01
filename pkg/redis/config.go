package redis

import "time"

type Config struct {
	ConnectionURL  string        `env:"REDIS_URL,required" envDefault:"redis://localhost:6379/0"`
	RetryAttempts  int           `env:"REDIS_RETRY_ATTEMPTS" envDefault:"3"`
	RetryInterval  time.Duration `env:"REDIS_RETRY_INTERVAL" envDefault:"5s"`
	ConnectTimeout time.Duration `env:"REDIS_CONNECT_TIMEOUT" envDefault:"30s"`
	ScanBatchSize  int           `env:"REDIS_SCAN_BATCH_SIZE" envDefault:"1000"`
}
