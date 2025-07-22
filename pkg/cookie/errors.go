package cookie

import "errors"

var (
	ErrNoSecret         = errors.New("cookie.no_secret")
	ErrSecretTooShort   = errors.New("cookie.secret_too_short")
	ErrInvalidSignature = errors.New("cookie.invalid_signature")
	ErrDecryptionFailed = errors.New("cookie.decryption_failed")
	ErrCookieNotFound   = errors.New("cookie.not_found")
	ErrInvalidFormat    = errors.New("cookie.invalid_format")
)
