package email

import "errors"

var (
	ErrFailedToSendEmail = errors.New("mailer.errors.failed_to_send_email")
	ErrInvalidConfig     = errors.New("mailer.errors.invalid_config")
)
