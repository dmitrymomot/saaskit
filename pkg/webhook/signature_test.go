package webhook_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/webhook"
)

func TestSignPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		secret  string
		payload []byte
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid signature",
			secret:  "webhook_secret_123",
			payload: []byte(`{"event":"user.created","id":"123"}`),
			wantErr: false,
		},
		{
			name:    "empty secret",
			secret:  "",
			payload: []byte(`{"event":"user.created"}`),
			wantErr: true,
			errMsg:  "secret is required",
		},
		{
			name:    "empty payload",
			secret:  "secret",
			payload: []byte{},
			wantErr: true,
			errMsg:  "payload cannot be empty",
		},
		{
			name:    "nil payload",
			secret:  "secret",
			payload: nil,
			wantErr: true,
			errMsg:  "payload cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			headers, err := webhook.SignPayload(tt.secret, tt.payload)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, headers.Signature)
			assert.NotZero(t, headers.Timestamp)
			assert.NotEmpty(t, headers.ID)

			// Verify signature format (should be hex encoded)
			_, err = hex.DecodeString(headers.Signature)
			assert.NoError(t, err, "signature should be hex encoded")

			// Verify timestamp is recent
			age := time.Since(time.Unix(headers.Timestamp, 0))
			assert.Less(t, age, time.Second, "timestamp should be recent")
		})
	}
}

