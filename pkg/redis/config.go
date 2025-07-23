package redis

import "time"

type Config struct {
	ConnectionURL  string        `env:"REDIS_URL,required" envDefault:"redis://localhost:6379/0"` // ConnectionURL is the URL of the database. It should be in the format "redis://:password@localhost:6379/0"
	RetryAttempts  int           `env:"REDIS_RETRY_ATTEMPTS" envDefault:"3"`                      // RetryAttempts is the number of retry attempts to connect to the database.
	RetryInterval  time.Duration `env:"REDIS_RETRY_INTERVAL" envDefault:"5s"`                     // RetryInterval is the interval between retry attempts. It should be in the format "5s" for 5
	ConnectTimeout time.Duration `env:"REDIS_CONNECT_TIMEOUT" envDefault:"30s"`                   // ConnectTimeout is the timeout for connecting to the database. It should be in the format "30s" for 30 seconds.
}
