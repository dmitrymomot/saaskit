package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Migrate migrates the database to the latest version using the provided connection pool.
// If the database is already up to date, it will return nil. If the database is not up to date,
// it will apply all available migrations and return nil. If an error occurs, it will return the error.
func Migrate(ctx context.Context, pool *pgxpool.Pool, cfg Config, log logger) error {
	if cfg.MigrationsPath == "" {
		return errors.Join(ErrFailedToApplyMigrations, ErrMigrationPathNotProvided)
	}

	// Check if the provided directory exists
	if _, err := os.Stat(cfg.MigrationsPath); err != nil {
		if os.IsNotExist(err) {
			return errors.Join(ErrMigrationsDirNotFound, err)
		}
		return errors.Join(ErrFailedToApplyMigrations, err)
	}

	// Convert pgx pool to database/sql DB since goose expects it
	db := stdlib.OpenDBFromPool(pool)
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.ErrorContext(ctx, "Failed to close database connection", "error", err)
		}
	}(db)

	// Set the custom logger for goose
	goose.SetLogger(newSlogAdapter(log))
	goose.SetTableName(cfg.MigrationsTable)

	// Set the dialect
	if err := goose.SetDialect("postgres"); err != nil {
		return errors.Join(ErrFailedToApplyMigrations, err)
	}

	// Run migrations with context
	if err := goose.UpContext(ctx, db, cfg.MigrationsPath); err != nil {
		return errors.Join(ErrFailedToApplyMigrations, err)
	}

	return nil
}

// migrateSlogAdapter is an adapter that converts goose's logger interface to use slog.
type migrateSlogAdapter struct {
	log logger
}

func newSlogAdapter(log logger) goose.Logger {
	return &migrateSlogAdapter{
		log: log,
	}
}

func (a *migrateSlogAdapter) Fatalf(format string, v ...any) {
	a.log.ErrorContext(context.Background(), fmt.Sprintf(format, v...))
}

func (a *migrateSlogAdapter) Printf(format string, v ...any) {
	a.log.InfoContext(context.Background(), fmt.Sprintf(format, v...))
}
