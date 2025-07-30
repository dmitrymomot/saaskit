package email

import "context"

// EmailSender represents an interface for sending emails.
type EmailSender interface {
	SendEmail(ctx context.Context, params SendEmailParams) error
}

// SendEmailParams represents the parameters for sending an email.
type SendEmailParams struct {
	SendTo   string `json:"send_to"`
	Subject  string `json:"subject"`
	BodyHTML string `json:"body_html"`
	Tag      string `json:"tag,omitempty"`
}
