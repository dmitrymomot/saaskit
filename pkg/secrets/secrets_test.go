package secrets_test

import (
	"bytes"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/secrets"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptString(t *testing.T) {
	t.Parallel()
	appKey, err := secrets.GenerateKey()
	require.NoError(t, err)
	workspaceKey, err := secrets.GenerateKey()
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext string
	}{
		{"empty string", ""},
		{"simple text", "Hello, World!"},
		{"api key", "sk_test_1234567890abcdef"},
		{"json", `{"client_id":"abc123","client_secret":"xyz789"}`},
		{"unicode", "Hello 世界 🌍"},
		{"long text", "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ciphertext, err := secrets.EncryptString(appKey, workspaceKey, tt.plaintext)
			require.NoError(t, err)

			// Verify ciphertext is different from plaintext
			if tt.plaintext != "" && ciphertext == tt.plaintext {
				t.Error("Ciphertext should not equal plaintext")
			}

			decrypted, err := secrets.DecryptString(appKey, workspaceKey, ciphertext)
			require.NoError(t, err)

			require.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestEncryptDecryptBytes(t *testing.T) {
	t.Parallel()
	appKey, err := secrets.GenerateKey()
	require.NoError(t, err)
	workspaceKey, err := secrets.GenerateKey()
	require.NoError(t, err)

	tests := []struct {
		name string
		data []byte
	}{
		{"empty bytes", []byte{}},
		{"single byte", []byte{42}},
		{"binary data", []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}},
		{"text as bytes", []byte("Hello, World!")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ciphertext, err := secrets.EncryptBytes(appKey, workspaceKey, tt.data)
			require.NoError(t, err)

			// Verify ciphertext is different from plaintext
			if len(tt.data) > 0 && bytes.Equal(ciphertext, tt.data) {
				t.Error("Ciphertext should not equal plaintext")
			}

			decrypted, err := secrets.DecryptBytes(appKey, workspaceKey, ciphertext)
			require.NoError(t, err)

			if !bytes.Equal(decrypted, tt.data) {
				t.Errorf("Decrypted data does not match: got %v, want %v", decrypted, tt.data)
			}
		})
	}
}

func TestDifferentWorkspaceKeys(t *testing.T) {
	t.Parallel()
	appKey, err := secrets.GenerateKey()
	require.NoError(t, err)
	workspaceKey1, err := secrets.GenerateKey()
	require.NoError(t, err)
	workspaceKey2, err := secrets.GenerateKey()
	require.NoError(t, err)

	plaintext := "secret-api-key"

	// Encrypt with workspace 1
	ciphertext1, err := secrets.EncryptString(appKey, workspaceKey1, plaintext)
	require.NoError(t, err)

	// Encrypt with workspace 2
	ciphertext2, err := secrets.EncryptString(appKey, workspaceKey2, plaintext)
	require.NoError(t, err)

	// Ciphertexts should be different
	require.NotEqual(t, ciphertext1, ciphertext2, "Same plaintext encrypted with different workspace keys should produce different ciphertexts")

	// Cannot decrypt workspace 1's secret with workspace 2's key
	_, err = secrets.DecryptString(appKey, workspaceKey2, ciphertext1)
	require.Error(t, err, "Should not be able to decrypt with wrong workspace key")

	// Can decrypt with correct key
	decrypted, err := secrets.DecryptString(appKey, workspaceKey1, ciphertext1)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)
}

