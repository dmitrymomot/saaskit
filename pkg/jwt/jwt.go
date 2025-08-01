package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JWT header constants required by RFC 7519
const (
	HeaderType      = "JWT"
	HeaderAlgorithm = "HS256" // HMAC-SHA256 chosen for security/performance balance
)

// Header represents the JWT header as defined in RFC 7515
type Header struct {
	Type      string `json:"typ"`
	Algorithm string `json:"alg"`
}

// StandardClaims represents the registered JWT claims defined in RFC 7519 Section 4.1.
// All fields use Unix timestamps for temporal claims to ensure consistent validation.
type StandardClaims struct {
	ID        string `json:"jti,omitempty"` // JWT ID - unique identifier for preventing token reuse
	Subject   string `json:"sub,omitempty"` // Subject - typically user ID or entity identifier
	Issuer    string `json:"iss,omitempty"` // Issuer - identifies who issued the token
	Audience  string `json:"aud,omitempty"` // Audience - intended recipient(s) of the token
	ExpiresAt int64  `json:"exp,omitempty"` // Expiration time - Unix timestamp when token expires
	NotBefore int64  `json:"nbf,omitempty"` // Not before - Unix timestamp when token becomes valid
	IssuedAt  int64  `json:"iat,omitempty"` // Issued at - Unix timestamp when token was created
}

// Valid validates the temporal claims against current time.
// Zero values are treated as unset (per RFC 7519) and are ignored during validation.
func (c StandardClaims) Valid() error {
	now := time.Now().Unix()

	if c.ExpiresAt > 0 && now > c.ExpiresAt {
		return ErrExpiredToken
	}

	if c.NotBefore > 0 && now < c.NotBefore {
		return ErrInvalidToken
	}

	return nil
}

// Service handles JWT token generation and validation using HMAC-SHA256.
// The signing key is kept in memory only and should be cryptographically secure.
type Service struct {
	signingKey []byte
}

// New creates a new JWT service with the provided signing key.
// The key should be at least 32 bytes for adequate security with HMAC-SHA256.
func New(signingKey []byte) (*Service, error) {
	if len(signingKey) == 0 {
		return nil, ErrMissingSigningKey
	}

	return &Service{
		signingKey: signingKey,
	}, nil
}

// NewFromString creates a new JWT service from a string signing key.
// Convenience wrapper around New() for string-based configuration.
func NewFromString(signingKey string) (*Service, error) {
	if signingKey == "" {
		return nil, ErrMissingSigningKey
	}

	return &Service{
		signingKey: []byte(signingKey),
	}, nil
}

// Generate creates a JWT token with the given claims.
// Accepts any JSON-serializable claims structure and returns a signed JWT string.
func (s *Service) Generate(claims any) (string, error) {
	if claims == nil {
		return "", ErrMissingClaims
	}

	header := Header{
		Type:      HeaderType,
		Algorithm: HeaderAlgorithm,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	// Build JWT payload: base64url(header).base64url(claims)
	headerEncoded := base64URLEncode(headerJSON)
	claimsEncoded := base64URLEncode(claimsJSON)
	payload := headerEncoded + "." + claimsEncoded

	signature := s.sign(payload)
	token := payload + "." + signature

	return token, nil
}

// Parse validates a JWT token and unmarshals its claims into the provided structure.
// Performs cryptographic verification, algorithm validation, and temporal claim checks.
func (s *Service) Parse(tokenString string, claims any) error {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return ErrInvalidToken
	}

	headerEncoded := parts[0]
	claimsEncoded := parts[1]
	signatureEncoded := parts[2]

	// Verify signature using constant-time comparison to prevent timing attacks
	payload := headerEncoded + "." + claimsEncoded
	expectedSignature := s.sign(payload)
	if subtle.ConstantTimeCompare([]byte(signatureEncoded), []byte(expectedSignature)) != 1 {
		return ErrInvalidSignature
	}

	headerJSON, err := base64URLDecode(headerEncoded)
	if err != nil {
		return fmt.Errorf("failed to decode header: %w", err)
	}

	var header Header
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return fmt.Errorf("failed to unmarshal header: %w", err)
	}

	// Reject tokens using unexpected algorithms to prevent algorithm confusion attacks
	if header.Algorithm != HeaderAlgorithm {
		return ErrUnexpectedSigningMethod
	}

	claimsJSON, err := base64URLDecode(claimsEncoded)
	if err != nil {
		return fmt.Errorf("failed to decode claims: %w", err)
	}

	if err := json.Unmarshal(claimsJSON, claims); err != nil {
		return fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	// Validate temporal claims if the type implements the Valid interface
	if validator, ok := claims.(interface{ Valid() error }); ok {
		if err := validator.Valid(); err != nil {
			return err
		}
	}

	return nil
}

// sign creates an HMAC-SHA256 signature for the given payload.
// Returns base64url-encoded signature as required by RFC 7515.
func (s *Service) sign(payload string) string {
	h := hmac.New(sha256.New, s.signingKey)
	h.Write([]byte(payload))
	return base64URLEncode(h.Sum(nil))
}

// base64URLEncode encodes data using base64url encoding without padding.
// Padding removal is required by RFC 7515 for JWT tokens.
func base64URLEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

// base64URLDecode decodes base64url-encoded data, restoring padding as needed.
// JWT tokens omit padding per RFC 7515, but Go's decoder requires it.
func base64URLDecode(s string) ([]byte, error) {
	switch len(s) % 4 {
	case 2:
		s += strings.Repeat("=", 2)
	case 3:
		s += strings.Repeat("=", 1)
	}

	return base64.URLEncoding.DecodeString(s)
}
