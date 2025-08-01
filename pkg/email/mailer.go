package email

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// EmailSender abstracts email providers to enable testing and provider switching.
// Implementation should handle retries and provider-specific error mapping.
type EmailSender interface {
	SendEmail(ctx context.Context, params SendEmailParams) error
}

// SendEmailParams contains email content and metadata.
// Tag is used for analytics and campaign tracking in email providers.
// BodyHTML is expected to be pre-rendered from templates for performance.
type SendEmailParams struct {
	SendTo   string `json:"send_to"`
	Subject  string `json:"subject"`
	BodyHTML string `json:"body_html"`
	Tag      string `json:"tag,omitempty"`
}

// emailRegex is a simple regex for validating email addresses.
// This covers most common cases while avoiding overly complex validation.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Validate checks that all required fields are present and valid.
// Returns ErrInvalidParams with details about which field failed validation.
func (p SendEmailParams) Validate() error {
	if strings.TrimSpace(p.SendTo) == "" {
		return fmt.Errorf("%w: SendTo is required", ErrInvalidParams)
	}
	if !emailRegex.MatchString(p.SendTo) {
		return fmt.Errorf("%w: SendTo must be a valid email address", ErrInvalidParams)
	}
	if strings.TrimSpace(p.Subject) == "" {
		return fmt.Errorf("%w: Subject is required", ErrInvalidParams)
	}
	if strings.TrimSpace(p.BodyHTML) == "" {
		return fmt.Errorf("%w: BodyHTML is required", ErrInvalidParams)
	}
	return nil
}
