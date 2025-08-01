package pg

import "time"

type Config struct {
	// PostgreSQL connection URL (postgres://user:pass@host:port/db)
	ConnectionString string `env:"PG_CONN_URL,required"`

	// Connection pool settings optimized for SaaS workloads.
	// Default 10 open connections handles typical web traffic without overwhelming the database.
	// Adjust based on your expected concurrent requests and database capacity.
	MaxOpenConns int32 `env:"PG_MAX_OPEN_CONNS" envDefault:"10"`

	// Minimum idle connections kept warm to reduce connection establishment overhead.
	// Default 5 provides good balance between resource usage and response time.
	MaxIdleConns int32 `env:"PG_MAX_IDLE_CONNS" envDefault:"5"`

	// Health check frequency to detect connection issues early.
	// 1 minute interval catches problems without excessive overhead.
	HealthCheckPeriod time.Duration `env:"PG_HEALTHCHECK_PERIOD" envDefault:"1m"`

	// Force connection refresh to prevent stale connections in load balancer environments.
	// 10 minutes prevents issues with connection poolers like PgBouncer.
	MaxConnIdleTime time.Duration `env:"PG_MAX_CONN_IDLE_TIME" envDefault:"10m"`

	// Total connection lifetime to handle database failovers and network changes.
	// 30 minutes balances connection stability with adaptability to infrastructure changes.
	MaxConnLifetime time.Duration `env:"PG_MAX_CONN_LIFETIME" envDefault:"30m"`

	// Retry configuration for handling transient network issues during startup.
	// 3 attempts with exponential backoff handles most temporary connection problems.
	RetryAttempts int           `env:"PG_RETRY_ATTEMPTS" envDefault:"3"`
	RetryInterval time.Duration `env:"PG_RETRY_INTERVAL" envDefault:"5s"`

	// Migration settings for database schema management.
	MigrationsPath  string `env:"PG_MIGRATIONS_PATH" envDefault:"internal/db/migrations"`
	MigrationsTable string `env:"PG_MIGRATIONS_TABLE" envDefault:"schema_migrations"`
}
