package mongo

import "time"

// Config represents the configuration for the database.
type Config struct {
	ConnectionURL   string        `env:"MONGODB_URL,required"`                         // ConnectionURL is the URL of the database.
	ConnectTimeout  time.Duration `env:"MONGODB_CONNECT_TIMEOUT" envDefault:"10s"`     // ConnectTimeout is the timeout for connecting to the database.
	MaxPoolSize     uint64        `env:"MONGODB_MAX_POOL_SIZE" envDefault:"100"`       // MaxPoolSize is the maximum number of connections in the connection pool.
	MinPoolSize     uint64        `env:"MONGODB_MIN_POOL_SIZE" envDefault:"1"`         // MinPoolSize is the minimum number of connections in the connection pool.
	MaxConnIdleTime time.Duration `env:"MONGODB_MAX_CONN_IDLE_TIME" envDefault:"300s"` // MaxConnIdleTime is the maximum time that a connection can remain idle in the connection pool.
	RetryWrites     bool          `env:"MONGODB_RETRY_WRITES" envDefault:"true"`       // RetryWrites specifies whether to retry write operations.
	RetryReads      bool          `env:"MONGODB_RETRY_READS" envDefault:"true"`        // RetryReads specifies whether to retry read operations.
	RetryAttempts   int           `env:"MONGODB_RETRY_ATTEMPTS" envDefault:"3"`        // RetryAttempts is the number of retry attempts to connect to the database.
	RetryInterval   time.Duration `env:"MONGODB_RETRY_INTERVAL" envDefault:"5s"`       // RetryInterval is the interval between retry attempts. It should be in the format "5s" for 5 seconds.
}
