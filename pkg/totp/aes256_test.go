package totp_test

import (
	"encoding/base64"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/totp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptSecret(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		plainText string
		key       []byte
		wantErr   error
	}{
		{
			name:      "Valid encryption and decryption",
			plainText: "MYSECRETKEY123",
			key:       make([]byte, 32),
			wantErr:   nil,
		},
		{
			name:      "Empty plaintext",
			plainText: "",
			key:       make([]byte, 32),
			wantErr:   nil,
		},
		{
			name:      "Invalid key size",
			plainText: "MYSECRETKEY123",
			key:       make([]byte, 16),
			wantErr:   totp.ErrInvalidEncryptionKeyLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Encrypt
			encrypted, err := totp.EncryptSecret(tt.plainText, tt.key)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, encrypted)

			// Decrypt
			decrypted, err := totp.DecryptSecret(encrypted, tt.key)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.plainText, decrypted)
		})
	}
}

func TestDecryptSecret_Invalid(t *testing.T) {
	t.Parallel()
	key := make([]byte, 32)
	tests := []struct {
		name             string
		cipherTextBase64 string
	}{
		{
			name:             "Invalid base64",
			cipherTextBase64: "invalid-base64!@#$",
		},
		{
			name:             "Too short ciphertext",
			cipherTextBase64: base64.StdEncoding.EncodeToString([]byte("short")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := totp.DecryptSecret(tt.cipherTextBase64, key)
			assert.Error(t, err)
		})
	}
}

func TestGenerateEncryptionKey(t *testing.T) {
	t.Parallel()
	key, err := totp.GenerateEncryptionKey()
	require.NoError(t, err)
	assert.Len(t, key, 32)
}

func TestGenerateEncodedEncryptionKey(t *testing.T) {
	t.Parallel()
	key, err := totp.GenerateEncodedEncryptionKey()
	require.NoError(t, err)
	require.NotEmpty(t, key)

	decoded, err := base64.StdEncoding.DecodeString(key)
	require.NoError(t, err)
	require.Len(t, decoded, 32)
}
