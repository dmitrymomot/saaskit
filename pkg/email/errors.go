package email

import "errors"

// Domain errors for email operations.
// These are designed to be wrapped with internal errors using errors.Join()
// to provide both user-facing messages and detailed logging context.
var (
	ErrFailedToSendEmail = errors.New("failed to send email")
	ErrInvalidConfig     = errors.New("invalid email configuration")
	ErrInvalidParams     = errors.New("invalid email parameters")
)
