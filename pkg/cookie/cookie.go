package cookie

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
)

const (
	minSecretLength = 32
	flashPrefix     = "__flash_"
)

type Manager struct {
	secrets  []string
	defaults Options
}

func New(secrets []string, opts ...Option) (*Manager, error) {
	if len(secrets) == 0 {
		return nil, ErrNoSecret
	}

	secrets = slices.DeleteFunc(secrets, func(s string) bool { return s == "" })
	if len(secrets) == 0 {
		return nil, ErrNoSecret
	}

	for i, s := range secrets {
		if len(s) < minSecretLength {
			return nil, fmt.Errorf("%w: secret %d has %d chars, need at least %d", ErrSecretTooShort, i, len(s), minSecretLength)
		}
	}

	defaults := Options{
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	defaults = applyOptions(defaults, opts)

	return &Manager{
		secrets:  secrets,
		defaults: defaults,
	}, nil
}

func (m *Manager) Set(w http.ResponseWriter, name, value string, opts ...Option) error {
	options := applyOptions(m.defaults, opts)

	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     options.Path,
		Domain:   options.Domain,
		MaxAge:   options.MaxAge,
		Secure:   options.Secure,
		HttpOnly: options.HttpOnly,
		SameSite: options.SameSite,
	}

	http.SetCookie(w, cookie)
	return nil
}

func (m *Manager) Get(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return "", ErrCookieNotFound
		}
		return "", err
	}
	return cookie.Value, nil
}

func (m *Manager) Delete(w http.ResponseWriter, name string) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     m.defaults.Path,
		Domain:   m.defaults.Domain,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: m.defaults.HttpOnly,
		SameSite: m.defaults.SameSite,
		Secure:   m.defaults.Secure,
	}
	http.SetCookie(w, cookie)
}

func (m *Manager) SetSigned(w http.ResponseWriter, name, value string, opts ...Option) error {
	signed := m.sign(value)
	return m.Set(w, name, signed, opts...)
}

func (m *Manager) GetSigned(r *http.Request, name string) (string, error) {
	signed, err := m.Get(r, name)
	if err != nil {
		return "", err
	}

	return m.verify(signed)
}

func (m *Manager) SetEncrypted(w http.ResponseWriter, name, value string, opts ...Option) error {
	encrypted, err := m.encrypt(value)
	if err != nil {
		return err
	}
	return m.Set(w, name, encrypted, opts...)
}

func (m *Manager) GetEncrypted(r *http.Request, name string) (string, error) {
	encrypted, err := m.Get(r, name)
	if err != nil {
		return "", err
	}

	return m.decrypt(encrypted)
}

func (m *Manager) SetFlash(w http.ResponseWriter, r *http.Request, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal flash: %w", err)
	}

	return m.SetEncrypted(w, flashPrefix+key, string(data))
}

func (m *Manager) GetFlash(w http.ResponseWriter, r *http.Request, key string, dest any) error {
	cookieName := flashPrefix + key

	data, err := m.GetEncrypted(r, cookieName)
	if err != nil {
		return err
	}

	// Flash cookies are automatically deleted after reading to prevent replay attacks
	m.Delete(w, cookieName)

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("unmarshal flash: %w", err)
	}

	return nil
}

func (m *Manager) sign(value string) string {
	mac := hmac.New(sha256.New, []byte(m.secrets[0]))
	mac.Write([]byte(value))
	signature := base64.URLEncoding.EncodeToString(mac.Sum(nil))

	return base64.URLEncoding.EncodeToString([]byte(value)) + "|" + signature
}

func (m *Manager) verify(signed string) (string, error) {
	parts := strings.SplitN(signed, "|", 2)
	if len(parts) != 2 {
		return "", ErrInvalidFormat
	}

	encodedValue, signature := parts[0], parts[1]

	value, err := base64.URLEncoding.DecodeString(encodedValue)
	if err != nil {
		return "", ErrInvalidFormat
	}

	// Try all secrets to support key rotation - old cookies remain valid during transition
	for _, secret := range m.secrets {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(value)
		expectedSig := base64.URLEncoding.EncodeToString(mac.Sum(nil))

		// Use constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) == 1 {
			return string(value), nil
		}
	}

	return "", ErrInvalidSignature
}

func (m *Manager) encrypt(value string) (string, error) {
	// AES-256 requires exactly 32 bytes for the key
	block, err := aes.NewCipher([]byte(m.secrets[0][:32]))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Generate cryptographically secure random nonce to ensure each encryption is unique
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Prepend nonce to ciphertext for self-contained decryption
	ciphertext := gcm.Seal(nonce, nonce, []byte(value), nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

func (m *Manager) decrypt(encrypted string) (string, error) {
	ciphertext, err := base64.URLEncoding.DecodeString(encrypted)
	if err != nil {
		return "", ErrInvalidFormat
	}

	// Try all secrets to support key rotation during decryption
	var lastErr error
	for _, secret := range m.secrets {
		block, err := aes.NewCipher([]byte(secret[:32]))
		if err != nil {
			lastErr = err
			continue
		}

		gcm, err := cipher.NewGCM(block)
		if err != nil {
			lastErr = err
			continue
		}

		if len(ciphertext) < gcm.NonceSize() {
			lastErr = ErrInvalidFormat
			continue
		}

		// Extract nonce from the beginning of ciphertext
		nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
		plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err == nil {
			return string(plaintext), nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return "", ErrDecryptionFailed
	}
	return "", ErrDecryptionFailed
}