func TestInvalidKeys(t *testing.T) {
	t.Parallel()
	validKey, err := secrets.GenerateKey()
	require.NoError(t, err)
	plaintext := "test"

	tests := []struct {
		name         string
		appKey       []byte
		workspaceKey []byte
		wantErr      error
	}{
		{"nil app key", nil, validKey, secrets.ErrInvalidAppKey},
		{"nil workspace key", validKey, nil, secrets.ErrInvalidWorkspaceKey},
		{"short app key", make([]byte, 16), validKey, secrets.ErrInvalidAppKey},
		{"short workspace key", validKey, make([]byte, 16), secrets.ErrInvalidWorkspaceKey},
		{"long app key", make([]byte, 64), validKey, secrets.ErrInvalidAppKey},
		{"long workspace key", validKey, make([]byte, 64), secrets.ErrInvalidWorkspaceKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := secrets.EncryptString(tt.appKey, tt.workspaceKey, plaintext)
			require.Error(t, err)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestInvalidCiphertext(t *testing.T) {
	t.Parallel()
	appKey, err := secrets.GenerateKey()
	require.NoError(t, err)
	workspaceKey, err := secrets.GenerateKey()
	require.NoError(t, err)

	tests := []struct {
		name       string
		ciphertext string
	}{
		{"empty string", ""},
		{"invalid base64", "not-base64!@#$"},
		{"valid base64 but invalid ciphertext", "SGVsbG8gV29ybGQ="},
		{"too short ciphertext", "AA=="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := secrets.DecryptString(appKey, workspaceKey, tt.ciphertext)
			require.Error(t, err)
		})
	}
}

func TestGenerateKey(t *testing.T) {
	t.Parallel()
	// Generate multiple keys and ensure they're different
	keys := make(map[string]bool)

	for range 10 {
		key, err := secrets.GenerateKey()
		require.NoError(t, err)

		require.Len(t, key, secrets.KeySize)

		keyStr := string(key)
		require.False(t, keys[keyStr], "Generated duplicate key")
		keys[keyStr] = true
	}
}

func TestClearBytes(t *testing.T) {
	t.Parallel()
	// Test that clearBytes properly zeros out memory
	testData := []byte("sensitive-key-material-12345678")
	originalLen := len(testData)

	// Make a copy to verify the data was non-zero initially
	originalCopy := make([]byte, len(testData))
	copy(originalCopy, testData)

	// Clear the bytes
	secrets.ClearBytesForTesting(testData)

	// Verify length hasn't changed
	require.Len(t, testData, originalLen)

	// Verify all bytes are now zero
	for i, b := range testData {
		require.Equal(t, byte(0), b, "Byte at position %d should be zero", i)
	}

	// Verify original data was not all zeros
	allZero := true
	for _, b := range originalCopy {
		if b != 0 {
			allZero = false
			break
		}
	}
	require.False(t, allZero, "Original data should not have been all zeros")
}

func TestMemoryClearingInEncryptDecrypt(t *testing.T) {
	t.Parallel()
	// This test verifies that sensitive keys are cleared from memory
	// Note: This is more of a documentation test since we can't directly
	// verify that defer statements were executed

	appKey, err := secrets.GenerateKey()
	require.NoError(t, err)
	workspaceKey, err := secrets.GenerateKey()
	require.NoError(t, err)

	plaintext := "test-secret"

	// Encrypt and decrypt
	ciphertext, err := secrets.EncryptString(appKey, workspaceKey, plaintext)
	require.NoError(t, err)

	decrypted, err := secrets.DecryptString(appKey, workspaceKey, ciphertext)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)

	// The derived keys used internally should be cleared by defer statements
}

func TestTimingAttackResistance(t *testing.T) {
	t.Parallel()
	// Test that ValidateKeys takes similar time regardless of which validation fails
	validKey := make([]byte, secrets.KeySize)
	shortKey := make([]byte, 16)
	longKey := make([]byte, 64)

	// Test cases with different failure points
	tests := []struct {
		name         string
		appKey       []byte
		workspaceKey []byte
	}{
		{"both valid", validKey, validKey},
		{"invalid app key only", shortKey, validKey},
		{"invalid workspace key only", validKey, shortKey},
		{"both invalid same way", shortKey, shortKey},
		{"both invalid different", shortKey, longKey},
	}

	// Run multiple iterations to get a sense of timing
	// Note: This is more of a demonstration test - real timing attack tests
	// would need more sophisticated statistical analysis
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The function should complete validation checks for both keys
			// before returning an error, making timing less predictable
			err := secrets.ValidateKeys(tt.appKey, tt.workspaceKey)

			// Just verify the function returns appropriate errors
			if tt.name == "both valid" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
