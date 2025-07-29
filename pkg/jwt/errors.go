package jwt

import "errors"

var (
	ErrInvalidToken            = errors.New("invalid token")
	ErrExpiredToken            = errors.New("token is expired")
	ErrInvalidSigningMethod    = errors.New("invalid signing method")
	ErrMissingSigningKey       = errors.New("missing signing key")
	ErrInvalidSigningKey       = errors.New("invalid signing key")
	ErrInvalidClaims           = errors.New("invalid claims")
	ErrMissingClaims           = errors.New("missing claims")
	ErrInvalidSignature        = errors.New("invalid signature")
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
)
