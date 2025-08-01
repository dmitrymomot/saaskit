package mongo

import "time"

// Config provides MongoDB connection parameters optimized for SaaS applications.
// All fields are populated from environment variables to simplify deployment.
type Config struct {
	ConnectionURL string `env:"MONGODB_URL,required"`

	// Connection timeouts should be aggressive enough to fail fast but allow for
	// MongoDB Atlas cold starts which can take 5-8 seconds
	ConnectTimeout time.Duration `env:"MONGODB_CONNECT_TIMEOUT" envDefault:"10s"`

	// Pool size defaults work well for typical SaaS traffic:
	// - Max 100: handles burst traffic without overwhelming MongoDB
	// - Min 1: maintains warm connection for immediate availability
	// - 5min idle: balances connection overhead vs availability
	MaxPoolSize     uint64        `env:"MONGODB_MAX_POOL_SIZE" envDefault:"100"`
	MinPoolSize     uint64        `env:"MONGODB_MIN_POOL_SIZE" envDefault:"1"`
	MaxConnIdleTime time.Duration `env:"MONGODB_MAX_CONN_IDLE_TIME" envDefault:"300s"`

	// Retry writes/reads are essential for MongoDB Atlas replica sets
	// where primary elections can cause transient failures
	RetryWrites bool `env:"MONGODB_RETRY_WRITES" envDefault:"true"`
	RetryReads  bool `env:"MONGODB_RETRY_READS" envDefault:"true"`

	// Connection retry parameters handle Atlas cold starts and network hiccups
	// 3 attempts with 5s intervals covers most transient failures without
	// excessive startup delays (max 15s additional delay)
	RetryAttempts int           `env:"MONGODB_RETRY_ATTEMPTS" envDefault:"3"`
	RetryInterval time.Duration `env:"MONGODB_RETRY_INTERVAL" envDefault:"5s"`
}
