package email

import "errors"

var (
	ErrFailedToSendEmail = errors.New("failed to send email")
	ErrInvalidConfig     = errors.New("invalid email configuration")
)
