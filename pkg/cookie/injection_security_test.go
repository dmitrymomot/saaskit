package cookie_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/cookie"
)

func TestCookieNameInjectionPrevention(t *testing.T) {
	t.Parallel()

	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	maliciousNames := []struct {
		name        string
		cookieName  string
		description string
	}{
		{"newline_injection", "test\nSet-Cookie: evil=value", "Newline injection in cookie name"},
		{"carriage_return", "test\rSet-Cookie: evil=value", "Carriage return injection"},
		{"crlf_injection", "test\r\nSet-Cookie: evil=value", "CRLF injection"},
		{"semicolon_injection", "test; evil=value", "Semicolon injection"},
		{"comma_injection", "test, evil=value", "Comma injection"},
		{"equals_injection", "test=evil", "Equals sign injection"},
		{"space_injection", "test evil", "Space injection"},
		{"tab_injection", "test\tevil", "Tab injection"},
		{"null_byte", "test\x00evil", "Null byte injection"},
		{"unicode_newline", "test\u2028evil", "Unicode line separator"},
		{"unicode_paragraph", "test\u2029evil", "Unicode paragraph separator"},
	}

	for _, tc := range maliciousNames {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()

			_ = m.Set(w, tc.cookieName, "value")

			// Go's http.SetCookie validates cookie names, preventing header injection
			cookieStr := w.Header().Get("Set-Cookie")

			if cookieStr != "" {
				setCookieHeaders := w.Header().Values("Set-Cookie")
				assert.Len(t, setCookieHeaders, 1, "Should only have one Set-Cookie header for %s", tc.description)

				assert.NotContains(t, cookieStr, "evil=value", "Cookie should not contain injected content for %s", tc.description)
			}

			w2 := httptest.NewRecorder()
			_ = m.SetSigned(w2, tc.cookieName, "value")

			cookieStr2 := w2.Header().Get("Set-Cookie")
			if cookieStr2 != "" {
				setCookieHeaders2 := w2.Header().Values("Set-Cookie")
				assert.Len(t, setCookieHeaders2, 1, "Should only have one Set-Cookie header for signed %s", tc.description)
				assert.NotContains(t, cookieStr2, "evil=value", "Signed cookie should not contain injected content for %s", tc.description)
			}

			w3 := httptest.NewRecorder()
			_ = m.SetEncrypted(w3, tc.cookieName, "value")

			cookieStr3 := w3.Header().Get("Set-Cookie")
			if cookieStr3 != "" {
				setCookieHeaders3 := w3.Header().Values("Set-Cookie")
				assert.Len(t, setCookieHeaders3, 1, "Should only have one Set-Cookie header for encrypted %s", tc.description)
				assert.NotContains(t, cookieStr3, "evil=value", "Encrypted cookie should not contain injected content for %s", tc.description)
			}
		})
	}
}

func TestCookieValueInjectionPrevention(t *testing.T) {
	t.Parallel()

	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	maliciousValues := []struct {
		name        string
		value       string
		description string
	}{
		{"newline_injection", "value\nSet-Cookie: evil=injected", "Newline injection in value"},
		{"crlf_injection", "value\r\nSet-Cookie: evil=injected", "CRLF injection in value"},
		{"semicolon_attack", "value; Path=/evil", "Semicolon attribute injection"},
		{"comma_injection", "value, evil=injected", "Comma injection"},
		{"null_byte", "value\x00; evil=injected", "Null byte injection"},
		{"unicode_controls", "value\u2028; evil=injected", "Unicode control characters"},
	}

	for _, tc := range maliciousValues {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			_ = m.Set(w, "test", tc.value)

			cookieStr := w.Header().Get("Set-Cookie")
			if cookieStr != "" {
				setCookieHeaders := w.Header().Values("Set-Cookie")
				assert.Len(t, setCookieHeaders, 1, "Should only have one Set-Cookie header for %s", tc.description)

				// Go automatically quotes values containing special characters to prevent injection
				t.Logf("Cookie value for %s: %s", tc.description, cookieStr)

				if strings.Contains(tc.value, "\n") || strings.Contains(tc.value, "\r") ||
					strings.Contains(tc.value, ";") || strings.Contains(tc.value, ",") {
					assert.True(t, strings.Contains(cookieStr, "\"") ||
						!strings.Contains(cookieStr, "evil=injected"),
						"Dangerous values should be quoted or sanitized for %s", tc.description)
				}
			}
		})
	}
}

