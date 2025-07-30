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

func TestDefaultSecurityAttributes(t *testing.T) {
	t.Parallel()

	m, err := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	err = m.Set(w, "test", "value")
	require.NoError(t, err)

	cookieStr := w.Header().Get("Set-Cookie")
	require.NotEmpty(t, cookieStr)

	assert.Contains(t, cookieStr, "HttpOnly", "Cookies should have HttpOnly by default")
	assert.Contains(t, cookieStr, "SameSite=Lax", "Cookies should have SameSite=Lax by default")
	assert.Contains(t, cookieStr, "Path=/", "Cookies should have Path=/ by default")

	// Secure is false by default for development flexibility
	assert.NotContains(t, cookieStr, "Secure", "Cookies should not be Secure by default")
}

func TestSecureAttributeEnforcement(t *testing.T) {
	t.Parallel()

	m, _ := cookie.New(
		[]string{"this-is-a-very-long-secret-key-32-chars-long"},
		cookie.WithSecure(true),
	)

	testCases := []struct {
		name   string
		method func(w http.ResponseWriter, name, value string, opts ...cookie.Option) error
	}{
		{"Set", func(w http.ResponseWriter, name, value string, opts ...cookie.Option) error {
			return m.Set(w, name, value, opts...)
		}},
		{"SetSigned", func(w http.ResponseWriter, name, value string, opts ...cookie.Option) error {
			return m.SetSigned(w, name, value, opts...)
		}},
		{"SetEncrypted", func(w http.ResponseWriter, name, value string, opts ...cookie.Option) error {
			return m.SetEncrypted(w, name, value, opts...)
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			err := tc.method(w, "secure-test", "value")
			require.NoError(t, err)

			cookieStr := w.Header().Get("Set-Cookie")
			assert.Contains(t, cookieStr, "Secure", "%s should set Secure attribute", tc.name)
		})
	}
}

func TestHttpOnlyAttributeEnforcement(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		httpOnly   bool
		shouldHave bool
	}{
		{"httponly_true", true, true},
		{"httponly_false", false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, _ := cookie.New(
				[]string{"this-is-a-very-long-secret-key-32-chars-long"},
				cookie.WithHTTPOnly(tc.httpOnly),
			)

			w := httptest.NewRecorder()
			err := m.Set(w, "httponly-test", "value")
			require.NoError(t, err)

			cookieStr := w.Header().Get("Set-Cookie")

			if tc.shouldHave {
				assert.Contains(t, cookieStr, "HttpOnly", "Cookie should have HttpOnly attribute")
			} else {
				assert.NotContains(t, cookieStr, "HttpOnly", "Cookie should not have HttpOnly attribute")
			}
		})
	}
}

func TestSameSiteAttributeEnforcement(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		sameSite http.SameSite
		expected string
	}{
		{"samesite_strict", http.SameSiteStrictMode, "SameSite=Strict"},
		{"samesite_lax", http.SameSiteLaxMode, "SameSite=Lax"},
		{"samesite_none", http.SameSiteNoneMode, "SameSite=None"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, _ := cookie.New(
				[]string{"this-is-a-very-long-secret-key-32-chars-long"},
				cookie.WithSameSite(tc.sameSite),
			)

			w := httptest.NewRecorder()
			err := m.Set(w, "samesite-test", "value")
			require.NoError(t, err)

			cookieStr := w.Header().Get("Set-Cookie")
			assert.Contains(t, cookieStr, tc.expected, "Cookie should have correct SameSite attribute")
		})
	}
}

func TestSameSiteNoneRequiresSecure(t *testing.T) {
	t.Parallel()

	// SameSite=None requires Secure attribute per browser requirements
	m, _ := cookie.New(
		[]string{"this-is-a-very-long-secret-key-32-chars-long"},
		cookie.WithSameSite(http.SameSiteNoneMode),
		cookie.WithSecure(true),
	)

	w := httptest.NewRecorder()
	err := m.Set(w, "none-secure-test", "value")
	require.NoError(t, err)

	cookieStr := w.Header().Get("Set-Cookie")
	assert.Contains(t, cookieStr, "SameSite=None", "Cookie should have SameSite=None")
	assert.Contains(t, cookieStr, "Secure", "Cookie with SameSite=None should also be Secure")
}

