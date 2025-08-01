package email_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/dmitrymomot/saaskit/pkg/email"
)

// MockEmailSender is a mock implementation of EmailSender for testing
type MockEmailSender struct {
	mock.Mock
}

func (m *MockEmailSender) SendEmail(ctx context.Context, params email.SendEmailParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func TestSendEmailParams_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  email.SendEmailParams
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid params",
			params: email.SendEmailParams{
				SendTo:   "user@example.com",
				Subject:  "Test Subject",
				BodyHTML: "<p>Test body</p>",
				Tag:      "test",
			},
			wantErr: false,
		},
		{
			name: "valid params without tag",
			params: email.SendEmailParams{
				SendTo:   "user@example.com",
				Subject:  "Test Subject",
				BodyHTML: "<p>Test body</p>",
			},
			wantErr: false,
		},
		{
			name: "empty SendTo",
			params: email.SendEmailParams{
				SendTo:   "",
				Subject:  "Test Subject",
				BodyHTML: "<p>Test body</p>",
			},
			wantErr: true,
			errMsg:  "SendTo is required",
		},
		{
			name: "whitespace only SendTo",
			params: email.SendEmailParams{
				SendTo:   "   ",
				Subject:  "Test Subject",
				BodyHTML: "<p>Test body</p>",
			},
			wantErr: true,
			errMsg:  "SendTo is required",
		},
		{
			name: "invalid email format",
			params: email.SendEmailParams{
				SendTo:   "invalid-email",
				Subject:  "Test Subject",
				BodyHTML: "<p>Test body</p>",
			},
			wantErr: true,
			errMsg:  "SendTo must be a valid email address",
		},
		{
			name: "invalid email missing domain",
			params: email.SendEmailParams{
				SendTo:   "user@",
				Subject:  "Test Subject",
				BodyHTML: "<p>Test body</p>",
			},
			wantErr: true,
			errMsg:  "SendTo must be a valid email address",
		},
		{
			name: "invalid email missing local part",
			params: email.SendEmailParams{
				SendTo:   "@example.com",
				Subject:  "Test Subject",
				BodyHTML: "<p>Test body</p>",
			},
			wantErr: true,
			errMsg:  "SendTo must be a valid email address",
		},
		{
			name: "empty Subject",
			params: email.SendEmailParams{
				SendTo:   "user@example.com",
				Subject:  "",
				BodyHTML: "<p>Test body</p>",
			},
			wantErr: true,
			errMsg:  "Subject is required",
		},
		{
			name: "whitespace only Subject",
			params: email.SendEmailParams{
				SendTo:   "user@example.com",
				Subject:  "   ",
				BodyHTML: "<p>Test body</p>",
			},
			wantErr: true,
			errMsg:  "Subject is required",
		},
		{
			name: "empty BodyHTML",
			params: email.SendEmailParams{
				SendTo:   "user@example.com",
				Subject:  "Test Subject",
				BodyHTML: "",
			},
			wantErr: true,
			errMsg:  "BodyHTML is required",
		},
		{
			name: "whitespace only BodyHTML",
			params: email.SendEmailParams{
				SendTo:   "user@example.com",
				Subject:  "Test Subject",
				BodyHTML: "   ",
			},
			wantErr: true,
			errMsg:  "BodyHTML is required",
		},
		{
			name: "complex valid email",
			params: email.SendEmailParams{
				SendTo:   "test.user+tag@sub.example.com",
				Subject:  "Test Subject",
				BodyHTML: "<p>Test body</p>",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.params.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, email.ErrInvalidParams)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDevSender_SendEmail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("successful send with tag", func(t *testing.T) {
		t.Parallel()

		// Create temp directory for test
		tempDir := t.TempDir()
		sender := email.NewDevSender(tempDir)

		params := email.SendEmailParams{
			SendTo:   "user@example.com",
			Subject:  "Test Email",
			BodyHTML: "<p>Test content</p>",
			Tag:      "welcome",
		}

		err := sender.SendEmail(ctx, params)
		assert.NoError(t, err)

		// Verify files were created
		files, err := os.ReadDir(tempDir)
		assert.NoError(t, err)
		assert.Len(t, files, 2) // HTML + JSON files

		// Find the files
		var htmlFile, jsonFile string
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".html") {
				htmlFile = filepath.Join(tempDir, file.Name())
			}
			if strings.HasSuffix(file.Name(), ".json") {
				jsonFile = filepath.Join(tempDir, file.Name())
			}
		}

		assert.NotEmpty(t, htmlFile)
		assert.NotEmpty(t, jsonFile)

		// Verify HTML content
		htmlContent, err := os.ReadFile(htmlFile)
		assert.NoError(t, err)
		assert.Equal(t, "<p>Test content</p>", string(htmlContent))

		// Verify JSON metadata
		jsonContent, err := os.ReadFile(jsonFile)
		assert.NoError(t, err)
		var metadata map[string]interface{}
		err = json.Unmarshal(jsonContent, &metadata)
		assert.NoError(t, err)
		assert.Equal(t, "user@example.com", metadata["send_to"])
		assert.Equal(t, "Test Email", metadata["subject"])
		assert.Equal(t, "welcome", metadata["tag"])
		assert.NotEmpty(t, metadata["timestamp"])
	})

	t.Run("successful send without tag uses subject", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		sender := email.NewDevSender(tempDir)

		params := email.SendEmailParams{
			SendTo:   "user@example.com",
			Subject:  "Password Reset",
			BodyHTML: "<p>Reset your password</p>",
			// No tag - should use subject
		}

		err := sender.SendEmail(ctx, params)
		assert.NoError(t, err)

		// Verify files were created with subject-based filename
		files, err := os.ReadDir(tempDir)
		assert.NoError(t, err)
		assert.Len(t, files, 2)

		// Check that filename contains sanitized subject
		found := false
		for _, file := range files {
			if strings.Contains(file.Name(), "password_reset") {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected filename to contain sanitized subject")
	})

	t.Run("validation error", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		sender := email.NewDevSender(tempDir)

		params := email.SendEmailParams{
			SendTo:   "", // Invalid
			Subject:  "Test Email",
			BodyHTML: "<p>Test content</p>",
		}

		err := sender.SendEmail(ctx, params)
		assert.Error(t, err)
		assert.ErrorIs(t, err, email.ErrInvalidParams)

		// Verify no files were created
		files, err := os.ReadDir(tempDir)
		assert.NoError(t, err)
		assert.Len(t, files, 0)
	})

	t.Run("directory creation error simulation", func(t *testing.T) {
		t.Parallel()

		// Use an invalid path that will cause MkdirAll to fail
		// On Unix systems, we can't create directories with null bytes
		invalidDir := "/dev/null/cannot-create-here"
		sender := email.NewDevSender(invalidDir)

		params := email.SendEmailParams{
			SendTo:   "user@example.com",
			Subject:  "Test Email",
			BodyHTML: "<p>Test content</p>",
		}

		err := sender.SendEmail(ctx, params)
		assert.Error(t, err)
		assert.ErrorIs(t, err, email.ErrFailedToSendEmail)
		assert.Contains(t, err.Error(), "failed to create directory")
	})

	t.Run("edge case - unicode in content", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		sender := email.NewDevSender(tempDir)

		params := email.SendEmailParams{
			SendTo:   "user@example.com",
			Subject:  "Unicode Test 🚀",
			BodyHTML: "<p>Test with unicode: 你好世界 🌍</p>",
			Tag:      "unicode-test",
		}

		err := sender.SendEmail(ctx, params)
		assert.NoError(t, err)

		// Verify files were created and content is preserved
		files, err := os.ReadDir(tempDir)
		assert.NoError(t, err)
		assert.Len(t, files, 2)

		// Find and verify HTML content
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".html") {
				content, err := os.ReadFile(filepath.Join(tempDir, file.Name()))
				assert.NoError(t, err)
				assert.Contains(t, string(content), "你好世界 🌍")
				break
			}
		}
	})

	t.Run("edge case - very long email content", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		sender := email.NewDevSender(tempDir)

		// Create long content
		longContent := "<p>" + strings.Repeat("Very long email content. ", 1000) + "</p>"

		params := email.SendEmailParams{
			SendTo:   "user@example.com",
			Subject:  "Long Content Test",
			BodyHTML: longContent,
			Tag:      "long-content",
		}

		err := sender.SendEmail(ctx, params)
		assert.NoError(t, err)

		// Verify files were created
		files, err := os.ReadDir(tempDir)
		assert.NoError(t, err)
		assert.Len(t, files, 2)
	})
}

