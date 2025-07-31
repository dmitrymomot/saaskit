package pg

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect establishes a PostgreSQL connection pool with retry logic for reliable SaaS startup.
// Uses exponential backoff to handle transient network issues without overwhelming the database.
func Connect(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	connConfig, err := pgxpool.ParseConfig(cfg.ConnectionString)
	if err != nil {
		return nil, errors.Join(ErrFailedToParseDBConfig, err)
	}
	connConfig.MaxConns = cfg.MaxOpenConns
	connConfig.MinConns = cfg.MaxIdleConns
	connConfig.HealthCheckPeriod = cfg.HealthCheckPeriod
	connConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	connConfig.MaxConnLifetime = cfg.MaxConnLifetime

	// Exponential backoff: attempt 1 waits RetryInterval, attempt 2 waits 2x, attempt 3 waits 3x.
	// This prevents thundering herd problems when multiple services restart simultaneously.
	for i := range cfg.RetryAttempts {
		conn, err := pgxpool.NewWithConfig(ctx, connConfig)
		if err != nil {
			time.Sleep(time.Duration(i+1) * cfg.RetryInterval)
			continue
		}

		// Verify connection with actual database ping to catch authentication and permission issues.
		if err := conn.Ping(ctx); err != nil {
			conn.Close()
			time.Sleep(time.Duration(i+1) * cfg.RetryInterval)
			continue
		}

		return conn, nil
	}

	return nil, ErrFailedToOpenDBConnection
}
