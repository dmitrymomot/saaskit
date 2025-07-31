package token_test

import (
	"encoding/base64"
	"strings"
	"sync"
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
			wantError: token.ErrInvalidToken,
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

func TestLargePayloads(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{
			name:    "1KB payload",
			size:    1024,
			wantErr: false,
		},
		{
			name:    "10KB payload",
			size:    10 * 1024,
			wantErr: false,
		},
		{
			name:    "100KB payload",
			size:    100 * 1024,
			wantErr: false,
		},
		{
			name:    "1MB payload",
			size:    1024 * 1024,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create large payload
			largeData := make([]byte, tt.size)
			for i := range largeData {
				largeData[i] = byte(i % 256)
			}

			type largePayload struct {
				ID   int    `json:"id"`
				Data []byte `json:"data"`
			}

			payload := largePayload{ID: 1, Data: largeData}
			secret := "secret123"

			// Generate token
			tokenStr, err := token.GenerateToken(payload, secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Parse token
			parsed, err := token.ParseToken[largePayload](tokenStr, secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Verify data integrity
			if parsed.ID != payload.ID {
				t.Errorf("ParseToken() ID = %v, want %v", parsed.ID, payload.ID)
			}
			if len(parsed.Data) != len(payload.Data) {
				t.Errorf("ParseToken() data length = %v, want %v", len(parsed.Data), len(payload.Data))
			}
		})
	}
}

func TestConcurrentUsage(t *testing.T) {
	t.Parallel()
	const (
		goroutines = 100
		iterations = 100
	)

	payload := testPayload{ID: 1, Name: "concurrent"}
	secret := "secret123"

	// Test concurrent token generation
	t.Run("concurrent generation", func(t *testing.T) {
		t.Parallel()
		var wg sync.WaitGroup
		errors := make(chan error, goroutines*iterations)

		for i := range goroutines {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := range iterations {
					p := testPayload{ID: id*1000 + j, Name: "test"}
					_, err := token.GenerateToken(p, secret)
					if err != nil {
						errors <- err
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("concurrent GenerateToken() error = %v", err)
		}
	})

	// Test concurrent token parsing
	t.Run("concurrent parsing", func(t *testing.T) {
		t.Parallel()
		// Generate a token to parse
		tokenStr, err := token.GenerateToken(payload, secret)
		if err != nil {
			t.Fatalf("GenerateToken() error = %v", err)
		}

		var wg sync.WaitGroup
		errors := make(chan error, goroutines*iterations)

		for range goroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range iterations {
					_, err := token.ParseToken[testPayload](tokenStr, secret)
					if err != nil {
						errors <- err
					}
				}
			}()
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("concurrent ParseToken() error = %v", err)
		}
	})

	// Test mixed concurrent operations
	t.Run("mixed operations", func(t *testing.T) {
		t.Parallel()
		var wg sync.WaitGroup
		errors := make(chan error, goroutines*iterations*2)

		// Half goroutines generate tokens
		for i := range goroutines / 2 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := range iterations {
					p := testPayload{ID: id*1000 + j, Name: "generate"}
					tokenStr, err := token.GenerateToken(p, secret)
					if err != nil {
						errors <- err
						continue
					}
					// Immediately parse the generated token
					_, err = token.ParseToken[testPayload](tokenStr, secret)
					if err != nil {
						errors <- err
					}
				}
			}(i)
		}

		// Other half parse a shared token
		tokenStr, _ := token.GenerateToken(payload, secret)
		for range goroutines / 2 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range iterations {
					_, err := token.ParseToken[testPayload](tokenStr, secret)
					if err != nil {
						errors <- err
					}
				}
			}()
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("mixed operations error = %v", err)
		}
	})
}
