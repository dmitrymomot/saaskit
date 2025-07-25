# mailer

Email sending package with type-safe templates and provider abstraction.

## Overview

This package provides email sending functionality for the SaaS application, featuring a provider-agnostic interface with Postmark implementation and type-safe email templates using templ. It's designed for synchronous email operations with built-in tracking capabilities.

## Internal Usage

This package is internal to the project and provides email sending capabilities for authentication flows, notifications, and transactional emails across the application.

## Features

- Provider-agnostic EmailSender interface for easy provider switching
- Postmark integration with automatic email tracking (opens and links)
- Type-safe email templates using templ components
- Comprehensive validation of configuration at client creation
- Support for HTML email with responsive design
- Reusable email components (buttons, headers, footers, OTP displays)

## Usage

### Basic Example

```go
import
    "github.com/dmitrymomot/saaskit/pkg/email"
    "github.com/dmitrymomot/saaskit/pkg/config"
)

// Load configuration from environment variables
cfg, err := config.Load[mailer.Config]()
if err != nil {
	// Handle error
}

// Create email client
client, err := mailer.NewPostmarkClient(cfg)
if err != nil {
    // Handle configuration error
}

// Send email
err = client.SendEmail(ctx, mailer.SendEmailParams{
    SendTo:   "user@example.com",
    Subject:  "Welcome to Our Service",
    BodyHTML: "<h1>Welcome!</h1><p>Thanks for signing up.</p>",
    Tag:      "welcome-email", // optional, for Postmark categorization
})
```

### Using Email Templates

```go
import (
    "github.com/dmitrymomot/saaskit/pkg/email"
    "github.com/dmitrymomot/saaskit/pkg/email/templates"
    "github.com/dmitrymomot/saaskit/pkg/email/templates/components"
)

// Build an email using components
emailTemplate := components.Layout(
	components.Header("Welcome to Our Platform"),
	components.Text("We're excited to have you join us."),
	components.PrimaryButton("Get Started", "https://example.com/start"),
	components.Footer(" 2025 Example Inc."),
)

// Render template to HTML
htmlBody, err := templates.Render(context.Background(), emailTemplate)
if err != nil {
	// Handle error
}

// Send email with the rendered template
err = client.SendEmail(context.Background(), mailer.SendEmailParams{
	SendTo:   "user@example.com",
	Subject:  "Welcome!",
	BodyHTML: htmlBody,
	Tag:      "onboarding",
})
```

### Error Handling

```go
err := client.SendEmail(ctx, params)
if err != nil {
    if errors.Is(err, mailer.ErrInvalidConfig) {
        // Configuration issue - check Postmark tokens
    } else if errors.Is(err, mailer.ErrFailedToSendEmail) {
        // Email delivery failed - check logs for Postmark error details
    }
}
```

## Best Practices

### Integration Guidelines

- Initialize the mailer client once at application startup and reuse it
- Use environment variables for all configuration values
- Always validate email addresses before sending
- Use meaningful tags for email categorization and analytics

### Project-Specific Considerations

- The package is designed for synchronous operations - integrate with taskqueue for async email sending
- Template components are optimized for email clients with inline styles
- All emails automatically include tracking for opens and link clicks
- Support email is set as Reply-To header for all outgoing emails

## API Reference

### Types

```go
type Config struct {
    PostmarkServerToken  string `env:"POSTMARK_SERVER_TOKEN"`
    PostmarkAccountToken string `env:"POSTMARK_ACCOUNT_TOKEN"`
    SenderEmail          string `env:"SENDER_EMAIL,required"`
    SupportEmail         string `env:"SUPPORT_EMAIL,required"`
}

type EmailSender interface {
    SendEmail(ctx context.Context, params SendEmailParams) error
}

type SendEmailParams struct {
    SendTo   string `json:"send_to"`
    Subject  string `json:"subject"`
    BodyHTML string `json:"body_html"`
    Tag      string `json:"tag,omitempty"`
}
```

### Functions

```go
func NewPostmarkClient(cfg Config) (EmailSender, error)
func MustNewPostmarkClient(cfg Config) EmailSender
```

### Template Functions

```go
// templates package
func Render(ctx context.Context, tpl templ.Component) (string, error)

// templates/components package
func Layout() templ.Component
func Header(title, subtitle string) templ.Component
func Text() templ.Component
func TextWarning() templ.Component
func TextSecondary() templ.Component
func Logo(logoURL, alt string) templ.Component
func Footer() templ.Component
func FooterLink(text, url string) templ.Component
func ButtonGroup() templ.Component
func PrimaryButton(text, url string) templ.Component
func SuccessButton(text, url string) templ.Component
func DangerButton(text, url string) templ.Component
func Link(text, url string) templ.Component
func OTP(otp string) templ.Component
```

### Error Types

```go
var ErrFailedToSendEmail = errors.New("mailer.errors.failed_to_send_email")
var ErrInvalidConfig = errors.New("mailer.errors.invalid_config")
```
