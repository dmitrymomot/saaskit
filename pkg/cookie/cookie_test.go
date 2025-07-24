package cookie_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/cookie"
)

func TestNew(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		secrets []string
		wantErr error
	}{
		{
			name:    "no secrets",
			secrets: []string{},
			wantErr: cookie.ErrNoSecret,
		},
		{
			name:    "empty secrets",
			secrets: []string{"", ""},
			wantErr: cookie.ErrNoSecret,
		},
		{
			name:    "secret too short",
			secrets: []string{"short"},
			wantErr: cookie.ErrSecretTooShort,
		},
		{
			name:    "valid secret",
			secrets: []string{"this-is-a-very-long-secret-key-32-chars-long"},
			wantErr: nil,
		},
		{
			name: "multiple secrets with rotation",
			secrets: []string{
				"this-is-a-very-long-secret-key-32-chars-long",
				"this-is-old-very-long-secret-key-32-chars-ok",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := cookie.New(tt.secrets)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManager_SetGet(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"simple", "test", "value"},
		{"empty value", "empty", ""},
		{"special chars", "special", "hello=world&foo=bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			r := &http.Request{Header: http.Header{}}

			err := m.Set(w, tt.key, tt.value)
			if err != nil {
				t.Fatalf("Set() error = %v", err)
			}

			r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))

			got, err := m.Get(r, tt.key)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if got != tt.value {
				t.Errorf("Get() = %v, want %v", got, tt.value)
			}
		})
	}
}

func TestManager_SetGetSigned(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	w := httptest.NewRecorder()
	r := &http.Request{Header: http.Header{}}

	value := "test-value"
	err := m.SetSigned(w, "signed", value)
	if err != nil {
		t.Fatalf("SetSigned() error = %v", err)
	}

	r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))

	got, err := m.GetSigned(r, "signed")
	if err != nil {
		t.Fatalf("GetSigned() error = %v", err)
	}

	if got != value {
		t.Errorf("GetSigned() = %v, want %v", got, value)
	}
}

func TestManager_SignedTamperDetection(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	w := httptest.NewRecorder()

	err := m.SetSigned(w, "signed", "original-value")
	if err != nil {
		t.Fatalf("SetSigned() error = %v", err)
	}

	// Get the signed value and tamper with it
	r := &http.Request{Header: http.Header{}}
	r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))

	signedValue, _ := m.Get(r, "signed")
	parts := strings.Split(signedValue, "|")
	if len(parts) == 2 {
		// Tamper with the value part
		tamperedValue := base64.URLEncoding.EncodeToString([]byte("tampered-value")) + "|" + parts[1]
		r = &http.Request{Header: http.Header{}}
		r.AddCookie(&http.Cookie{Name: "signed", Value: tamperedValue})

		_, err = m.GetSigned(r, "signed")
		if !errors.Is(err, cookie.ErrInvalidSignature) {
			t.Errorf("GetSigned() with tampered cookie error = %v, want %v", err, cookie.ErrInvalidSignature)
		}
	} else {
		t.Errorf("Invalid signed cookie format")
	}
}

func TestManager_SetGetEncrypted(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	w := httptest.NewRecorder()
	r := &http.Request{Header: http.Header{}}

	value := "sensitive-data"
	err := m.SetEncrypted(w, "encrypted", value)
	if err != nil {
		t.Fatalf("SetEncrypted() error = %v", err)
	}

	r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))

	got, err := m.GetEncrypted(r, "encrypted")
	if err != nil {
		t.Fatalf("GetEncrypted() error = %v", err)
	}

	if got != value {
		t.Errorf("GetEncrypted() = %v, want %v", got, value)
	}
}