func TestVerifySignature(t *testing.T) {
	t.Parallel()

	secret := "test_secret"
	payload := []byte(`{"test":"data"}`)

	// Create a valid signature
	validHeaders, err := webhook.SignPayload(secret, payload)
	require.NoError(t, err)

	// Create expired signature
	expiredHeaders := webhook.SignatureHeaders{
		Signature: validHeaders.Signature,
		Timestamp: time.Now().Add(-2 * time.Hour).Unix(),
		ID:        validHeaders.ID,
	}

	// Create future signature
	futureHeaders := webhook.SignatureHeaders{
		Signature: validHeaders.Signature,
		Timestamp: time.Now().Add(2 * time.Hour).Unix(),
		ID:        validHeaders.ID,
	}

	// Create invalid signature
	invalidHeaders := webhook.SignatureHeaders{
		Signature: "invalid_signature",
		Timestamp: validHeaders.Timestamp,
		ID:        validHeaders.ID,
	}

	tests := []struct {
		name    string
		secret  string
		payload []byte
		headers webhook.SignatureHeaders
		maxAge  time.Duration
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid signature",
			secret:  secret,
			payload: payload,
			headers: validHeaders,
			maxAge:  5 * time.Minute,
			wantErr: false,
		},
		{
			name:    "valid signature no age check",
			secret:  secret,
			payload: payload,
			headers: validHeaders,
			maxAge:  0, // No age check
			wantErr: false,
		},
		{
			name:    "expired signature",
			secret:  secret,
			payload: payload,
			headers: expiredHeaders,
			maxAge:  time.Hour,
			wantErr: true,
			errMsg:  "signature timestamp too old",
		},
		{
			name:    "future signature",
			secret:  secret,
			payload: payload,
			headers: futureHeaders,
			maxAge:  time.Hour,
			wantErr: true,
			errMsg:  "signature timestamp is in the future",
		},
		{
			name:    "invalid signature",
			secret:  secret,
			payload: payload,
			headers: invalidHeaders,
			maxAge:  5 * time.Minute,
			wantErr: true,
			errMsg:  "signature mismatch",
		},
		{
			name:    "wrong secret",
			secret:  "wrong_secret",
			payload: payload,
			headers: validHeaders,
			maxAge:  5 * time.Minute,
			wantErr: true,
			errMsg:  "signature mismatch",
		},
		{
			name:    "empty secret",
			secret:  "",
			payload: payload,
			headers: validHeaders,
			maxAge:  5 * time.Minute,
			wantErr: true,
			errMsg:  "secret is required",
		},
		{
			name:    "empty payload",
			secret:  secret,
			payload: []byte{},
			headers: validHeaders,
			maxAge:  5 * time.Minute,
			wantErr: true,
			errMsg:  "payload cannot be empty",
		},
		{
			name:    "missing signature",
			secret:  secret,
			payload: payload,
			headers: webhook.SignatureHeaders{
				Timestamp: validHeaders.Timestamp,
				ID:        validHeaders.ID,
			},
			maxAge:  5 * time.Minute,
			wantErr: true,
			errMsg:  "signature is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := webhook.VerifySignature(tt.secret, tt.payload, tt.headers, tt.maxAge)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExtractSignatureHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		headers map[string]string
		want    webhook.SignatureHeaders
		wantErr bool
		errMsg  string
	}{
		{
			name: "standard headers",
			headers: map[string]string{
				"X-Webhook-Signature": "sig123",
				"X-Webhook-Timestamp": "1234567890",
				"X-Webhook-ID":        "id123",
			},
			want: webhook.SignatureHeaders{
				Signature: "sig123",
				Timestamp: 1234567890,
				ID:        "id123",
			},
			wantErr: false,
		},
		{
			name: "lowercase headers",
			headers: map[string]string{
				"x-webhook-signature": "sig123",
				"x-webhook-timestamp": "1234567890",
				"x-webhook-id":        "id123",
			},
			want: webhook.SignatureHeaders{
				Signature: "sig123",
				Timestamp: 1234567890,
				ID:        "id123",
			},
			wantErr: false,
		},
		{
			name: "uppercase headers",
			headers: map[string]string{
				"X-WEBHOOK-SIGNATURE": "sig123",
				"X-WEBHOOK-TIMESTAMP": "1234567890",
				"X-WEBHOOK-ID":        "id123",
			},
			want: webhook.SignatureHeaders{
				Signature: "sig123",
				Timestamp: 1234567890,
				ID:        "id123",
			},
			wantErr: false,
		},
		{
			name: "missing ID is ok",
			headers: map[string]string{
				"X-Webhook-Signature": "sig123",
				"X-Webhook-Timestamp": "1234567890",
			},
			want: webhook.SignatureHeaders{
				Signature: "sig123",
				Timestamp: 1234567890,
				ID:        "",
			},
			wantErr: false,
		},
		{
			name: "missing signature",
			headers: map[string]string{
				"X-Webhook-Timestamp": "1234567890",
				"X-Webhook-ID":        "id123",
			},
			wantErr: true,
			errMsg:  "missing required signature headers",
		},
		{
			name: "missing timestamp",
			headers: map[string]string{
				"X-Webhook-Signature": "sig123",
				"X-Webhook-ID":        "id123",
			},
			wantErr: true,
			errMsg:  "missing required signature headers",
		},
		{
			name: "invalid timestamp format",
			headers: map[string]string{
				"X-Webhook-Signature": "sig123",
				"X-Webhook-Timestamp": "not-a-number",
				"X-Webhook-ID":        "id123",
			},
			wantErr: true,
			errMsg:  "invalid timestamp format",
		},
		{
			name:    "empty headers",
			headers: map[string]string{},
			wantErr: true,
			errMsg:  "missing required signature headers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := webhook.ExtractSignatureHeaders(tt.headers)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSignatureConsistency(t *testing.T) {
	t.Parallel()

	secret := "test_webhook_secret"
	payload := []byte(`{"user_id":"123","action":"login"}`)

	// Sign the payload
	headers, err := webhook.SignPayload(secret, payload)
	require.NoError(t, err)

	// Manually recreate the signature to verify algorithm
	signaturePayload := fmt.Sprintf("%d.%s", headers.Timestamp, payload)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(signaturePayload))
	expectedSig := hex.EncodeToString(h.Sum(nil))

	assert.Equal(t, expectedSig, headers.Signature, "signature should match manual calculation")

	// Verify the signature
	err = webhook.VerifySignature(secret, payload, headers, 5*time.Minute)
	assert.NoError(t, err, "signature should verify successfully")
}

func TestTimingAttackResistance(t *testing.T) {
	t.Parallel()

	secret := "secret"
	payload := []byte("test")
	headers, err := webhook.SignPayload(secret, payload)
	require.NoError(t, err)

	// Create signatures with increasing differences
	validSig := headers.Signature

	// Same length, one char different
	invalidSig1 := validSig[:len(validSig)-1] + "X"

	// Same length, completely different
	invalidSig2 := ""
	for range validSig {
		invalidSig2 += "0"
	}

	// Test that verification uses constant-time comparison
	// We can't easily test timing, but we can ensure it fails correctly
	headers.Signature = invalidSig1
	err1 := webhook.VerifySignature(secret, payload, headers, 0)
	require.Error(t, err1)

	headers.Signature = invalidSig2
	err2 := webhook.VerifySignature(secret, payload, headers, 0)
	require.Error(t, err2)

	// Both should fail with same error
	assert.Equal(t, err1.Error(), err2.Error())
}

func BenchmarkSignPayload(b *testing.B) {
	secret := "benchmark_secret"
	payload := []byte(`{"event":"test","data":{"id":"123","name":"test","timestamp":1234567890}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := webhook.SignPayload(secret, payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVerifySignature(b *testing.B) {
	secret := "benchmark_secret"
	payload := []byte(`{"event":"test","data":{"id":"123","name":"test","timestamp":1234567890}}`)

	headers, err := webhook.SignPayload(secret, payload)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := webhook.VerifySignature(secret, payload, headers, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExtractSignatureHeaders(b *testing.B) {
	headers := map[string]string{
		"X-Webhook-Signature": "abc123def456",
		"X-Webhook-Timestamp": strconv.FormatInt(time.Now().Unix(), 10),
		"X-Webhook-ID":        "test-id-123",
		"Other-Header":        "ignored",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := webhook.ExtractSignatureHeaders(headers)
		if err != nil {
			b.Fatal(err)
		}
	}
}
