package jwt

import "errors"

var (
	// ErrInvalidToken is returned when the token is invalid
	ErrInvalidToken = errors.New("invalid token")

	// ErrExpiredToken is returned when the token is expired
	ErrExpiredToken = errors.New("token is expired")

	// ErrInvalidSigningMethod is returned when the signing method is invalid
	ErrInvalidSigningMethod = errors.New("invalid signing method")

	// ErrMissingSigningKey is returned when the signing key is missing
	ErrMissingSigningKey = errors.New("missing signing key")

	// ErrInvalidSigningKey is returned when the signing key is invalid
	ErrInvalidSigningKey = errors.New("invalid signing key")

	// ErrInvalidClaims is returned when the claims are invalid
	ErrInvalidClaims = errors.New("invalid claims")

	// ErrMissingClaims is returned when the claims are missing
	ErrMissingClaims = errors.New("missing claims")

	// ErrInvalidSignature is returned when the signature is invalid
	ErrInvalidSignature = errors.New("invalid signature")

	// ErrUnexpectedSigningMethod is returned when the signing method is unexpected
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
)