func TestManager_SecretRotation(t *testing.T) {
	t.Parallel()
	oldSecret := "old-secret-that-is-32-characters-long-exactly"
	newSecret := "new-secret-that-is-32-characters-long-exactly"

	m1, _ := cookie.New([]string{oldSecret})
	w := httptest.NewRecorder()
	r := &http.Request{Header: http.Header{}}

	err := m1.SetSigned(w, "test", "value")
	if err != nil {
		t.Fatalf("SetSigned() error = %v", err)
	}

	cookieStr := w.Header().Get("Set-Cookie")

	m2, _ := cookie.New([]string{newSecret, oldSecret})
	r.Header.Set("Cookie", cookieStr)

	got, err := m2.GetSigned(r, "test")
	if err != nil {
		t.Fatalf("GetSigned() with rotated secret error = %v", err)
	}

	if got != "value" {
		t.Errorf("GetSigned() = %v, want %v", got, "value")
	}
}

func TestManager_Flash(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	type testData struct {
		Message string
		Count   int
	}

	w := httptest.NewRecorder()
	r := &http.Request{Header: http.Header{}}

	original := testData{Message: "Hello", Count: 42}
	err := m.SetFlash(w, r, "notification", original)
	if err != nil {
		t.Fatalf("SetFlash() error = %v", err)
	}

	r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))
	w = httptest.NewRecorder()

	var got testData
	err = m.GetFlash(w, r, "notification", &got)
	if err != nil {
		t.Fatalf("GetFlash() error = %v", err)
	}

	if got != original {
		t.Errorf("GetFlash() = %v, want %v", got, original)
	}

	deleteCookie := w.Header().Get("Set-Cookie")
	if !strings.Contains(deleteCookie, "Max-Age=-1") && !strings.Contains(deleteCookie, "Max-Age=0") {
		t.Errorf("Flash cookie was not deleted after reading, got: %s", deleteCookie)
	}
}

func TestManager_Delete(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	w := httptest.NewRecorder()

	m.Delete(w, "test")

	cookieStr := w.Header().Get("Set-Cookie")
	if cookieStr == "" {
		t.Error("Delete() did not set any cookie")
		return
	}

	// Max-Age might be output as Max-Age=0 by some versions
	if !strings.Contains(cookieStr, "Max-Age=-1") && !strings.Contains(cookieStr, "Max-Age=0") {
		t.Errorf("Delete() did not set Max-Age=-1 or Max-Age=0, got: %s", cookieStr)
	}
	if !strings.Contains(cookieStr, "test=") {
		t.Errorf("Delete() did not set correct cookie name, got: %s", cookieStr)
	}
}

func TestManager_Options(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New(
		[]string{"this-is-a-very-long-secret-key-32-chars-long"},
		cookie.WithDomain(".example.com"),
		cookie.WithPath("/app"),
		cookie.WithSecure(true),
		cookie.WithHTTPOnly(false),
		cookie.WithSameSite(http.SameSiteStrictMode),
	)

	w := httptest.NewRecorder()

	err := m.Set(w, "test", "value", cookie.WithMaxAge(3600))
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	cookieStr := w.Header().Get("Set-Cookie")

	checks := []string{
		"Domain=example.com", // Go's http package strips leading dot
		"Path=/app",
		"Max-Age=3600",
		"Secure",
		"SameSite=Strict",
	}

	for _, check := range checks {
		if !strings.Contains(cookieStr, check) {
			t.Errorf("Cookie missing %s, got: %s", check, cookieStr)
		}
	}

	if strings.Contains(cookieStr, "HttpOnly") {
		t.Error("Cookie should not have HttpOnly")
	}
}

func BenchmarkManager_SetSigned(b *testing.B) {
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})
	w := httptest.NewRecorder()

	b.ResetTimer()
	for b.Loop() {
		m.SetSigned(w, "bench", "benchmark-value")
	}
}

func BenchmarkManager_GetSigned(b *testing.B) {
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})
	w := httptest.NewRecorder()
	r := &http.Request{Header: http.Header{}}

	m.SetSigned(w, "bench", "benchmark-value")
	r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))

	b.ResetTimer()
	for b.Loop() {
		m.GetSigned(r, "bench")
	}
}

func BenchmarkManager_SetEncrypted(b *testing.B) {
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})
	w := httptest.NewRecorder()

	b.ResetTimer()
	for b.Loop() {
		m.SetEncrypted(w, "bench", "benchmark-value")
	}
}

