package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// SignatureHeaders contains the standard webhook signature headers.
// Follows security patterns used by Stripe, GitHub, and other major webhook providers.
type SignatureHeaders struct {
	Signature string
	Timestamp int64
	ID        string
}

// Headers returns the signature headers as a map for easy HTTP header setting.
// Uses standard header names that are widely recognized.
func (s SignatureHeaders) Headers() map[string]string {
	return map[string]string{
		"X-Webhook-Signature": s.Signature,
		"X-Webhook-Timestamp": strconv.FormatInt(s.Timestamp, 10),
		"X-Webhook-ID":        s.ID,
	}
}

// SignPayload creates an HMAC-SHA256 signature for webhook authentication.
// Timestamp binding prevents replay attacks, following industry best practices.
// Signature format: HMAC-SHA256(secret, timestamp + "." + payload)
func SignPayload(secret string, payload []byte) (SignatureHeaders, error) {
	if secret == "" {
		return SignatureHeaders{}, fmt.Errorf("%w: secret is required", ErrInvalidConfiguration)
	}
	if len(payload) == 0 {
		return SignatureHeaders{}, fmt.Errorf("%w: payload cannot be empty", ErrInvalidPayload)
	}

	timestamp := time.Now().Unix()
	id := uuid.New().String()

	// Bind signature to timestamp to prevent replay attacks
	// Format matches Stripe's webhook signature scheme for compatibility
	signaturePayload := fmt.Sprintf("%d.%s", timestamp, payload)

	// Use HMAC-SHA256 for cryptographic security without excessive overhead
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(signaturePayload))
	signature := hex.EncodeToString(h.Sum(nil))

	return SignatureHeaders{
		Signature: signature,
		Timestamp: timestamp,
		ID:        id,
	}, nil
}

// VerifySignature validates webhook authenticity and prevents replay attacks.
// Uses constant-time comparison and timestamp validation for security.
func VerifySignature(secret string, payload []byte, headers SignatureHeaders, maxAge time.Duration) error {
	if secret == "" {
		return fmt.Errorf("%w: secret is required", ErrInvalidConfiguration)
	}
	if len(payload) == 0 {
		return fmt.Errorf("%w: payload cannot be empty", ErrInvalidPayload)
	}
	if headers.Signature == "" {
		return fmt.Errorf("%w: signature is missing", ErrInvalidConfiguration)
	}

	// Validate timestamp window to prevent replay attacks
	if maxAge > 0 {
		age := time.Since(time.Unix(headers.Timestamp, 0))
		if age > maxAge {
			return fmt.Errorf("%w: signature timestamp too old: %v", ErrInvalidConfiguration, age)
		}
		// Allow reasonable clock skew but reject far-future timestamps
		if age < -1*time.Minute {
			return fmt.Errorf("%w: signature timestamp is in the future", ErrInvalidConfiguration)
		}
	}

	// Recreate signature using the same algorithm as SignPayload
	signaturePayload := fmt.Sprintf("%d.%s", headers.Timestamp, payload)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(signaturePayload))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	// Use constant-time comparison to prevent timing-based attacks
	if !hmac.Equal([]byte(expectedSignature), []byte(headers.Signature)) {
		return fmt.Errorf("%w: signature mismatch", ErrInvalidConfiguration)
	}

	return nil
}

// ExtractSignatureHeaders extracts webhook signature data from HTTP headers.
// Handles case variations for maximum compatibility across HTTP implementations.
func ExtractSignatureHeaders(headers map[string]string) (SignatureHeaders, error) {
	var sig SignatureHeaders
	var err error

	// Handle common case variations since HTTP headers are case-insensitive
	signatureKeys := []string{"X-Webhook-Signature", "x-webhook-signature", "X-WEBHOOK-SIGNATURE"}
	timestampKeys := []string{"X-Webhook-Timestamp", "x-webhook-timestamp", "X-WEBHOOK-TIMESTAMP"}
	idKeys := []string{"X-Webhook-ID", "x-webhook-id", "X-WEBHOOK-ID", "X-Webhook-Id"}

	// Find signature header using case-insensitive search
	for _, key := range signatureKeys {
		if val, ok := headers[key]; ok {
			sig.Signature = val
			break
		}
	}

	// Extract and parse timestamp
	for _, key := range timestampKeys {
		if val, ok := headers[key]; ok {
			sig.Timestamp, err = strconv.ParseInt(val, 10, 64)
			if err != nil {
				return SignatureHeaders{}, fmt.Errorf("%w: invalid timestamp format", ErrInvalidConfiguration)
			}
			break
		}
	}

	// Extract webhook ID (optional but useful for idempotency)
	for _, key := range idKeys {
		if val, ok := headers[key]; ok {
			sig.ID = val
			break
		}
	}

	if sig.Signature == "" || sig.Timestamp == 0 {
		return SignatureHeaders{}, fmt.Errorf("%w: missing required signature headers", ErrInvalidConfiguration)
	}

	return sig, nil
}
