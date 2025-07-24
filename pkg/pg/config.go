package pg

import "time"

type Config struct {
	ConnectionString  string        `env:"PG_CONN_URL,required"`                   // ConnectionString is the connection string to the database.
	MaxOpenConns      int32         `env:"PG_MAX_OPEN_CONNS" envDefault:"10"`      // MaxOpenConns is the maximum number of open connections to the database.
	MaxIdleConns      int32         `env:"PG_MAX_IDLE_CONNS" envDefault:"5"`       // MaxIdleConns is the maximum number of idle connections to the database.
	HealthCheckPeriod time.Duration `env:"PG_HEALTHCHECK_PERIOD" envDefault:"1m"`  // HealthCheckPeriod is the period between health checks.
	MaxConnIdleTime   time.Duration `env:"PG_MAX_CONN_IDLE_TIME" envDefault:"10m"` // MaxConnIdleTime is the maximum amount of time a connection may be idle to be reused.
	MaxConnLifetime   time.Duration `env:"PG_MAX_CONN_LIFETIME" envDefault:"30m"`  // MaxConnLifetime is the maximum amount of time a connection may be reused.

	RetryAttempts int           `env:"PG_RETRY_ATTEMPTS" envDefault:"3"`  // RetryAttempts is the number of retry attempts to connect to the database.
	RetryInterval time.Duration `env:"PG_RETRY_INTERVAL" envDefault:"5s"` // RetryInterval is the interval between retry attempts. It should be in the format "5s" for 5 seconds.

	MigrationsPath  string `env:"PG_MIGRATIONS_PATH" envDefault:"internal/db/migrations"` // MigrationsPath is the path to the migrations directory.
	MigrationsTable string `env:"PG_MIGRATIONS_TABLE" envDefault:"schema_migrations"`     // MigrationsTable is the name of the table used to store the migration version.
}
