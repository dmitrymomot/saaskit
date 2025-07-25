package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/mrz1836/postmark"
)

// postmarkClient implements the EmailSender interface.
type postmarkClient struct {
	client *postmark.Client
	config Config
}

// NewPostmarkClient creates a new instance of the mailer client
// with the provided server token and account token from the config (see mailer.Config).
// The client is used to send emails synchronously using the Postmark API.
// For asynchronous email sending, use the email enqueuer.
func NewPostmarkClient(cfg Config) (EmailSender, error) {
	if cfg.PostmarkServerToken == "" {
		return nil, fmt.Errorf("%w: PostmarkServerToken is required", ErrInvalidConfig)
	}
	if cfg.PostmarkAccountToken == "" {
		return nil, fmt.Errorf("%w: PostmarkAccountToken is required", ErrInvalidConfig)
	}

	return &postmarkClient{
		client: postmark.NewClient(cfg.PostmarkServerToken, cfg.PostmarkAccountToken),
		config: cfg,
	}, nil
}

// MustNewPostmarkClient creates a new instance of the mailer client
// with the provided server token and account token from the config (see mailer.Config).
// The client is used to send emails synchronously using the Postmark API.
// For asynchronous email sending, use the email enqueuer.
// Panics if the config cannot be loaded.
func MustNewPostmarkClient(cfg Config) EmailSender {
	client, err := NewPostmarkClient(cfg)
	if err != nil {
		panic(err)
	}
	return client
}

// SendEmail sends an email using the Postmark API with tracking enabled for opens and links.
// It uses the configured sender email as the "From" address and support email as "Reply-To".
// Returns an error if the send fails or if Postmark returns an error response.
func (c *postmarkClient) SendEmail(ctx context.Context, params SendEmailParams) error {
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
