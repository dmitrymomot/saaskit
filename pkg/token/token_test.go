package token_test

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/token"
)

type testPayload struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestGenerateAndParseToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		payload testPayload
		secret  string
		wantErr bool
	}{
		{
			name:    "valid token",
			payload: testPayload{ID: 1, Name: "test"},
			secret:  "secret123",
			wantErr: false,
		},
		{
			name:    "empty secret",
			payload: testPayload{ID: 1, Name: "test"},
			secret:  "",
			wantErr: false,
		},
		{
			name:    "empty payload",
			payload: testPayload{},
			secret:  "secret123",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Generate token
			tokenStr, err := token.GenerateToken(tt.payload, tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Verify token format
			parts := strings.Split(tokenStr, ".")
			if len(parts) != 2 {
				t.Errorf("GenerateToken() invalid token format, got %v parts, want 2", len(parts))
				return
			}

			// Parse token
			got, err := token.ParseToken[testPayload](tokenStr, tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Compare payload
			if got.ID != tt.payload.ID || got.Name != tt.payload.Name {
				t.Errorf("ParseToken() got = %v, want %v", got, tt.payload)
			}
		})
	}
}

func TestParseToken_InvalidCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		token     string
		secret    string
		wantError error
	}{
		{
			name:      "invalid token format",
			token:     "invalid",
			secret:    "secret123",
			wantError: token.ErrInvalidToken,
		},
		{
			name:      "invalid base64 payload",
			token:     "!@#$.sig", // Invalid base64 characters
			secret:    "secret123",
			wantError: base64.CorruptInputError(0),
		},
		{
			name:      "invalid signature",
			token:     "eyJpZCI6MSwiZXhwIjoxNTE2MjM5MDIyfQ.invalid",
			secret:    "secret123",
			wantError: token.ErrSignatureInvalid,
		},
		{
			name:      "tampered payload",
			token:     "eyJpZCI6OSwiZXhwIjoxNTE2MjM5MDIyfQ.sig",
			secret:    "secret123",
			wantError: token.ErrSignatureInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := token.ParseToken[testPayload](tt.token, tt.secret)
			if err != tt.wantError {
				t.Errorf("ParseToken() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestTokenSignatureVerification(t *testing.T) {
	t.Parallel()
	payload := testPayload{ID: 1, Name: "test"}
	secret := "secret123"

	// Generate token with original secret
	tokenStr, err := token.GenerateToken(payload, secret)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Try to parse with wrong secret
	_, err = token.ParseToken[testPayload](tokenStr, "wrongsecret")
	if err != token.ErrSignatureInvalid {
		t.Errorf("ParseToken() with wrong secret error = %v, want %v", err, token.ErrSignatureInvalid)
	}

	// Tamper with the payload
	parts := strings.Split(tokenStr, ".")
	tamperedPayload := base64.RawURLEncoding.EncodeToString([]byte(`{"id":2,"name":"hacked"}`))
	tamperedToken := tamperedPayload + "." + parts[1]

	// Try to parse tampered token
	_, err = token.ParseToken[testPayload](tamperedToken, secret)
	if err != token.ErrSignatureInvalid {
		t.Errorf("ParseToken() with tampered payload error = %v, want %v", err, token.ErrSignatureInvalid)
	}
}
