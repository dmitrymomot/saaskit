package pg

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect opens a new Postgres database connection with the provided configuration.
// It attempts to connect to the database multiple times based on the configured retry attempts.
// Returns the database connection and any error encountered.
func Connect(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	// Parse the connection string.
	connConfig, err := pgxpool.ParseConfig(cfg.ConnectionString)
	if err != nil {
		return nil, errors.Join(ErrFailedToParseDBConfig, err)
	}
	connConfig.MaxConns = cfg.MaxOpenConns
	connConfig.MinConns = cfg.MaxIdleConns
	connConfig.HealthCheckPeriod = cfg.HealthCheckPeriod
	connConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	connConfig.MaxConnLifetime = cfg.MaxConnLifetime

	// Retry connect to the database with exponential backoff.
	for i := range cfg.RetryAttempts {
		// Open a new connections pool.
		conn, err := pgxpool.NewWithConfig(ctx, connConfig)
		if err != nil {
			time.Sleep(time.Duration(i+1) * cfg.RetryInterval)
			continue
		}

		// Ping the database to check if the connection is available.
		if err := conn.Ping(ctx); err != nil {
			conn.Close()
			time.Sleep(time.Duration(i+1) * cfg.RetryInterval)
			continue
		}

		return conn, nil // Connection is available.
	}

	// Failed to open a connection.
	return nil, ErrFailedToOpenDBConnection
}
