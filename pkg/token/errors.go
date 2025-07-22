package token

import "errors"

var (
	ErrInvalidToken     = errors.New("invalid token format")
	ErrSignatureInvalid = errors.New("signature mismatch")
)
