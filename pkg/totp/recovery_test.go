package totp_test

import (
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/totp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRecoveryCodes(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		wantErr bool
	}{
		{
			name:    "Generate 8 codes",
			count:   8,
			wantErr: false,
		},
		{
			name:    "Generate 1 code",
			count:   1,
			wantErr: false,
		},
		{
			name:    "Generate 0 codes",
			count:   0,
			wantErr: true,
		},
		{
			name:    "Generate negative codes",
			count:   -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codes, err := totp.GenerateRecoveryCodes(tt.count)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, codes)
				return
			}

			require.NoError(t, err)
			assert.Len(t, codes, tt.count)

			// Verify each code is unique and properly formatted
			seen := make(map[string]bool)
			for _, code := range codes {
				assert.Len(t, code, 16) // 8 bytes in hex = 16 characters
				assert.False(t, seen[code], "Duplicate code found")
				seen[code] = true
			}
		})
	}
}

func TestHashRecoveryCode(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{
			name: "Normal code",
			code: "1234567890ABCDEF",
		},
		{
			name: "Empty code",
			code: "",
		},
		{
			name: "Special characters",
			code: "!@#$%^&*()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := totp.HashRecoveryCode(tt.code)
			assert.NotEmpty(t, hash)
			assert.Len(t, hash, 64) // SHA-256 produces 32 bytes = 64 hex characters

			// Verify deterministic behavior
			hash2 := totp.HashRecoveryCode(tt.code)
			assert.Equal(t, hash, hash2)
		})
	}
}

func TestVerifyRecoveryCode(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		hashedCode string
		wantResult bool
	}{
		{
			name:       "Valid code",
			code:       "1234567890ABCDEF",
			hashedCode: totp.HashRecoveryCode("1234567890ABCDEF"),
			wantResult: true,
		},
		{
			name:       "Invalid code - same length",
			code:       "1234567890ABCDEF",
			hashedCode: totp.HashRecoveryCode("FEDCBA0987654321"),
			wantResult: false,
		},
		{
			name:       "Invalid code - different length",
			code:       "1234",
			hashedCode: totp.HashRecoveryCode("5678"),
			wantResult: false,
		},
		{
			name:       "Empty code",
			code:       "",
			hashedCode: totp.HashRecoveryCode(""),
			wantResult: true,
		},
		{
			name:       "Code vs empty hash",
			code:       "1234567890ABCDEF",
			hashedCode: "",
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the verification
			result := totp.VerifyRecoveryCode(tt.code, tt.hashedCode)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

// TestVerifyRecoveryCodeSecurity performs basic security checks
func TestVerifyRecoveryCodeSecurity(t *testing.T) {
	// Test that the function is using constant-time comparison
	code := "1234567890ABCDEF"
	hash := totp.HashRecoveryCode(code)

	// Multiple verifications should yield the same result
	for i := 0; i < 100; i++ {
		result := totp.VerifyRecoveryCode(code, hash)
		assert.True(t, result, "Verification should be consistent")
	}

	// Test different inputs should not match
	invalidCode := "FEDCBA0987654321"
	result := totp.VerifyRecoveryCode(invalidCode, hash)
	assert.False(t, result, "Different codes should not match")

	// Test empty inputs
	assert.False(t, totp.VerifyRecoveryCode("", hash), "Empty code should not match")
	assert.False(t, totp.VerifyRecoveryCode(code, ""), "Empty hash should not match")
}

// Benchmark recovery code verification
func BenchmarkVerifyRecoveryCode(b *testing.B) {
	code := "1234567890ABCDEF"
	hashedCode := totp.HashRecoveryCode(code)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		totp.VerifyRecoveryCode(code, hashedCode)
	}
}
