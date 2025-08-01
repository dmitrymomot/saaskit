package pg

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrFailedToOpenDBConnection = errors.New("failed to open db connection")
	ErrEmptyConnectionString    = errors.New("empty postgres connection string, use DATABASE_URL env var")
	ErrHealthcheckFailed        = errors.New("healthcheck failed, connection is not available")
	ErrFailedToParseDBConfig    = errors.New("failed to parse db config")
	ErrFailedToApplyMigrations  = errors.New("failed to apply migrations")
	ErrMigrationsDirNotFound    = errors.New("migrations directory not found")
	ErrMigrationPathNotProvided = errors.New("migration path not provided")
)

// IsNotFoundError detects pgx.ErrNoRows for consistent "not found" handling across queries.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, pgx.ErrNoRows)
}

// IsTxClosedError detects attempts to use closed transactions, helping debug concurrency issues.
func IsTxClosedError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, pgx.ErrTxClosed)
}

// IsDuplicateKeyError detects PostgreSQL unique constraint violations (SQLSTATE 23505).
// Common in SaaS applications for email uniqueness, username conflicts, etc.
func IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// IsForeignKeyViolationError detects referential integrity violations (SQLSTATE 23503).
// Occurs when trying to insert/update records that reference non-existent foreign keys.
func IsForeignKeyViolationError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
