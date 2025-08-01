package totp

import "errors"

var (
	ErrFailedToEncryptSecret         = errors.New("failed to encrypt TOTP secret")
	ErrFailedToDecryptSecret         = errors.New("failed to decrypt TOTP secret")
	ErrInvalidCipherTooShort         = errors.New("cipher text too short")
	ErrFailedToGenerateEncryptionKey = errors.New("failed to generate encryption key")
	ErrFailedToLoadEncryptionKey     = errors.New("failed to load encryption key")
	ErrInvalidEncryptionKeyLength    = errors.New("invalid encryption key length")
	ErrFailedToGenerateSecretKey     = errors.New("failed to generate TOTP secret key")
	ErrFailedToValidateTOTP          = errors.New("failed to validate TOTP")
	ErrMissingSecret                 = errors.New("missing secret")
	ErrInvalidSecret                 = errors.New("invalid secret")
	ErrMissingAccountName            = errors.New("missing account name")
	ErrMissingIssuer                 = errors.New("missing issuer")
	ErrEncryptionKeyNotSet           = errors.New("TOTP encryption key not set")
	ErrInvalidOTP                    = errors.New("invalid OTP format")
	ErrInvalidRecoveryCodeCount      = errors.New("invalid recovery code count, must be greater than 0")
	ErrFailedToGenerateRecoveryCode  = errors.New("failed to generate recovery code")
	ErrFailedToGenerateTOTP          = errors.New("failed to generate TOTP")
)
