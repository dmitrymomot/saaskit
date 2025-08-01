package totp_test

import (
	"testing"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/totp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSecretKey(t *testing.T) {
	t.Parallel()
	secret, err := totp.GenerateSecretKey()
	require.NoError(t, err)
	assert.NotEmpty(t, secret)
	assert.Regexp(t, totp.ValidateSecretKeyRegex, secret)
}

func TestGetTOTPURI(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		params  totp.TOTPParams
		want    string
		wantErr bool
	}{
		{
			name: "Basic URI",
			params: totp.TOTPParams{
				Secret:      "ABCDEFGHIJKLMNOP",
				AccountName: "test@example.com",
				Issuer:      "TestApp",
			},
			want:    "otpauth://totp/TestApp:test@example.com?algorithm=SHA1&digits=6&issuer=TestApp&period=30&secret=ABCDEFGHIJKLMNOP",
			wantErr: false,
		},
		{
			name: "URI with special characters",
			params: totp.TOTPParams{
				Secret:      "ABCDEFGHIJKLMNOP",
				AccountName: "test+user@example.com",
				Issuer:      "Test & App",
				Algorithm:   "SHA1",
				Digits:      6,
				Period:      30,
			},
			want:    "otpauth://totp/Test%20&%20App:test+user@example.com?algorithm=SHA1&digits=6&issuer=Test+%26+App&period=30&secret=ABCDEFGHIJKLMNOP",
			wantErr: false,
		},
		// ... rest of the test cases remain the same
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := totp.GetTOTPURI(tt.params)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateTOTP(t *testing.T) {
	t.Parallel()
	validSecret, err := totp.GenerateSecretKey()
	require.NoError(t, err)
	require.NotEmpty(t, validSecret)

	// Generate valid OTP for testing
	validOTP, err := totp.GenerateTOTP(validSecret)
	require.NoError(t, err)
	require.NotEmpty(t, validOTP)

	tests := []struct {
		name    string
		secret  string
		otp     string
		wantErr bool
		result  bool
	}{
		{
			name:    "Invalid base32 secret",
			secret:  "invalid-base32!@#$",
			otp:     "123456",
			wantErr: true,
			result:  false,
		},
		{
			name:    "Invalid OTP length",
			secret:  "ABCDEFGHIJKLMNOP",
			otp:     "12345",
			wantErr: true,
			result:  false,
		},
		{
			name:    "Invalid OTP characters",
			secret:  "ABCDEFGHIJKLMNOP",
			otp:     "12345a",
			wantErr: true,
			result:  false,
		},
		{
			name:    "Empty secret",
			secret:  "",
			otp:     "123456",
			wantErr: true,
			result:  false,
		},
		{
			name:    "Empty OTP",
			secret:  "ABCDEFGHIJKLMNOP",
			otp:     "",
			wantErr: true,
			result:  false,
		},
		{
			name:    "Valid OTP",
			secret:  validSecret,
			otp:     validOTP,
			wantErr: false,
			result:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := totp.ValidateTOTP(tt.secret, tt.otp)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.result, result)
		})
	}
}

func TestGenerateHOTP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		key     []byte
		counter int64
		digits  int
		wantErr bool
	}{
		{
			name:    "Valid HOTP",
			key:     []byte("12345678901234567890"),
			counter: 0,
			digits:  6,
			wantErr: false,
		},
		{
			name:    "8 digits",
			key:     []byte("12345678901234567890"),
			counter: 1,
			digits:  8,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code := totp.GenerateHOTP(tt.key, tt.counter, tt.digits)
			assert.True(t, code >= 0)
			assert.True(t, code < int(pow10(tt.digits)))
		})
	}
}

func pow10(n int) int64 {
	result := int64(1)
	for range n {
		result *= 10
	}
	return result
}

func TestValidateTOTPWithTimeWindow(t *testing.T) {
	t.Parallel()
	validSecret, err := totp.GenerateSecretKey()
	require.NoError(t, err)
	require.NotEmpty(t, validSecret)

	// Generate OTPs for -30s, now, and +30s
	pastOTP, err := totp.GenerateTOTPWithTime(validSecret, time.Now().Add(-30*time.Second))
	require.NoError(t, err)

	currentOTP, err := totp.GenerateTOTP(validSecret)
	require.NoError(t, err)

	futureOTP, err := totp.GenerateTOTPWithTime(validSecret, time.Now().Add(30*time.Second))
	require.NoError(t, err)

	tests := []struct {
		name    string
		otp     string
		wantErr bool
		result  bool
	}{
		{
			name:    "Past OTP within window",
			otp:     pastOTP,
			wantErr: false,
			result:  true,
		},
		{
			name:    "Current OTP",
			otp:     currentOTP,
			wantErr: false,
			result:  true,
		},
		{
			name:    "Future OTP within window",
			otp:     futureOTP,
			wantErr: false,
			result:  true,
		},
		{
			name:    "Invalid OTP",
			otp:     "123456",
			wantErr: false,
			result:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := totp.ValidateTOTP(validSecret, tt.otp)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.result, result)
		})
	}
}
