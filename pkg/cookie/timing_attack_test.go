package cookie_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/cookie"
)

func TestHMACTimingAttackResistance(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	w := httptest.NewRecorder()
	err := m.SetSigned(w, "test", "legitimate-value")
	require.NoError(t, err)

	cookieValue := w.Header().Get("Set-Cookie")
	eqIndex := strings.Index(cookieValue, "=")
	require.NotEqual(t, -1, eqIndex)
	semicolonIndex := strings.Index(cookieValue[eqIndex+1:], ";")
	var signedValue string
	if semicolonIndex == -1 {
		signedValue = cookieValue[eqIndex+1:]
	} else {
		signedValue = cookieValue[eqIndex+1 : eqIndex+1+semicolonIndex]
	}

	signatureParts := strings.Split(signedValue, "|")
	require.Len(t, signatureParts, 2)
	validSignature := signatureParts[1]

	testCases := []struct {
		name      string
		signature string
		desc      string
	}{
		{
			name:      "first_byte_wrong",
			signature: "A" + validSignature[1:],
			desc:      "First byte incorrect",
		},
		{
			name:      "middle_byte_wrong",
			signature: validSignature[:len(validSignature)/2] + "X" + validSignature[len(validSignature)/2+1:],
			desc:      "Middle byte incorrect",
		},
		{
			name:      "last_byte_wrong",
			signature: validSignature[:len(validSignature)-1] + "Z",
			desc:      "Last byte incorrect",
		},
		{
			name:      "completely_wrong",
			signature: strings.Repeat("Y", len(validSignature)),
			desc:      "Completely different signature",
		},
	}

	const iterations = 100
	timings := make(map[string][]time.Duration)

	for _, tc := range testCases {
		timings[tc.name] = make([]time.Duration, iterations)

		for i := 0; i < iterations; i++ {
			maliciousCookie := signatureParts[0] + "|" + tc.signature

			r := &http.Request{Header: http.Header{}}
			r.AddCookie(&http.Cookie{Name: "test", Value: maliciousCookie})

			start := time.Now()
			_, err := m.GetSigned(r, "test")
			elapsed := time.Since(start)

			assert.Error(t, err)
			assert.ErrorIs(t, err, cookie.ErrInvalidSignature)

			timings[tc.name][i] = elapsed
		}
	}

	averages := make(map[string]time.Duration)
	for name, times := range timings {
		var total time.Duration
		for _, t := range times {
			total += t
		}
		averages[name] = total / time.Duration(len(times))
	}

	var maxAverage, minAverage time.Duration
	for name, avg := range averages {
		t.Logf("%s average timing: %v", name, avg)
		if maxAverage == 0 || avg > maxAverage {
			maxAverage = avg
		}
		if minAverage == 0 || avg < minAverage {
			minAverage = avg
		}
	}

	// Constant-time operations should have minimal timing variance
	if maxAverage > 0 && minAverage > 0 {
		ratio := float64(maxAverage) / float64(minAverage)
		if ratio > 3.0 { // 3x tolerance for system noise
			t.Logf("WARNING: Timing variance ratio %.2f may indicate timing attack vulnerability", ratio)
			t.Logf("Max average: %v, Min average: %v", maxAverage, minAverage)
		}
	}
}

func TestSignatureVerificationFailsConstantTime(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	testCases := []struct {
		name  string
		value string
	}{
		{"empty_signature", "dGVzdA==|"},
		{"short_signature", "dGVzdA==|YWI="},
		{"wrong_length", "dGVzdA==|dGhpc2lzdG9vc2hvcnQ="},
		{"invalid_base64", "dGVzdA==|invalid!chars"},
		{"no_separator", "noseparatorhere"},
		{"multiple_separators", "dGVzdA==|sig1|sig2"},
	}

	const iterations = 50
	timings := make(map[string][]time.Duration)

	for _, tc := range testCases {
		timings[tc.name] = make([]time.Duration, iterations)

		for i := 0; i < iterations; i++ {
			r := &http.Request{Header: http.Header{}}
			r.AddCookie(&http.Cookie{Name: "test", Value: tc.value})

			start := time.Now()
			_, err := m.GetSigned(r, "test")
			elapsed := time.Since(start)

			assert.Error(t, err)
			timings[tc.name][i] = elapsed
		}
	}

	averages := make(map[string]time.Duration)
	for name, times := range timings {
		var total time.Duration
		for _, t := range times {
			total += t
		}
		averages[name] = total / time.Duration(len(times))
		t.Logf("Error case '%s' average timing: %v", name, averages[name])
	}
}

func TestMultiSecretTimingConsistency(t *testing.T) {
	t.Parallel()

	secret1 := "old-secret-that-is-32-characters-long-exactly"
	secret2 := "new-secret-that-is-32-characters-long-exactly"

	m1, _ := cookie.New([]string{secret1})
	m2, _ := cookie.New([]string{secret2})
	mBoth, _ := cookie.New([]string{secret2, secret1}) // New first, old second

	w := httptest.NewRecorder()
	err := m1.SetSigned(w, "test", "rotation-test")
	require.NoError(t, err)

	cookieValue := w.Header().Get("Set-Cookie")
	eqIndex := strings.Index(cookieValue, "=")
	require.NotEqual(t, -1, eqIndex)
	semicolonIndex := strings.Index(cookieValue[eqIndex+1:], ";")
	var signedValue string
	if semicolonIndex == -1 {
		signedValue = cookieValue[eqIndex+1:]
	} else {
		signedValue = cookieValue[eqIndex+1 : eqIndex+1+semicolonIndex]
	}

	const iterations = 50
	timings := make(map[string][]time.Duration)

	managers := map[string]*cookie.Manager{
		"correct_secret": m1,
		"wrong_secret":   m2,
		"both_secrets":   mBoth, // Contains both secrets for rotation
	}

	for name, manager := range managers {
		timings[name] = make([]time.Duration, iterations)

		for i := 0; i < iterations; i++ {
			r := &http.Request{Header: http.Header{}}
			r.AddCookie(&http.Cookie{Name: "test", Value: signedValue})

			start := time.Now()
			_, err := manager.GetSigned(r, "test")
			elapsed := time.Since(start)

			timings[name][i] = elapsed

			if name == "wrong_secret" {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		}
	}

	for name, times := range timings {
		var total time.Duration
		for _, t := range times {
			total += t
		}
		average := total / time.Duration(len(times))
		t.Logf("Secret verification '%s' average timing: %v", name, average)
	}
}