func BenchmarkManager_GetEncrypted(b *testing.B) {
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})
	w := httptest.NewRecorder()
	r := &http.Request{Header: http.Header{}}

	m.SetEncrypted(w, "bench", "benchmark-value")
	r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))

	b.ResetTimer()
	for b.Loop() {
		m.GetEncrypted(r, "bench")
	}
}

func TestManager_EdgeCases(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	t.Run("get non-existent cookie", func(t *testing.T) {
		t.Parallel()
		r := &http.Request{Header: http.Header{}}
		_, err := m.Get(r, "nonexistent")
		if !errors.Is(err, cookie.ErrCookieNotFound) {
			t.Errorf("Get() error = %v, want %v", err, cookie.ErrCookieNotFound)
		}
	})

	t.Run("get signed non-existent cookie", func(t *testing.T) {
		t.Parallel()
		r := &http.Request{Header: http.Header{}}
		_, err := m.GetSigned(r, "nonexistent")
		if !errors.Is(err, cookie.ErrCookieNotFound) {
			t.Errorf("GetSigned() error = %v, want %v", err, cookie.ErrCookieNotFound)
		}
	})

	t.Run("get encrypted non-existent cookie", func(t *testing.T) {
		t.Parallel()
		r := &http.Request{Header: http.Header{}}
		_, err := m.GetEncrypted(r, "nonexistent")
		if !errors.Is(err, cookie.ErrCookieNotFound) {
			t.Errorf("GetEncrypted() error = %v, want %v", err, cookie.ErrCookieNotFound)
		}
	})

	t.Run("very long cookie value", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := &http.Request{Header: http.Header{}}

		longValue := strings.Repeat("a", 4000)
		err := m.SetEncrypted(w, "long", longValue)
		if err != nil {
			t.Fatalf("SetEncrypted() error = %v", err)
		}

		r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))
		got, err := m.GetEncrypted(r, "long")
		if err != nil {
			t.Fatalf("GetEncrypted() error = %v", err)
		}

		if got != longValue {
			t.Errorf("GetEncrypted() length = %d, want %d", len(got), len(longValue))
		}
	})

	t.Run("empty cookie value", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := &http.Request{Header: http.Header{}}

		err := m.SetSigned(w, "empty", "")
		if err != nil {
			t.Fatalf("SetSigned() error = %v", err)
		}

		r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))
		got, err := m.GetSigned(r, "empty")
		if err != nil {
			t.Fatalf("GetSigned() error = %v", err)
		}

		if got != "" {
			t.Errorf("GetSigned() = %v, want empty string", got)
		}
	})

	t.Run("special characters in value", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := &http.Request{Header: http.Header{}}

		specialValue := "value with = & | and 中文字符"
		err := m.SetEncrypted(w, "special", specialValue)
		if err != nil {
			t.Fatalf("SetEncrypted() error = %v", err)
		}

		r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))
		got, err := m.GetEncrypted(r, "special")
		if err != nil {
			t.Fatalf("GetEncrypted() error = %v", err)
		}

		if got != specialValue {
			t.Errorf("GetEncrypted() = %v, want %v", got, specialValue)
		}
	})
}

func TestManager_EncryptedRotation(t *testing.T) {
	t.Parallel()
	oldSecret := "old-secret-that-is-32-characters-long-exactly"
	newSecret := "new-secret-that-is-32-characters-long-exactly"

	m1, _ := cookie.New([]string{oldSecret})
	w := httptest.NewRecorder()
	r := &http.Request{Header: http.Header{}}

	value := "encrypted-data"
	err := m1.SetEncrypted(w, "test", value)
	if err != nil {
		t.Fatalf("SetEncrypted() error = %v", err)
	}

	cookieStr := w.Header().Get("Set-Cookie")

	m2, _ := cookie.New([]string{newSecret, oldSecret})
	r.Header.Set("Cookie", cookieStr)

	got, err := m2.GetEncrypted(r, "test")
	if err != nil {
		t.Fatalf("GetEncrypted() with rotated secret error = %v", err)
	}

	if got != value {
		t.Errorf("GetEncrypted() = %v, want %v", got, value)
	}

	m3, _ := cookie.New([]string{newSecret})
	r.Header.Set("Cookie", cookieStr)

	_, err = m3.GetEncrypted(r, "test")
	if !errors.Is(err, cookie.ErrDecryptionFailed) {
		t.Errorf("GetEncrypted() without old secret error = %v, want %v", err, cookie.ErrDecryptionFailed)
	}
}

