package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/mrz1836/postmark"
)

type postmarkClient struct {
	client *postmark.Client
	config Config
}

// NewPostmarkClient creates a Postmark-backed email sender.
// Both tokens are required for runtime operation - this enforces
// explicit configuration rather than silent failures in production.
func NewPostmarkClient(cfg Config) (EmailSender, error) {
	if cfg.PostmarkServerToken == "" {
		return nil, fmt.Errorf("%w: PostmarkServerToken is required", ErrInvalidConfig)
	}
	if cfg.PostmarkAccountToken == "" {
		return nil, fmt.Errorf("%w: PostmarkAccountToken is required", ErrInvalidConfig)
	}
	if cfg.SenderEmail == "" {
		return nil, fmt.Errorf("%w: SenderEmail is required", ErrInvalidConfig)
	}
	if !emailRegex.MatchString(cfg.SenderEmail) {
		return nil, fmt.Errorf("%w: SenderEmail must be a valid email address", ErrInvalidConfig)
	}
	if cfg.SupportEmail == "" {
		return nil, fmt.Errorf("%w: SupportEmail is required", ErrInvalidConfig)
	}
	if !emailRegex.MatchString(cfg.SupportEmail) {
		return nil, fmt.Errorf("%w: SupportEmail must be a valid email address", ErrInvalidConfig)
	}

	return &postmarkClient{
		client: postmark.NewClient(cfg.PostmarkServerToken, cfg.PostmarkAccountToken),
		config: cfg,
	}, nil
}

// MustNewPostmarkClient creates a Postmark client that panics on invalid config.
// Follows framework pattern of failing fast during initialization rather than
// allowing broken services to start.
func MustNewPostmarkClient(cfg Config) EmailSender {
	client, err := NewPostmarkClient(cfg)
	if err != nil {
		panic(err)
	}
	return client
}

// SendEmail implements EmailSender using Postmark's transactional API.
// Tracking is enabled by default for analytics - opens and HTML link clicks only
// to avoid privacy issues with plain text. Reply-To is set to support email
// to ensure customer responses reach the right team.
func (c *postmarkClient) SendEmail(ctx context.Context, params SendEmailParams) error {
	if err := params.Validate(); err != nil {
		return err
	}

	resp, err := c.client.SendEmail(ctx, postmark.Email{
		From:       c.config.SenderEmail,
		ReplyTo:    c.config.SupportEmail,
		To:         params.SendTo,
		Subject:    params.Subject,
		Tag:        params.Tag,
		HTMLBody:   params.BodyHTML,
		TrackOpens: true,
		TrackLinks: "HtmlOnly",
	})
	if err != nil {
		return errors.Join(ErrFailedToSendEmail, err)
	}
	if resp.ErrorCode > 0 {
		return errors.Join(
			ErrFailedToSendEmail,
			fmt.Errorf("postmark error: %d - %s", resp.ErrorCode, resp.Message),
		)
	}
	return nil
}
