package email_test

import (
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
		{
			name: "valid server and account tokens",
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

	tests := []struct {
		name        string
		config      email.Config
		expectError bool
	}{
		{
			name: "empty server token",
			config: email.Config{
				PostmarkServerToken: "",
				SenderEmail:         "sender@example.com",
				SupportEmail:        "support@example.com",
			},
			expectError: true,
		},
		{
			name: "whitespace only server token",
			config: email.Config{
				PostmarkServerToken: "   ",
				SenderEmail:         "sender@example.com",
				SupportEmail:        "support@example.com",
			},
			expectError: true,
		},
		{
			name: "missing required emails",
			config: email.Config{
				PostmarkServerToken: "test-token",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := email.NewPostmarkClient(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestConfig_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  email.Config
		isValid bool
	}{
		{
			name: "valid config with both tokens",
			config: email.Config{
				PostmarkServerToken:  "server-token-123",
				PostmarkAccountToken: "account-token-456",
				SenderEmail:          "sender@example.com",
				SupportEmail:         "support@example.com",
			},
			isValid: true,
		},
		{
			name: "valid config with both tokens",
			config: email.Config{
				PostmarkServerToken:  "server-token-123",
				PostmarkAccountToken: "account-token-456",
				SenderEmail:          "sender@example.com",
				SupportEmail:         "support@example.com",
			},
			isValid: true,
		},
		{
			name: "invalid config - empty server token",
			config: email.Config{
				PostmarkServerToken: "",
				SenderEmail:         "sender@example.com",
				SupportEmail:        "support@example.com",
			},
			isValid: false,
		},
		{
			name: "invalid config - missing required emails",
			config: email.Config{
				PostmarkServerToken: "server-token-123",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := email.NewPostmarkClient(tt.config)
			if tt.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
