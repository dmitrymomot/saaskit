package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	DefaultDigits    = 6      // Standard 6-digit TOTP codes
	DefaultPeriod    = 30     // 30-second validity window (RFC 6238 standard)
	DefaultAlgorithm = "SHA1" // HMAC-SHA1 algorithm (RFC 6238 standard)
)

var (
	// ValidateSecretKeyRegex ensures Base32 format: uppercase A-Z, digits 2-7, optional padding
	ValidateSecretKeyRegex = regexp.MustCompile("^[A-Z2-7]+=*$")
)

// TOTPParams contains the parameters for TOTP URI generation
type TOTPParams struct {
	Secret      string // Base32-encoded TOTP secret key (required)
	AccountName string // User identifier like email (required)
	Issuer      string // Service name displayed in authenticator apps (required)
	Algorithm   string // HMAC algorithm (optional, defaults to SHA1)
	Digits      int    // Number of digits in generated codes (optional, defaults to 6)
	Period      int    // Code validity period in seconds (optional, defaults to 30)
}

// Validate ensures all required TOTP parameters are present and valid
func (p TOTPParams) Validate() error {
	if p.Secret == "" {
		return ErrMissingSecret
	}
	if !ValidateSecretKeyRegex.MatchString(p.Secret) {
		return ErrInvalidSecret
	}
	if p.AccountName == "" {
		return ErrMissingAccountName
	}
	if p.Issuer == "" {
		return ErrMissingIssuer
	}
	return nil
}

// GetDefaults returns a copy with RFC 6238 standard defaults applied to zero-valued fields
func (p TOTPParams) GetDefaults() TOTPParams {
	if p.Algorithm == "" {
		p.Algorithm = DefaultAlgorithm
	}
	if p.Digits == 0 {
		p.Digits = DefaultDigits
	}
	if p.Period == 0 {
		p.Period = DefaultPeriod
	}
	return p
}

// GenerateSecretKey generates a new Base32-encoded secret key for TOTP.
func GenerateSecretKey() (string, error) {
	secret := make([]byte, 20) // 160-bit secret (RFC 4226 recommendation for cryptographic strength)
	if _, err := rand.Read(secret); err != nil {
		return "", errors.Join(ErrFailedToGenerateSecretKey, err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// GetTOTPURI creates a properly encoded TOTP URI for use with authenticator apps.
// The URI format follows the Key Uri Format specification:
// https://github.com/google/google-authenticator/wiki/Key-Uri-Format
func GetTOTPURI(params TOTPParams) (string, error) {
	if err := params.Validate(); err != nil {
		return "", err
	}

	params = params.GetDefaults()

	label := fmt.Sprintf("%s:%s",
		url.PathEscape(params.Issuer),
		url.PathEscape(params.AccountName),
	)

	query := url.Values{}
	query.Set("secret", params.Secret)
	query.Set("issuer", params.Issuer)
	query.Set("algorithm", params.Algorithm)
	query.Set("digits", fmt.Sprintf("%d", params.Digits))
	query.Set("period", fmt.Sprintf("%d", params.Period))

	uri := fmt.Sprintf("otpauth://totp/%s?%s", label, query.Encode())

	return uri, nil
}

// ValidateTOTP validates the TOTP code provided by the user.
func ValidateTOTP(secret, otp string) (bool, error) {
	secret = strings.TrimSpace(strings.ToUpper(secret))
	if !ValidateSecretKeyRegex.MatchString(secret) {
		return false, ErrInvalidSecret
	}

	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return false, errors.Join(ErrFailedToValidateTOTP, err)
	}

	otp = strings.TrimSpace(otp)
	if !regexp.MustCompile(fmt.Sprintf(`^\d{%d}$`, DefaultDigits)).MatchString(otp) {
		return false, ErrInvalidOTP
	}

	currentTime := time.Now().Unix()
	interval := int64(DefaultPeriod)
	counter := currentTime / interval

	// Accept codes from previous, current, and next 30-second windows to handle clock drift
	for i := -1; i <= 1; i++ {
		code := GenerateHOTP(key, counter+int64(i), DefaultDigits)
		if fmt.Sprintf("%06d", code) == otp {
			return true, nil
		}
	}

	return false, nil
}

// GenerateTOTP generates a time-based one-time password for the current 30-second window.
// The secret must be a valid Base32-encoded string.
func GenerateTOTP(secret string) (string, error) {
	secret = strings.TrimSpace(strings.ToUpper(secret))
	if !ValidateSecretKeyRegex.MatchString(secret) {
		return "", ErrInvalidSecret
	}

	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", errors.Join(ErrFailedToGenerateTOTP, err)
	}

	currentTime := time.Now().Unix()
	counter := currentTime / int64(DefaultPeriod)

	code := GenerateHOTP(key, counter, DefaultDigits)

	return fmt.Sprintf("%06d", code), nil
}

// GenerateHOTP implements RFC 4226 HMAC-based One-Time Password algorithm.
// The algorithm converts a counter value into a numeric code using HMAC-SHA1.
func GenerateHOTP(key []byte, counter int64, digits int) int {
	// Convert counter to big-endian 8-byte array (RFC 4226 requirement)
	counterBytes := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		counterBytes[i] = byte(counter & 0xff)
		counter = counter >> 8
	}

	// Calculate HMAC-SHA1 hash of the counter
	hmacHash := hmac.New(sha1.New, key)
	hmacHash.Write(counterBytes)
	hash := hmacHash.Sum(nil)

	// Dynamic truncation (RFC 4226): use last 4 bits as offset into hash
	offset := hash[len(hash)-1] & 0x0f
	// Extract 31-bit value (clear MSB to ensure positive number)
	code := (int(hash[offset]&0x7f) << 24) |
		(int(hash[offset+1]&0xff) << 16) |
		(int(hash[offset+2]&0xff) << 8) |
		(int(hash[offset+3] & 0xff))

	// Reduce to desired number of digits
	code = code % int(math.Pow10(digits))

	return code
}

// GenerateTOTPWithTime generates a TOTP code for the 30-second window containing the specified time.
// Useful for testing or generating codes for specific moments.
func GenerateTOTPWithTime(secret string, t time.Time) (string, error) {
	secret = strings.TrimSpace(strings.ToUpper(secret))
	if !ValidateSecretKeyRegex.MatchString(secret) {
		return "", ErrInvalidSecret
	}

	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", errors.Join(ErrFailedToGenerateTOTP, err)
	}

	counter := t.Unix() / int64(DefaultPeriod)

	code := GenerateHOTP(key, counter, DefaultDigits)

	return fmt.Sprintf("%06d", code), nil
}
