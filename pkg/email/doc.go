// Package email provides a provider-agnostic interface for sending transactional emails
// with built-in support for Postmark and type-safe email templates using templ.
//
// The package follows SaasKit principles:
//   - Zero-config defaults with explicit configuration
//   - Provider abstraction for easy vendor switching
//   - Type-safe templates with templ integration
//   - Modern error handling with sentinel errors
//   - Performance optimization with minimal allocations
//
// # Architecture
//
// The package is built around the EmailSender interface, allowing different email
// providers to be swapped without changing application code. Currently supported:
//   - PostmarkClient for production email delivery with tracking
//   - DevSender for local development (saves emails to disk)
//
// All implementations validate email parameters before sending and provide
// consistent error handling across providers.
//
// # Usage
//
// Basic email sending with Postmark:
//
//	import "github.com/dmitrymomot/saaskit/pkg/email"
//
//	cfg := email.Config{
//	    PostmarkServerToken:  "your-server-token",
//	    PostmarkAccountToken: "your-account-token",
//	    SenderEmail:          "noreply@example.com",
//	    SupportEmail:         "support@example.com",
//	}
//
//	client, err := email.NewPostmarkClient(cfg)
//	if err != nil {
//	    // Handle configuration error
//	}
//
//	err = client.SendEmail(ctx, email.SendEmailParams{
//	    SendTo:   "user@example.com",
//	    Subject:  "Welcome!",
//	    BodyHTML: htmlContent,
//	    Tag:      "welcome", // optional, for analytics
//	})
//
// Development mode saves emails locally:
//
//	devSender := email.NewDevSender("./email-output")
//	err := devSender.SendEmail(ctx, params)
//	// Creates timestamped HTML and JSON files in ./email-output/
//
// Using type-safe templates:
//
//	import "github.com/dmitrymomot/saaskit/pkg/email/templates"
//
//	html, err := templates.Render(ctx, myTemplComponent)
//	if err != nil {
//	    return err
//	}
//
//	err = client.SendEmail(ctx, email.SendEmailParams{
//	    SendTo:   "user@example.com",
//	    Subject:  "Subject",
//	    BodyHTML: html,
//	})
//
// # Configuration
//
// The Config struct requires all fields for production use:
//   - PostmarkServerToken: API token for sending emails
//   - PostmarkAccountToken: Account token for administrative operations
//   - SenderEmail: From address for all emails
//   - SupportEmail: Reply-to address for user responses
//
// Use MustNewPostmarkClient for initialization that panics on invalid config,
// following the framework pattern of failing fast during startup.
//
// # Error Handling
//
// The package provides sentinel errors for common failure scenarios:
//   - ErrInvalidConfig: Configuration validation failed
//   - ErrInvalidParams: Email parameters validation failed
//   - ErrFailedToSendEmail: Email delivery failed
//
// All errors can be checked using errors.Is() for programmatic handling:
//
//	if errors.Is(err, email.ErrInvalidParams) {
//	    // Handle validation error
//	}
//
// # Templates
//
// The templates subpackage provides reusable email components built with templ:
//   - Layout, Header, Footer for consistent structure
//   - PrimaryButton, SuccessButton, DangerButton for CTAs
//   - Text, TextWarning, TextSecondary for content styling
//   - OTP component for authentication codes
//
// Templates are pre-styled for email clients with inline CSS and responsive design.
//
// # Performance Considerations
//
// The package is optimized for minimal allocations:
//   - Template rendering uses strings.Builder for zero-allocation string construction
//   - Email validation uses a pre-compiled regex
//   - DevSender sanitizes filenames efficiently with a single regex pass
//
// For high-volume sending, consider integrating with a queue system for
// asynchronous processing.
package email
