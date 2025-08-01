package cookie_test

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/cookie"
)

func TestNonceUniqueness(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	nonceSet := make(map[string]bool)
	const numIterations = 1000
	value := "test-value-for-nonce-uniqueness"

	for i := 0; i < numIterations; i++ {
		w := httptest.NewRecorder()
		err := m.SetEncrypted(w, "test", value)
		require.NoError(t, err)

		cookieValue := w.Header().Get("Set-Cookie")
		eqIndex := strings.Index(cookieValue, "=")
		require.NotEqual(t, -1, eqIndex)
		semicolonIndex := strings.Index(cookieValue[eqIndex+1:], ";")
		var encryptedValue string
		if semicolonIndex == -1 {
			encryptedValue = cookieValue[eqIndex+1:]
		} else {
			encryptedValue = cookieValue[eqIndex+1 : eqIndex+1+semicolonIndex]
		}

		ciphertext, err := base64.URLEncoding.DecodeString(encryptedValue)
		require.NoError(t, err)

		// GCM nonces are 12 bytes by standard
		require.GreaterOrEqual(t, len(ciphertext), 12)
		nonce := string(ciphertext[:12])

		if nonceSet[nonce] {
			t.Fatalf("Nonce collision detected at iteration %d. This is a critical security vulnerability!", i)
		}
		nonceSet[nonce] = true
	}

	assert.Len(t, nonceSet, numIterations, "All nonces should be unique")
}

func TestConcurrentNonceUniqueness(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	const numGoroutines = 50
	const iterationsPerGoroutine = 20
	nonceMap := make(map[string]int)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < iterationsPerGoroutine; i++ {
				w := httptest.NewRecorder()
				err := m.SetEncrypted(w, "test", "concurrent-test")
				require.NoError(t, err)

				cookieValue := w.Header().Get("Set-Cookie")
				eqIndex := strings.Index(cookieValue, "=")
				require.NotEqual(t, -1, eqIndex)
				semicolonIndex := strings.Index(cookieValue[eqIndex+1:], ";")
				var encryptedValue string
				if semicolonIndex == -1 {
					encryptedValue = cookieValue[eqIndex+1:]
				} else {
					encryptedValue = cookieValue[eqIndex+1 : eqIndex+1+semicolonIndex]
				}

				ciphertext, err := base64.URLEncoding.DecodeString(encryptedValue)
				require.NoError(t, err)
				require.GreaterOrEqual(t, len(ciphertext), 12)

				nonce := string(ciphertext[:12])

				mu.Lock()
				if count, exists := nonceMap[nonce]; exists {
					t.Errorf("Nonce reuse detected! Goroutine %d, iteration %d. Previous count: %d",
						goroutineID, i, count)
				}
				nonceMap[nonce] = goroutineID*1000 + i
				mu.Unlock()
			}
		}(g)
	}

	wg.Wait()

	expectedTotal := numGoroutines * iterationsPerGoroutine
	assert.Len(t, nonceMap, expectedTotal, "All concurrent nonces should be unique")
}

func TestNonceReuseDetection(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	encryptedValues := make([]string, 100)
	value := "same-plaintext-value"

	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		err := m.SetEncrypted(w, "test", value)
		require.NoError(t, err)

		cookieValue := w.Header().Get("Set-Cookie")
		eqIndex := strings.Index(cookieValue, "=")
		require.NotEqual(t, -1, eqIndex)
		semicolonIndex := strings.Index(cookieValue[eqIndex+1:], ";")
		if semicolonIndex == -1 {
			encryptedValues[i] = cookieValue[eqIndex+1:]
		} else {
			encryptedValues[i] = cookieValue[eqIndex+1 : eqIndex+1+semicolonIndex]
		}
	}

	// Each encryption must produce different ciphertext due to unique nonces
	for i := 0; i < 100; i++ {
		for j := i + 1; j < 100; j++ {
			if encryptedValues[i] == encryptedValues[j] {
				t.Fatalf("Identical ciphertext found at indices %d and %d. This indicates nonce reuse!", i, j)
			}
		}
	}
}

func TestNonceFromCryptoRand(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	const numSamples = 500
	nonces := make([][]byte, numSamples)

	for i := 0; i < numSamples; i++ {
		w := httptest.NewRecorder()
		err := m.SetEncrypted(w, "test", "entropy-test")
		require.NoError(t, err)

		cookieValue := w.Header().Get("Set-Cookie")
		eqIndex := strings.Index(cookieValue, "=")
		require.NotEqual(t, -1, eqIndex)
		semicolonIndex := strings.Index(cookieValue[eqIndex+1:], ";")
		var encryptedValue string
		if semicolonIndex == -1 {
			encryptedValue = cookieValue[eqIndex+1:]
		} else {
			encryptedValue = cookieValue[eqIndex+1 : eqIndex+1+semicolonIndex]
		}

		ciphertext, err := base64.URLEncoding.DecodeString(encryptedValue)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(ciphertext), 12)

		nonces[i] = make([]byte, 12)
		copy(nonces[i], ciphertext[:12])
	}

	// Statistical test for crypto/rand quality
	zeroCount := 0
	for i := 0; i < numSamples; i++ {
		for j := 0; j < 12; j++ {
			if nonces[i][j] == 0 {
				zeroCount++
			}
		}
	}

	// Expected ~0.39% zero bytes in random data
	expectedZeros := float64(numSamples*12) / 256.0
	tolerance := expectedZeros * 0.5

	if float64(zeroCount) > expectedZeros+tolerance {
		t.Errorf("Too many zero bytes in nonces (%d), expected around %.1f (±%.1f). Possible weak entropy source.",
			zeroCount, expectedZeros, tolerance)
	}

	sequentialCount := 0
	for i := 0; i < numSamples-1; i++ {
		if subtle.ConstantTimeCompare(nonces[i], nonces[i+1]) == 1 {
			sequentialCount++
		}
	}

	if sequentialCount > 0 {
		t.Errorf("Found %d sequential identical nonces, indicating weak randomness source", sequentialCount)
	}
}