func TestDevSender_SanitizeFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal string",
			input:    "Hello World",
			expected: "hello_world",
		},
		{
			name:     "with special characters",
			input:    "Test@Email#Subject!",
			expected: "testemailsubject",
		},
		{
			name:     "with multiple spaces",
			input:    "Multiple   Spaces   Here",
			expected: "multiple___spaces___here",
		},
		{
			name:     "empty string fallback to subject",
			input:    "",
			expected: "test_subject",
		},
		{
			name:     "only special characters fallback to subject",
			input:    "!@#$%^&*()",
			expected: "email",
		},
		{
			name:     "very long string truncated",
			input:    strings.Repeat("a", 150),
			expected: strings.Repeat("a", 100), // Truncated to 100 chars
		},
		{
			name:     "allowed characters preserved",
			input:    "test-file_name.backup",
			expected: "test-file_name.backup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// We need to test this indirectly by checking filenames created by DevSender
			tempDir := t.TempDir()
			sender := email.NewDevSender(tempDir)

			params := email.SendEmailParams{
				SendTo:   "user@example.com",
				Subject:  "Test Subject", // Use consistent subject
				BodyHTML: "<p>Test content</p>",
				Tag:      tt.input, // Use the test input as tag
			}

			err := sender.SendEmail(context.Background(), params)
			assert.NoError(t, err)

			// Check that filename contains the sanitized version
			files, err := os.ReadDir(tempDir)
			assert.NoError(t, err)
			assert.Len(t, files, 2)

			// Find HTML file and check if it contains expected sanitized string
			for _, file := range files {
				if strings.HasSuffix(file.Name(), ".html") {
					// Filename format: YYYY_MM_DD_HHMMSS_identifier.html
					// We need to extract everything after the timestamp
					// Split by underscore and remove timestamp parts (first 4: YYYY_MM_DD_HHMMSS)
					parts := strings.Split(file.Name(), "_")
					if len(parts) >= 5 { // YYYY_MM_DD_HHMMSS_identifier.html
						// Join all parts after the timestamp
						identifierPart := strings.Join(parts[4:], "_")
						identifierPart = strings.TrimSuffix(identifierPart, ".html")
						assert.Equal(t, tt.expected, identifierPart)
					} else {
						// Fallback: check if the expected string is contained in the filename
						assert.Contains(t, file.Name(), tt.expected)
					}
					break
				}
			}
		})
	}
}

