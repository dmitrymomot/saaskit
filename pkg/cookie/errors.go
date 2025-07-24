package cookie

import "errors"

var (
	ErrNoSecret            = errors.New("no secret provided for cookie manager")
	ErrSecretTooShort      = errors.New("secret must be at least 32 characters long")
	ErrInvalidSignature    = errors.New("cookie signature verification failed")
	ErrDecryptionFailed    = errors.New("failed to decrypt cookie value")
	ErrCookieNotFound      = errors.New("cookie not found in request")
	ErrInvalidFormat       = errors.New("invalid cookie format")
	ErrInvalidSecretLength = errors.New("secret length is invalid for AES encryption")
)