func TestManager_InvalidFormats(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	t.Run("invalid signed format - no separator", func(t *testing.T) {
		t.Parallel()
		r := &http.Request{Header: http.Header{}}
		r.AddCookie(&http.Cookie{Name: "test", Value: "noseparator"})

		_, err := m.GetSigned(r, "test")
		if !errors.Is(err, cookie.ErrInvalidFormat) {
			t.Errorf("GetSigned() error = %v, want %v", err, cookie.ErrInvalidFormat)
		}
	})

	t.Run("invalid signed format - bad base64", func(t *testing.T) {
		t.Parallel()
		r := &http.Request{Header: http.Header{}}
		r.AddCookie(&http.Cookie{Name: "test", Value: "invalid!base64|signature"})

		_, err := m.GetSigned(r, "test")
		if !errors.Is(err, cookie.ErrInvalidFormat) {
			t.Errorf("GetSigned() error = %v, want %v", err, cookie.ErrInvalidFormat)
		}
	})

	t.Run("invalid encrypted format - bad base64", func(t *testing.T) {
		t.Parallel()
		r := &http.Request{Header: http.Header{}}
		r.AddCookie(&http.Cookie{Name: "test", Value: "invalid!base64"})

		_, err := m.GetEncrypted(r, "test")
		if !errors.Is(err, cookie.ErrInvalidFormat) {
			t.Errorf("GetEncrypted() error = %v, want %v", err, cookie.ErrInvalidFormat)
		}
	})

	t.Run("invalid encrypted format - short ciphertext", func(t *testing.T) {
		t.Parallel()
		r := &http.Request{Header: http.Header{}}
		r.AddCookie(&http.Cookie{Name: "test", Value: base64.URLEncoding.EncodeToString([]byte("short"))})

		_, err := m.GetEncrypted(r, "test")
		if !errors.Is(err, cookie.ErrDecryptionFailed) {
			t.Errorf("GetEncrypted() error = %v, want %v", err, cookie.ErrDecryptionFailed)
		}
	})
}

func TestManager_FlashEdgeCases(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	t.Run("flash non-existent", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := &http.Request{Header: http.Header{}}

		var result string
		err := m.GetFlash(w, r, "nonexistent", &result)
		if !errors.Is(err, cookie.ErrCookieNotFound) {
			t.Errorf("GetFlash() error = %v, want %v", err, cookie.ErrCookieNotFound)
		}
	})

	t.Run("flash with complex struct", func(t *testing.T) {
		t.Parallel()
		type complexData struct {
			ID        string
			Timestamp time.Time
			Values    []int
			Meta      map[string]interface{}
		}

		w := httptest.NewRecorder()
		r := &http.Request{Header: http.Header{}}

		original := complexData{
			ID:        "test-123",
			Timestamp: time.Now().UTC(),
			Values:    []int{1, 2, 3, 4, 5},
			Meta: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
				"key3": true,
			},
		}

		err := m.SetFlash(w, r, "complex", original)
		if err != nil {
			t.Fatalf("SetFlash() error = %v", err)
		}

		r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))
		w = httptest.NewRecorder()

		var got complexData
		err = m.GetFlash(w, r, "complex", &got)
		if err != nil {
			t.Fatalf("GetFlash() error = %v", err)
		}

		if got.ID != original.ID {
			t.Errorf("GetFlash() ID = %v, want %v", got.ID, original.ID)
		}
		if !got.Timestamp.Equal(original.Timestamp) {
			t.Errorf("GetFlash() Timestamp = %v, want %v", got.Timestamp, original.Timestamp)
		}
	})

	t.Run("flash unmarshalable value", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := &http.Request{Header: http.Header{}}

		invalidValue := func() {}
		err := m.SetFlash(w, r, "invalid", invalidValue)
		if err == nil {
			t.Error("SetFlash() with function should return error")
		}
	})
}

