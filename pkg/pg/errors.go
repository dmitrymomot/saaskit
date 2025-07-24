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

// IsNotFoundError checks if the given error is a "not found" error.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, pgx.ErrNoRows)
}

// IsTxClosedError checks if the given error is a "transaction closed" error.
func IsTxClosedError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, pgx.ErrTxClosed)
}

// IsDuplicateKeyError checks if the error is a duplicate key error.
func IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// IsForeignKeyViolationError checks if the error is a foreign key violation error.
func IsForeignKeyViolationError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
