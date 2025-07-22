# Secrets Package

This package provides secure encryption and decryption of secrets using a compound key approach. It combines an application-wide master key with workspace-specific keys to provide strong isolation between different workspaces/tenants.

## Features

- **Compound Key Encryption**: Combines app key + workspace key using HKDF
- **AES-256-GCM**: Industry-standard authenticated encryption
- **Type-Safe API**: Separate methods for strings and bytes
- **Workspace Isolation**: Each workspace has its own encryption context
- **No External Dependencies**: Uses only Go standard library + x/crypto

## Usage

### Basic String Encryption

```go
import "github.com/dmitrymomot/saaskit/pkg/secrets"

// Keys must be exactly 32 bytes
appKey := []byte("your-32-byte-app-key-from-env...")
workspaceKey := []byte("workspace-specific-32-byte-key...")

// Encrypt a secret
encrypted, err := secrets.EncryptString(appKey, workspaceKey, "sk_test_1234567890")
if err != nil {
    log.Fatal(err)
}

// Decrypt a secret
decrypted, err := secrets.DecryptString(appKey, workspaceKey, encrypted)
if err != nil {
    log.Fatal(err)
}
```

### Binary Data Encryption

```go
// Encrypt binary data
pdfData := []byte{...}
encrypted, err := secrets.EncryptBytes(appKey, workspaceKey, pdfData)

// Decrypt binary data
decrypted, err := secrets.DecryptBytes(appKey, workspaceKey, encrypted)
```

### Key Generation

```go
// Generate a new key (e.g., for workspace)
key, err := secrets.GenerateKey()
if err != nil {
    log.Fatal(err)
}
// Store this key securely (encrypted with app key)
```

## Security Considerations

1. **App Key Storage**:
    - Store in environment variable
    - Never commit to version control
    - Use key management service in production

2. **Workspace Key Storage**:
    - Store encrypted in database
    - Encrypt workspace keys using app key before storage
    - Consider key rotation strategy

3. **Key Derivation**:
    - Uses HKDF with SHA-256
    - Provides cryptographic domain separation
    - Each workspace has unique encryption context

## Implementation Details

- **Encryption Algorithm**: AES-256-GCM
- **Key Derivation**: HKDF-SHA256
- **Nonce Size**: 12 bytes (GCM standard)
- **Output Format**: `base64(nonce || ciphertext || tag)`
- **Key Size**: 32 bytes for both app and workspace keys

## Error Handling

The package provides specific error types:

- `ErrInvalidAppKey`: App key is not 32 bytes
- `ErrInvalidWorkspaceKey`: Workspace key is not 32 bytes
- `ErrEncryptionFailed`: Encryption operation failed
- `ErrDecryptionFailed`: Decryption operation failed
- `ErrInvalidCiphertext`: Ciphertext format is invalid
- `ErrKeyDerivationFailed`: Key derivation operation failed

## Example: Storing API Keys

```go
// In your service layer
type SecretService struct {
    appKey []byte
    db     *sql.DB
}

func (s *SecretService) StoreAPIKey(workspaceID, name, apiKey string) error {
    // Load workspace key from DB (assume it's already decrypted)
    workspaceKey := s.getWorkspaceKey(workspaceID)

    // Encrypt the API key
    encrypted, err := secrets.EncryptString(s.appKey, workspaceKey, apiKey)
    if err != nil {
        return fmt.Errorf("failed to encrypt API key: %w", err)
    }

    // Store in database
    _, err = s.db.Exec(
        "INSERT INTO secrets (workspace_id, name, encrypted_value) VALUES ($1, $2, $3)",
        workspaceID, name, encrypted,
    )
    return err
}
```

## API Reference

### Constants

```go
const KeySize = 32  // Required size for both app and workspace keys (256 bits)
```

### Functions

```go
// EncryptString encrypts a string using compound key from app and workspace keys.
// Returns base64-encoded ciphertext.
func EncryptString(appKey, workspaceKey []byte, plaintext string) (string, error)

// DecryptString decrypts a base64-encoded ciphertext back to string.
func DecryptString(appKey, workspaceKey []byte, ciphertext string) (string, error)

// EncryptBytes encrypts raw bytes using compound key from app and workspace keys.
// Returns ciphertext in format: nonce + encrypted data + tag
func EncryptBytes(appKey, workspaceKey []byte, data []byte) ([]byte, error)

// DecryptBytes decrypts ciphertext back to raw bytes.
// Expects ciphertext in format: nonce + encrypted data + tag
func DecryptBytes(appKey, workspaceKey []byte, ciphertext []byte) ([]byte, error)

// GenerateKey creates a new random 32-byte key suitable for encryption
func GenerateKey() ([]byte, error)

// ValidateKeys checks that both keys are the correct length
func ValidateKeys(appKey, workspaceKey []byte) error
```

### Error Variables

```go
var (
    ErrInvalidAppKey       = errors.New("invalid app key: must be 32 bytes")
    ErrInvalidWorkspaceKey = errors.New("invalid workspace key: must be 32 bytes")
    ErrEncryptionFailed    = errors.New("encryption failed")
    ErrDecryptionFailed    = errors.New("decryption failed")
    ErrInvalidCiphertext   = errors.New("invalid ciphertext format")
    ErrKeyDerivationFailed = errors.New("key derivation failed")
)
```