func TestManager_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			w := httptest.NewRecorder()
			r := &http.Request{Header: http.Header{}}

			value := fmt.Sprintf("value-%d", id)
			err := m.SetEncrypted(w, "concurrent", value)
			if err != nil {
				t.Errorf("SetEncrypted() in goroutine %d error = %v", id, err)
				return
			}

			r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))
			got, err := m.GetEncrypted(r, "concurrent")
			if err != nil {
				t.Errorf("GetEncrypted() in goroutine %d error = %v", id, err)
				return
			}

			if got != value {
				t.Errorf("GetEncrypted() in goroutine %d = %v, want %v", id, got, value)
			}
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestManager_MultipleCookies(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

	w := httptest.NewRecorder()

	cookies := map[string]string{
		"plain":     "plain-value",
		"signed":    "signed-value",
		"encrypted": "encrypted-value",
	}

	err := m.Set(w, "plain", cookies["plain"])
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	err = m.SetSigned(w, "signed", cookies["signed"])
	if err != nil {
		t.Fatalf("SetSigned() error = %v", err)
	}

	err = m.SetEncrypted(w, "encrypted", cookies["encrypted"])
	if err != nil {
		t.Fatalf("SetEncrypted() error = %v", err)
	}

	req := &http.Request{Header: http.Header{}}
	for _, cookie := range w.Result().Cookies() {
		req.AddCookie(cookie)
	}

	got, err := m.Get(req, "plain")
	if err != nil || got != cookies["plain"] {
		t.Errorf("Get(plain) = %v, %v, want %v, nil", got, err, cookies["plain"])
	}

	got, err = m.GetSigned(req, "signed")
	if err != nil || got != cookies["signed"] {
		t.Errorf("GetSigned(signed) = %v, %v, want %v, nil", got, err, cookies["signed"])
	}

	got, err = m.GetEncrypted(req, "encrypted")
	if err != nil || got != cookies["encrypted"] {
		t.Errorf("GetEncrypted(encrypted) = %v, %v, want %v, nil", got, err, cookies["encrypted"])
	}
}

func TestManager_SecretsWithExactLength(t *testing.T) {
	t.Parallel()
	exactLengthSecret := strings.Repeat("a", 32) // minSecretLength = 32
	m, err := cookie.New([]string{exactLengthSecret})
	if err != nil {
		t.Errorf("New() with exact length secret error = %v", err)
	}

	w := httptest.NewRecorder()
	r := &http.Request{Header: http.Header{}}

	err = m.SetEncrypted(w, "test", "value")
	if err != nil {
		t.Fatalf("SetEncrypted() error = %v", err)
	}

	r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))
	got, err := m.GetEncrypted(r, "test")
	if err != nil {
		t.Fatalf("GetEncrypted() error = %v", err)
	}

	if got != "value" {
		t.Errorf("GetEncrypted() = %v, want %v", got, "value")
	}
}

func TestManager_DeleteWithCustomOptions(t *testing.T) {
	t.Parallel()
	m, _ := cookie.New(
		[]string{"this-is-a-very-long-secret-key-32-chars-long"},
		cookie.WithDomain("example.com"),
		cookie.WithPath("/app"),
		cookie.WithSecure(true),
		cookie.WithSameSite(http.SameSiteStrictMode),
	)

	w := httptest.NewRecorder()
	m.Delete(w, "test")

	cookieStr := w.Header().Get("Set-Cookie")

	checks := []string{
		"Domain=example.com",
		"Path=/app",
		"Secure",
		"SameSite=Strict",
		"HttpOnly",
	}

	for _, check := range checks {
		if !strings.Contains(cookieStr, check) {
			t.Errorf("Delete() cookie missing %s, got: %s", check, cookieStr)
		}
	}
}
