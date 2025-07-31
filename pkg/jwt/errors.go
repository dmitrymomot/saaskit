package jwt

import "errors"

var (
	ErrInvalidToken            = errors.New("jwt: invalid token")
	ErrExpiredToken            = errors.New("jwt: token is expired")
	ErrInvalidSigningMethod    = errors.New("jwt: invalid signing method")
	ErrMissingSigningKey       = errors.New("jwt: missing signing key")
	ErrInvalidSigningKey       = errors.New("jwt: invalid signing key")
	ErrInvalidClaims           = errors.New("jwt: invalid claims")
	ErrMissingClaims           = errors.New("jwt: missing claims")
	ErrInvalidSignature        = errors.New("jwt: invalid signature")
	ErrUnexpectedSigningMethod = errors.New("jwt: unexpected signing method")
)
