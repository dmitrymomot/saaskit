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
	// DefaultDigits is the default number of digits in the TOTP code
	DefaultDigits = 6
	// DefaultPeriod is the default period in seconds for TOTP code generation
	DefaultPeriod = 30
	// DefaultAlgorithm is the default algorithm used for TOTP code generation
	DefaultAlgorithm = "SHA1"
)

var (
	// ValidateSecretKeyRegex is used to validate the secret key format
	ValidateSecretKeyRegex = regexp.MustCompile("^[A-Z2-7]+=*$")
)

// TOTPParams contains the parameters for TOTP URI generation
type TOTPParams struct {
	// Secret is the TOTP secret key (required)
	Secret string
	// AccountName is the name of the account (required)
	AccountName string
	// Issuer is the name of the issuer (required)
	Issuer string
	// Algorithm is the algorithm used for TOTP code generation (optional, defaults to SHA1)
	Algorithm string
	// Digits is the number of digits in the TOTP code (optional, defaults to 6)
	Digits int
	// Period is the period in seconds for TOTP code generation (optional, defaults to 30)
	Period int
}

// Validate validates the TOTP parameters
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

// GetDefaults returns a copy of TOTPParams with default values set for optional fields
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
	secret := make([]byte, 20) // 160 bits as recommended by RFC 4226
	if _, err := rand.Read(secret); err != nil {
		return "", errors.Join(ErrFailedToGenerateSecretKey, err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// GetTOTPURI creates a properly encoded TOTP URI for use with authenticator apps.
// The URI format follows the Key Uri Format specification:
// https://github.com/google/google-authenticator/wiki/Key-Uri-Format
func GetTOTPURI(params TOTPParams) (string, error) {
	// Validate required parameters
	if err := params.Validate(); err != nil {
		return "", err
	}

	// Set default values for optional parameters
	params = params.GetDefaults()

	// Create the base URI without query parameters
	label := fmt.Sprintf("%s:%s",
		url.PathEscape(params.Issuer),
		url.PathEscape(params.AccountName),
	)

	// Create query parameters
	query := url.Values{}
	query.Set("secret", params.Secret)
	query.Set("issuer", params.Issuer)
	query.Set("algorithm", params.Algorithm)
	query.Set("digits", fmt.Sprintf("%d", params.Digits))
	query.Set("period", fmt.Sprintf("%d", params.Period))

	// Build the complete URI
	uri := fmt.Sprintf("otpauth://totp/%s?%s", label, query.Encode())

	return uri, nil
}

// ValidateTOTP validates the TOTP code provided by the user.
func ValidateTOTP(secret, otp string) (bool, error) {
	// Normalize and validate secret
	secret = strings.TrimSpace(strings.ToUpper(secret))
	if !ValidateSecretKeyRegex.MatchString(secret) {
		return false, ErrInvalidSecret
	}

	// Decode the secret
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return false, errors.Join(ErrFailedToValidateTOTP, err)
	}

	// Clean and validate OTP format
	otp = strings.TrimSpace(otp)
	if !regexp.MustCompile(fmt.Sprintf(`^\d{%d}$`, DefaultDigits)).MatchString(otp) {
		return false, ErrInvalidOTP
	}

	currentTime := time.Now().Unix()
	interval := int64(DefaultPeriod)
	counter := currentTime / interval

	// Allow for a time window of +/- 30 seconds
	for i := -1; i <= 1; i++ {
		code := GenerateHOTP(key, counter+int64(i), DefaultDigits)
		if fmt.Sprintf("%06d", code) == otp {
			return true, nil
		}
	}

	return false, nil
}

// GenerateTOTP generates a TOTP code.
// The code is valid for the current time period.
// The secret key must be a valid Base32-encoded string.
// The code is a 6-digit number.
func GenerateTOTP(secret string) (string, error) {
	// Normalize and validate secret
	secret = strings.TrimSpace(strings.ToUpper(secret))
	if !ValidateSecretKeyRegex.MatchString(secret) {
		return "", ErrInvalidSecret
	}

	// Decode the secret
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", errors.Join(ErrFailedToGenerateTOTP, err)
	}

	// Generate TOTP based on current time
	currentTime := time.Now().Unix()
	counter := currentTime / int64(DefaultPeriod)

	code := GenerateHOTP(key, counter, DefaultDigits)

	return fmt.Sprintf("%06d", code), nil
}

// GenerateHOTP generates an HOTP code.
func GenerateHOTP(key []byte, counter int64, digits int) int {
	counterBytes := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		counterBytes[i] = byte(counter & 0xff)
		counter = counter >> 8
	}

	hmacHash := hmac.New(sha1.New, key)
	hmacHash.Write(counterBytes)
	hash := hmacHash.Sum(nil)

	offset := hash[len(hash)-1] & 0x0f
	code := (int(hash[offset]&0x7f) << 24) |
		(int(hash[offset+1]&0xff) << 16) |
		(int(hash[offset+2]&0xff) << 8) |
		(int(hash[offset+3] & 0xff))

	code = code % int(math.Pow10(digits))

	return code
}

// GenerateTOTPWithTime generates a TOTP code for the specified time.
// The code is valid for the time period that includes the specified time.
// The secret key must be a valid Base32-encoded string.
// The code is a 6-digit number.
func GenerateTOTPWithTime(secret string, t time.Time) (string, error) {
	// Normalize and validate secret
	secret = strings.TrimSpace(strings.ToUpper(secret))
	if !ValidateSecretKeyRegex.MatchString(secret) {
		return "", ErrInvalidSecret
	}

	// Decode the secret
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", errors.Join(ErrFailedToGenerateTOTP, err)
	}

	// Generate TOTP based on the specified time
	counter := t.Unix() / int64(DefaultPeriod)

	code := GenerateHOTP(key, counter, DefaultDigits)

	return fmt.Sprintf("%06d", code), nil
}