func TestMustNewPostmarkClient(t *testing.T) {
	t.Parallel()

	t.Run("valid config does not panic", func(t *testing.T) {
		t.Parallel()

		cfg := email.Config{
			PostmarkServerToken:  "test-server-token",
			PostmarkAccountToken: "test-account-token",
			SenderEmail:          "sender@example.com",
			SupportEmail:         "support@example.com",
		}

		assert.NotPanics(t, func() {
			client := email.MustNewPostmarkClient(cfg)
			assert.NotNil(t, client)
		})
	})

	t.Run("invalid config panics", func(t *testing.T) {
		t.Parallel()

		cfg := email.Config{
			// Missing required fields
			PostmarkServerToken: "test-token",
			// Missing other required fields
		}

		assert.Panics(t, func() {
			email.MustNewPostmarkClient(cfg)
		})
	})
}

func TestEmailSender_Interface(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("mock sender successful send", func(t *testing.T) {
		t.Parallel()

		mockSender := new(MockEmailSender)
		params := email.SendEmailParams{
			SendTo:   "user@example.com",
			Subject:  "Test Email",
			BodyHTML: "<p>Test content</p>",
			Tag:      "test",
		}

		mockSender.On("SendEmail", ctx, params).Return(nil)

		err := mockSender.SendEmail(ctx, params)
		assert.NoError(t, err)

		mockSender.AssertExpectations(t)
	})

	t.Run("mock sender failed send", func(t *testing.T) {
		t.Parallel()

		mockSender := new(MockEmailSender)
		params := email.SendEmailParams{
			SendTo:   "user@example.com",
			Subject:  "Test Email",
			BodyHTML: "<p>Test content</p>",
		}

		mockSender.On("SendEmail", ctx, params).Return(email.ErrFailedToSendEmail)

		err := mockSender.SendEmail(ctx, params)
		assert.Error(t, err)
		assert.ErrorIs(t, err, email.ErrFailedToSendEmail)

		mockSender.AssertExpectations(t)
	})
}
