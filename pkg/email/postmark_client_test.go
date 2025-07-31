package email_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/email"
)

func TestNewPostmarkClient_ValidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config email.Config
	}{
		{
			name: "valid tokens",
			config: email.Config{
				PostmarkServerToken:  "test-server-token",
				PostmarkAccountToken: "test-account-token",
				SenderEmail:          "sender@example.com",
				SupportEmail:         "support@example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := email.NewPostmarkClient(tt.config)
			require.NoError(t, err)
			assert.NotNil(t, client)
		})
	}
}

func TestNewPostmarkClient_InvalidConfig(t *testing.T) {
	t.Parallel()

	t.Run("empty server token", func(t *testing.T) {
		t.Parallel()

		config := email.Config{
			PostmarkServerToken:  "",
			PostmarkAccountToken: "test-account-token",
			SenderEmail:          "sender@example.com",
			SupportEmail:         "support@example.com",
		}

		client, err := email.NewPostmarkClient(config)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.ErrorIs(t, err, email.ErrInvalidConfig)
		assert.Contains(t, err.Error(), "PostmarkServerToken is required")
	})

	t.Run("empty account token", func(t *testing.T) {
		t.Parallel()

		config := email.Config{
			PostmarkServerToken:  "test-server-token",
			PostmarkAccountToken: "",
			SenderEmail:          "sender@example.com",
			SupportEmail:         "support@example.com",
		}

		client, err := email.NewPostmarkClient(config)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.ErrorIs(t, err, email.ErrInvalidConfig)
		assert.Contains(t, err.Error(), "PostmarkAccountToken is required")
	})

	t.Run("missing sender email", func(t *testing.T) {
		t.Parallel()

		config := email.Config{
			PostmarkServerToken:  "test-server-token",
			PostmarkAccountToken: "test-account-token",
			SenderEmail:          "",
			SupportEmail:         "support@example.com",
		}

		client, err := email.NewPostmarkClient(config)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.ErrorIs(t, err, email.ErrInvalidConfig)
		assert.Contains(t, err.Error(), "SenderEmail is required")
	})

	t.Run("invalid sender email format", func(t *testing.T) {
		t.Parallel()

		config := email.Config{
			PostmarkServerToken:  "test-server-token",
			PostmarkAccountToken: "test-account-token",
			SenderEmail:          "invalid-email",
			SupportEmail:         "support@example.com",
		}

		client, err := email.NewPostmarkClient(config)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.ErrorIs(t, err, email.ErrInvalidConfig)
		assert.Contains(t, err.Error(), "SenderEmail must be a valid email address")
	})

	t.Run("missing support email", func(t *testing.T) {
		t.Parallel()

		config := email.Config{
			PostmarkServerToken:  "test-server-token",
			PostmarkAccountToken: "test-account-token",
			SenderEmail:          "sender@example.com",
			SupportEmail:         "",
		}

		client, err := email.NewPostmarkClient(config)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.ErrorIs(t, err, email.ErrInvalidConfig)
		assert.Contains(t, err.Error(), "SupportEmail is required")
	})

	t.Run("invalid support email format", func(t *testing.T) {
		t.Parallel()

		config := email.Config{
			PostmarkServerToken:  "test-server-token",
			PostmarkAccountToken: "test-account-token",
			SenderEmail:          "sender@example.com",
			SupportEmail:         "@invalid.com",
		}

		client, err := email.NewPostmarkClient(config)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.ErrorIs(t, err, email.ErrInvalidConfig)
		assert.Contains(t, err.Error(), "SupportEmail must be a valid email address")
	})
}

func TestPostmarkClient_SendEmail_ValidationError(t *testing.T) {
	t.Parallel()

	// Create a real postmark client to test validation
	cfg := email.Config{
		PostmarkServerToken:  "test-token",
		PostmarkAccountToken: "test-token",
		SenderEmail:          "sender@example.com",
		SupportEmail:         "support@example.com",
	}

	client, err := email.NewPostmarkClient(cfg)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("invalid params rejected", func(t *testing.T) {
		t.Parallel()

		params := email.SendEmailParams{
			SendTo:   "", // Invalid - empty
			Subject:  "Test Email",
			BodyHTML: "<p>Test content</p>",
		}

		err := client.SendEmail(ctx, params)
		assert.Error(t, err)
		assert.ErrorIs(t, err, email.ErrInvalidParams)
		assert.Contains(t, err.Error(), "SendTo is required")
	})

	t.Run("invalid email format rejected", func(t *testing.T) {
		t.Parallel()

		params := email.SendEmailParams{
			SendTo:   "invalid-email",
			Subject:  "Test Email",
			BodyHTML: "<p>Test content</p>",
		}

		err := client.SendEmail(ctx, params)
		assert.Error(t, err)
		assert.ErrorIs(t, err, email.ErrInvalidParams)
		assert.Contains(t, err.Error(), "SendTo must be a valid email address")
	})

	t.Run("empty subject rejected", func(t *testing.T) {
		t.Parallel()

		params := email.SendEmailParams{
			SendTo:   "user@example.com",
			Subject:  "",
			BodyHTML: "<p>Test content</p>",
		}

		err := client.SendEmail(ctx, params)
		assert.Error(t, err)
		assert.ErrorIs(t, err, email.ErrInvalidParams)
		assert.Contains(t, err.Error(), "Subject is required")
	})

	t.Run("empty body rejected", func(t *testing.T) {
		t.Parallel()

		params := email.SendEmailParams{
			SendTo:   "user@example.com",
			Subject:  "Test Email",
			BodyHTML: "",
		}

		err := client.SendEmail(ctx, params)
		assert.Error(t, err)
		assert.ErrorIs(t, err, email.ErrInvalidParams)
		assert.Contains(t, err.Error(), "BodyHTML is required")
	})
}
