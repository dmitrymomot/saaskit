# Email

Provider-agnostic interface for sending transactional emails with built-in support for Postmark and type-safe email templates using templ.

## Features

- Provider abstraction with Postmark implementation and development mode
- Type-safe email templates with templ integration
- Development sender that saves emails to disk for testing
- Built-in validation for email parameters and configuration

## Installation

```go
import "github.com/dmitrymomot/saaskit/pkg/email"
```

## Usage

```go
// Production usage with Postmark
cfg := email.Config{
    PostmarkServerToken:  "your-server-token",
    PostmarkAccountToken: "your-account-token",
    SenderEmail:          "noreply@example.com",
    SupportEmail:         "support@example.com",
}

client, err := email.NewPostmarkClient(cfg)
if err != nil {
    log.Fatal(err)
}

// Send an email
err = client.SendEmail(ctx, email.SendEmailParams{
    SendTo:   "user@example.com",
    Subject:  "Welcome!",
    BodyHTML: "<h1>Welcome</h1><p>Thanks for signing up!</p>",
    Tag:      "welcome", // optional, for analytics
})
```

## Common Operations

### Development Mode

```go
// Save emails to disk instead of sending
devSender := email.NewDevSender("./email-output")

err := devSender.SendEmail(ctx, email.SendEmailParams{
    SendTo:   "test@example.com",
    Subject:  "Test Email",
    BodyHTML: htmlContent,
})
// Creates: ./email-output/2024_01_15_143022_test_email.html
// Creates: ./email-output/2024_01_15_143022_test_email.json
```

### Using Templates

```go
import "github.com/dmitrymomot/saaskit/pkg/email/templates"

// Render a templ component to HTML
html, err := templates.Render(ctx, myTemplComponent)
if err != nil {
    return err
}

// Send the rendered email
err = client.SendEmail(ctx, email.SendEmailParams{
    SendTo:   "user@example.com",
    Subject:  "Your OTP Code",
    BodyHTML: html,
    Tag:      "otp",
})
```

## Error Handling

```go
// Package errors:
var (
    ErrInvalidConfig     = errors.New("invalid email configuration")
    ErrInvalidParams     = errors.New("invalid email parameters")
    ErrFailedToSendEmail = errors.New("failed to send email")
)

// Usage:
if errors.Is(err, email.ErrInvalidParams) {
    // handle validation error
}
```

## Configuration

```go
// All fields required for production
config := email.Config{
    PostmarkServerToken:  os.Getenv("POSTMARK_SERVER_TOKEN"),
    PostmarkAccountToken: os.Getenv("POSTMARK_ACCOUNT_TOKEN"),
    SenderEmail:          os.Getenv("SENDER_EMAIL"),
    SupportEmail:         os.Getenv("SUPPORT_EMAIL"),
}

// Panic on invalid config (for initialization)
client := email.MustNewPostmarkClient(config)
```

## API Documentation

```bash
# Full API documentation
go doc github.com/dmitrymomot/saaskit/pkg/email

# Specific function or type
go doc github.com/dmitrymomot/saaskit/pkg/email.EmailSender
```

## Notes

- Email validation uses a simple regex that covers most common cases
- DevSender sanitizes filenames and limits them to 100 characters
- Templates subpackage provides pre-styled email components (Layout, Buttons, Text styles, OTP)