func TestDomainAttributeSecurity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		domain   string
		expected string
	}{
		{"explicit_domain", "example.com", "Domain=example.com"},
		{"subdomain", "app.example.com", "Domain=app.example.com"},
		{"localhost", "localhost", "Domain=localhost"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, _ := cookie.New(
				[]string{"this-is-a-very-long-secret-key-32-chars-long"},
				cookie.WithDomain(tc.domain),
			)

			w := httptest.NewRecorder()
			err := m.Set(w, "domain-test", "value")
			require.NoError(t, err)

			cookieStr := w.Header().Get("Set-Cookie")
			assert.Contains(t, cookieStr, tc.expected, "Cookie should have correct Domain attribute")
		})
	}
}

func TestPathAttributeSecurity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		path     string
		expected string
	}{
		{"root_path", "/", "Path=/"},
		{"app_path", "/app", "Path=/app"},
		{"nested_path", "/app/admin", "Path=/app/admin"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, _ := cookie.New(
				[]string{"this-is-a-very-long-secret-key-32-chars-long"},
				cookie.WithPath(tc.path),
			)

			w := httptest.NewRecorder()
			err := m.Set(w, "path-test", "value")
			require.NoError(t, err)

			cookieStr := w.Header().Get("Set-Cookie")
			assert.Contains(t, cookieStr, tc.expected, "Cookie should have correct Path attribute")
		})
	}
}

func TestMaxAgeAttributeSecurity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		maxAge   int
		expected string
	}{
		{"session_cookie", 0, ""}, // Session cookie when MaxAge=0
		{"one_hour", 3600, "Max-Age=3600"},
		{"one_day", 86400, "Max-Age=86400"},
		{"negative_delete", -1, "Max-Age=0"}, // Go normalizes -1 to 0 for deletion
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, _ := cookie.New([]string{"this-is-a-very-long-secret-key-32-chars-long"})

			w := httptest.NewRecorder()
			var err error
			if tc.maxAge == 0 {
				err = m.Set(w, "maxage-test", "value")
			} else {
				err = m.Set(w, "maxage-test", "value", cookie.WithMaxAge(tc.maxAge))
			}
			require.NoError(t, err)

			cookieStr := w.Header().Get("Set-Cookie")

			if tc.expected == "" {
				assert.NotContains(t, cookieStr, "Max-Age=", "Session cookie should not have Max-Age")
			} else {
				assert.Contains(t, cookieStr, tc.expected, "Cookie should have correct Max-Age attribute")
			}
		})
	}
}

func TestDeleteCookieSecurityAttributes(t *testing.T) {
	t.Parallel()

	m, _ := cookie.New(
		[]string{"this-is-a-very-long-secret-key-32-chars-long"},
		cookie.WithDomain("example.com"),
		cookie.WithPath("/app"),
		cookie.WithSecure(true),
		cookie.WithSameSite(http.SameSiteStrictMode),
	)

	w := httptest.NewRecorder()
	m.Delete(w, "delete-test")

	cookieStr := w.Header().Get("Set-Cookie")
	require.NotEmpty(t, cookieStr)

	// Deleted cookies must maintain security attributes to ensure proper removal
	expectedAttributes := []string{
		"Domain=example.com",
		"Path=/app",
		"Secure",
		"HttpOnly",
		"SameSite=Strict",
		"Max-Age=-1",
	}

	for _, attr := range expectedAttributes {
		if attr == "Max-Age=-1" {
			if !strings.Contains(cookieStr, "Max-Age=-1") && !strings.Contains(cookieStr, "Max-Age=0") {
				t.Errorf("Delete cookie should have Max-Age=-1 or Max-Age=0, got: %s", cookieStr)
			}
		} else {
			assert.Contains(t, cookieStr, attr, "Delete cookie should maintain %s attribute", attr)
		}
	}
}
