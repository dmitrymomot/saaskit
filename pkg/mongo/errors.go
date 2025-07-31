package mongo

import "errors"

// Domain-specific errors enable proper error handling in application code.
// These wrap underlying MongoDB driver errors while providing stable error
// types that application code can check with errors.Is() for retry logic,
// fallback behavior, or appropriate user-facing error messages.
var (
	ErrFailedToConnectToMongo = errors.New("failed to connect to mongo")
	ErrHealthcheckFailed      = errors.New("mongo healthcheck failed")
)
