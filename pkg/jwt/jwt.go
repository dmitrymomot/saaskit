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

// Constants for JWT
const (
	// HeaderType is the type of the JWT header
	HeaderType = "JWT"
	// HeaderAlgorithm is the algorithm used for signing
	HeaderAlgorithm = "HS256"
)

// Header represents the JWT header
type Header struct {
	Type      string `json:"typ"`
	Algorithm string `json:"alg"`
}

// StandardClaims represents the standard JWT claims
type StandardClaims struct {
	// ID is a unique identifier for this token
	ID string `json:"jti,omitempty"`
	// Subject is the subject of the token
	Subject string `json:"sub,omitempty"`
	// Issuer is the issuer of the token
	Issuer string `json:"iss,omitempty"`
	// Audience is the audience of the token
	Audience string `json:"aud,omitempty"`
	// ExpiresAt is the time at which the token expires
	ExpiresAt int64 `json:"exp,omitempty"`
	// NotBefore is the time before which the token must not be accepted
	NotBefore int64 `json:"nbf,omitempty"`
	// IssuedAt is the time at which the token was issued
	IssuedAt int64 `json:"iat,omitempty"`
}

// Valid checks if the claims are valid
func (c StandardClaims) Valid() error {
	now := time.Now().Unix()

	// Check if the token is expired
	if c.ExpiresAt > 0 && now > c.ExpiresAt {
		return ErrExpiredToken
	}

	// Check if the token is not yet valid
	if c.NotBefore > 0 && now < c.NotBefore {
		return ErrInvalidToken
	}

	return nil
}

// Service is the JWT service
type Service struct {
	signingKey []byte
}

// New creates a new JWT service
func New(signingKey []byte) (*Service, error) {
	if len(signingKey) == 0 {
		return nil, ErrMissingSigningKey
	}

	return &Service{
		signingKey: signingKey,
	}, nil
}

// NewFromString creates a new JWT service from a string signing key
func NewFromString(signingKey string) (*Service, error) {
	if signingKey == "" {
		return nil, ErrMissingSigningKey
	}

	return &Service{
		signingKey: []byte(signingKey),
	}, nil
}

// Generate generates a JWT token with the given claims
func (s *Service) Generate(claims any) (string, error) {
	if claims == nil {
		return "", ErrMissingClaims
	}

	// Create the header
	header := Header{
		Type:      HeaderType,
		Algorithm: HeaderAlgorithm,
	}

	// Encode the header
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}

	// Encode the claims
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	// Create the payload (header.claims)
	headerEncoded := base64URLEncode(headerJSON)
	claimsEncoded := base64URLEncode(claimsJSON)
	payload := headerEncoded + "." + claimsEncoded

	// Sign the payload
	signature := s.sign(payload)

	// Create the token (payload.signature)
	token := payload + "." + signature

	return token, nil
}

// Parse parses a JWT token and returns the claims
func (s *Service) Parse(tokenString string, claims any) error {
	// Split the token
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return ErrInvalidToken
	}

	// Extract the parts
	headerEncoded := parts[0]
	claimsEncoded := parts[1]
	signatureEncoded := parts[2]

	// Verify the signature
	payload := headerEncoded + "." + claimsEncoded
	expectedSignature := s.sign(payload)
	if subtle.ConstantTimeCompare([]byte(signatureEncoded), []byte(expectedSignature)) != 1 {
		return ErrInvalidSignature
	}

	// Decode the header
	headerJSON, err := base64URLDecode(headerEncoded)
	if err != nil {
		return fmt.Errorf("failed to decode header: %w", err)
	}

	// Parse the header
	var header Header
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return fmt.Errorf("failed to unmarshal header: %w", err)
	}

	// Check the algorithm
	if header.Algorithm != HeaderAlgorithm {
		return ErrUnexpectedSigningMethod
	}

	// Decode the claims
	claimsJSON, err := base64URLDecode(claimsEncoded)
	if err != nil {
		return fmt.Errorf("failed to decode claims: %w", err)
	}

	// Parse the claims
	if err := json.Unmarshal(claimsJSON, claims); err != nil {
		return fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	// Check if the claims are valid (if they implement the Valid interface)
	if validator, ok := claims.(interface{ Valid() error }); ok {
		if err := validator.Valid(); err != nil {
			return err
		}
	}

	return nil
}

// sign signs the payload using HMAC-SHA256
func (s *Service) sign(payload string) string {
	h := hmac.New(sha256.New, s.signingKey)
	h.Write([]byte(payload))
	return base64URLEncode(h.Sum(nil))
}

// base64URLEncode encodes data to base64URL
func base64URLEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

// base64URLDecode decodes base64URL data
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	return base64.URLEncoding.DecodeString(s)
}