func TestFlashCookieInjectionPrevention(t *testing.T) {
	t.Parallel()

	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	maliciousFlashData := []struct {
		name string
		data any
		desc string
	}{
		{
			"string_injection",
			"normal\r\nSet-Cookie: evil=flash",
			"String with CRLF injection",
		},
		{
			"map_injection",
			map[string]string{
				"message": "hello\nSet-Cookie: evil=map",
				"type":    "success",
			},
			"Map with injection in value",
		},
		{
			"slice_injection",
			[]string{"item1", "item2\r\nevil=slice", "item3"},
			"Slice with injection",
		},
	}

	for _, tc := range maliciousFlashData {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			r := &http.Request{Header: http.Header{}}

			err := m.SetFlash(w, r, "notification", tc.data)

			if err == nil {
				setCookieHeaders := w.Header().Values("Set-Cookie")
				assert.Len(t, setCookieHeaders, 1, "Should only have one Set-Cookie header for %s", tc.desc)

				cookieStr := w.Header().Get("Set-Cookie")
				assert.NotContains(t, cookieStr, "evil=flash", "Flash cookie should not contain injected content")
				assert.NotContains(t, cookieStr, "evil=map", "Flash cookie should not contain injected content")
				assert.NotContains(t, cookieStr, "evil=slice", "Flash cookie should not contain injected content")
			}
		})
	}
}

func TestHeaderInjectionViaOptions(t *testing.T) {
	t.Parallel()

	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	maliciousOptions := []struct {
		name   string
		option cookie.Option
		desc   string
	}{
		{
			"domain_injection",
			cookie.WithDomain("example.com\r\nSet-Cookie: evil=domain"),
			"Domain injection",
		},
		{
			"path_injection",
			cookie.WithPath("/app\r\nSet-Cookie: evil=path"),
			"Path injection",
		},
	}

	for _, tc := range maliciousOptions {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			_ = m.Set(w, "test", "value", tc.option)

			setCookieHeaders := w.Header().Values("Set-Cookie")

			assert.LessOrEqual(t, len(setCookieHeaders), 1, "Should not create multiple Set-Cookie headers for %s", tc.desc)

			if len(setCookieHeaders) > 0 {
				cookieStr := setCookieHeaders[0]

				// Go sanitizes domain/path attributes, preventing header injection
				t.Logf("Cookie with potentially malicious %s: %s", tc.desc, cookieStr)

				hasCompleteHeaderInjection := strings.Contains(cookieStr, "\nSet-Cookie:") ||
					strings.Contains(cookieStr, "\r\nSet-Cookie:")

				// Critical security check: injection must not create new headers
				assert.False(t, hasCompleteHeaderInjection, "Must not allow header injection that creates new Set-Cookie headers")
			}
		})
	}
}

func TestCookieNameValidation(t *testing.T) {
	t.Parallel()

	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	validNames := []string{
		"test",
		"session_id",
		"user-token",
		"APP_COOKIE",
		"cookie123",
		"_underscore",
		"-dash",
	}

	for _, name := range validNames {
		t.Run("valid_"+name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			_ = m.Set(w, name, "value")

			cookieStr := w.Header().Get("Set-Cookie")
			if cookieStr != "" {
				assert.Contains(t, cookieStr, name+"=", "Valid cookie name should be preserved")
			}
		})
	}
}

func TestMultipleCookieHeaderHandling(t *testing.T) {
	t.Parallel()

	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	w := httptest.NewRecorder()

	err := m.Set(w, "cookie1", "value1")
	require.NoError(t, err)

	err = m.Set(w, "cookie2", "value2")
	require.NoError(t, err)

	err = m.SetSigned(w, "cookie3", "value3")
	require.NoError(t, err)

	setCookieHeaders := w.Header().Values("Set-Cookie")
	assert.Len(t, setCookieHeaders, 3, "Should have exactly 3 Set-Cookie headers")

	cookies := map[string]bool{
		"cookie1=value1": false,
		"cookie2=value2": false,
		"cookie3=":       false, // Signed cookie will have different value
	}

	for _, header := range setCookieHeaders {
		for pattern := range cookies {
			if strings.Contains(header, pattern) {
				cookies[pattern] = true
			}
		}
	}

	assert.True(t, cookies["cookie1=value1"], "Should find cookie1")
	assert.True(t, cookies["cookie2=value2"], "Should find cookie2")
	assert.True(t, cookies["cookie3="], "Should find cookie3")
}
