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

// Migrate applies database schema migrations using goose with pgx integration.
// Handles the complex pgx->database/sql conversion required since goose doesn't natively support pgx.
func Migrate(ctx context.Context, pool *pgxpool.Pool, cfg Config, log logger) error {
	if cfg.MigrationsPath == "" {
		return errors.Join(ErrFailedToApplyMigrations, ErrMigrationPathNotProvided)
	}

	if _, err := os.Stat(cfg.MigrationsPath); err != nil {
		if os.IsNotExist(err) {
			return errors.Join(ErrMigrationsDirNotFound, err)
		}
		return errors.Join(ErrFailedToApplyMigrations, err)
	}

	// Bridge pgx connection pool to database/sql interface required by goose.
	// This creates a wrapper that shares the underlying connections but provides
	// the standard library interface that goose migration tool expects.
	db := stdlib.OpenDBFromPool(pool)
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.ErrorContext(ctx, "Failed to close database connection", "error", err)
		}
	}(db)

	// Route goose migration logs through application logger instead of stdout.
	goose.SetLogger(newSlogAdapter(log))
	goose.SetTableName(cfg.MigrationsTable)

	if err := goose.SetDialect("postgres"); err != nil {
		return errors.Join(ErrFailedToApplyMigrations, err)
	}

	if err := goose.UpContext(ctx, db, cfg.MigrationsPath); err != nil {
		return errors.Join(ErrFailedToApplyMigrations, err)
	}

	return nil
}

// migrateSlogAdapter bridges goose's Printf-style logging to structured logging.
// Maps goose's Fatalf to ErrorContext and Printf to InfoContext for consistency.
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
