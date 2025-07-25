package email

import "context"

// EmailSender represents an interface for sending emails.
type EmailSender interface {
	SendEmail(ctx context.Context, params SendEmailParams) error
}

// SendEmailParams represents the parameters for sending an email.
type SendEmailParams struct {
	SendTo   string `json:"send_to"`       // Email address of the recipient
	Subject  string `json:"subject"`       // Subject of the email
	BodyHTML string `json:"body_html"`     // HTML body of the email
	Tag      string `json:"tag,omitempty"` // Optional
}
