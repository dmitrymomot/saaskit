package i18n_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/i18n"

	"github.com/stretchr/testify/assert"
)

func TestExtractorOptions(t *testing.T) {
	t.Parallel()
	t.Run("WithCookieName", func(t *testing.T) {
		t.Parallel()
		extractor := i18n.DefaultLangExtractor(i18n.WithCookieName("custom_lang"))

		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: "custom_lang", Value: "fr"})
		req.AddCookie(&http.Cookie{Name: "lang", Value: "de"})

		result := extractor(req)
		assert.Equal(t, "fr", result)
	})

	t.Run("WithQueryParamName", func(t *testing.T) {
		t.Parallel()
		extractor := i18n.DefaultLangExtractor(i18n.WithQueryParamName("locale"))

		req := httptest.NewRequest("GET", "/?locale=es&lang=fr", nil)

		result := extractor(req)
		assert.Equal(t, "es", result)
	})

	t.Run("WithSupportedLanguages", func(t *testing.T) {
		t.Parallel()
		extractor := i18n.DefaultLangExtractor(i18n.WithSupportedLanguages("en", "fr", "de"))

		tests := []struct {
			name     string
			setup    func(*http.Request)
			expected string
		}{
			{
				name: "valid language passes through",
				setup: func(req *http.Request) {
					req.AddCookie(&http.Cookie{Name: "lang", Value: "fr"})
				},
				expected: "fr",
			},
			{
				name: "invalid language rejected",
				setup: func(req *http.Request) {
					req.AddCookie(&http.Cookie{Name: "lang", Value: "ja"})
				},
				expected: "",
			},
			{
				name: "region variant fallback to base",
				setup: func(req *http.Request) {
					req.AddCookie(&http.Cookie{Name: "lang", Value: "fr-CA"})
				},
				expected: "fr",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest("GET", "/", nil)
				tt.setup(req)
				result := extractor(req)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("multiple options combined", func(t *testing.T) {
		t.Parallel()
		extractor := i18n.DefaultLangExtractor(
			i18n.WithCookieName("locale"),
			i18n.WithQueryParamName("l"),
			i18n.WithSupportedLanguages("en", "es", "pt"),
		)

		req := httptest.NewRequest("GET", "/?l=es", nil)
		req.AddCookie(&http.Cookie{Name: "locale", Value: "pt"})

		result := extractor(req)
		assert.Equal(t, "pt", result) // Cookie takes precedence
	})
}

func TestDefaultLangExtractorPriority(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		setup    func(*http.Request)
		expected string
	}{
		{
			name: "cookie highest priority",
			setup: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "lang", Value: "fr"})
				req.URL.RawQuery = "lang=de"
				req.Header.Set("Language", "es")
				req.Header.Set("Accept-Language", "en")
			},
			expected: "fr",
		},
		{
			name: "query param second priority",
			setup: func(req *http.Request) {
				req.URL.RawQuery = "lang=de"
				req.Header.Set("Language", "es")
				req.Header.Set("Accept-Language", "en")
			},
			expected: "de",
		},
		{
			name: "Language header third priority",
			setup: func(req *http.Request) {
				req.Header.Set("Language", "es")
				req.Header.Set("Accept-Language", "en")
			},
			expected: "es",
		},
		{
			name: "Accept-Language lowest priority",
			setup: func(req *http.Request) {
				req.Header.Set("Accept-Language", "en-US,en;q=0.9")
			},
			expected: "en-us",
		},
		{
			name: "empty when no language found",
			setup: func(req *http.Request) {
				// No language information
			},
			expected: "",
		},
	}

	extractor := i18n.DefaultLangExtractor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest("GET", "/", nil)
			tt.setup(req)
			result := extractor(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultLangExtractorWhitespaceHandling(t *testing.T) {
	extractor := i18n.DefaultLangExtractor()

	tests := []struct {
		name     string
		setup    func(*http.Request)
		expected string
	}{
		{
			name: "cookie with spaces",
			setup: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "lang", Value: "  fr  "})
			},
			expected: "fr",
		},
		{
			name: "query param with spaces",
			setup: func(req *http.Request) {
				req.URL.RawQuery = "lang=%20%20de%20%20" // URL encoded spaces
			},
			expected: "de",
		},
		{
			name: "Language header with spaces",
			setup: func(req *http.Request) {
				req.Header.Set("Language", "  es  ")
			},
			expected: "es",
		},
		{
			name: "empty string after trimming",
			setup: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "lang", Value: "   "})
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest("GET", "/", nil)
			tt.setup(req)
			result := extractor(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultLangExtractorValidation(t *testing.T) {
	t.Parallel()
	supported := []string{"en", "fr", "de", "es"}
	extractor := i18n.DefaultLangExtractor(i18n.WithSupportedLanguages(supported...))

	tests := []struct {
		name     string
		setup    func(*http.Request)
		expected string
	}{
		{
			name: "exact match in cookie",
			setup: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "lang", Value: "fr"})
			},
			expected: "fr",
		},
		{
			name: "region variant in cookie falls back to base",
			setup: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "lang", Value: "fr-CA"})
			},
			expected: "fr",
		},
		{
			name: "unsupported in cookie falls through to query",
			setup: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "lang", Value: "ja"})
				req.URL.RawQuery = "lang=de"
			},
			expected: "de",
		},
		{
			name: "Accept-Language with validation",
			setup: func(req *http.Request) {
				req.Header.Set("Accept-Language", "ja;q=0.9,fr;q=0.8,en;q=0.7")
			},
			expected: "fr", // First supported language
		},
		{
			name: "all sources invalid returns empty",
			setup: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "lang", Value: "ja"})
				req.URL.RawQuery = "lang=ko"
				req.Header.Set("Language", "zh")
				req.Header.Set("Accept-Language", "ja,ko,zh")
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest("GET", "/", nil)
			tt.setup(req)
			result := extractor(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultLangExtractorEdgeCases(t *testing.T) {
	t.Parallel()
	t.Run("empty cookie value", func(t *testing.T) {
		t.Parallel()
		extractor := i18n.DefaultLangExtractor()
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: "lang", Value: ""})
		req.URL.RawQuery = "lang=fr"

		result := extractor(req)
		assert.Equal(t, "fr", result) // Falls through to query param
	})

	t.Run("cookie error handling", func(t *testing.T) {
		t.Parallel()
		extractor := i18n.DefaultLangExtractor()
		req := httptest.NewRequest("GET", "/", nil)
		// No cookie set, should not panic
		result := extractor(req)
		assert.Equal(t, "", result)
	})

	t.Run("complex Accept-Language without validation", func(t *testing.T) {
		t.Parallel()
		extractor := i18n.DefaultLangExtractor()
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "en-US;q=0.8,en;q=0.7,fr-FR;q=0.9")

		result := extractor(req)
		assert.Equal(t, "fr-fr", result) // Highest quality value
	})

	t.Run("malformed Accept-Language header", func(t *testing.T) {
		t.Parallel()
		extractor := i18n.DefaultLangExtractor()
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "invalid;;q=abc,en")

		result := extractor(req)
		assert.NotEmpty(t, result) // Should handle gracefully
	})

	t.Run("case sensitivity", func(t *testing.T) {
		t.Parallel()
		// Test without validation - case is preserved
		extractor1 := i18n.DefaultLangExtractor()
		req1 := httptest.NewRequest("GET", "/", nil)
		req1.AddCookie(&http.Cookie{Name: "lang", Value: "FR"})

		result1 := extractor1(req1)
		assert.Equal(t, "fr", result1) // Always normalized to lowercase

		// Test with validation - uppercase should match lowercase in supported list (case-insensitive)
		extractor2 := i18n.DefaultLangExtractor(i18n.WithSupportedLanguages("en", "fr"))
		req2 := httptest.NewRequest("GET", "/", nil)
		req2.AddCookie(&http.Cookie{Name: "lang", Value: "FR"})

		result2 := extractor2(req2)
		assert.Equal(t, "fr", result2) // FR should match fr in supported languages (normalized to lowercase)
	})
}

func TestDefaultLangExtractorConcurrency(t *testing.T) {
	t.Parallel()
	extractor := i18n.DefaultLangExtractor(
		i18n.WithSupportedLanguages("en", "fr", "de"),
	)

	// Run multiple goroutines to ensure thread safety
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			req := httptest.NewRequest("GET", "/", nil)
			lang := []string{"en", "fr", "de", "ja"}[id%4]
			req.AddCookie(&http.Cookie{Name: "lang", Value: lang})

			result := extractor(req)
			if lang == "ja" {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, lang, result)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestExtractorIntegrationWithMiddleware(t *testing.T) {
	t.Parallel()
	// Create a custom extractor that only checks query params
	extractor := i18n.DefaultLangExtractor(
		i18n.WithCookieName(""), // Disable cookie checking
		i18n.WithQueryParamName("locale"),
		i18n.WithSupportedLanguages("en", "es", "pt"),
	)

	// Test the extractor works as expected
	req := httptest.NewRequest("GET", "/?locale=pt", nil)
	req.AddCookie(&http.Cookie{Name: "lang", Value: "fr"}) // Should be ignored

	result := extractor(req)
	assert.Equal(t, "pt", result)

	// Test with unsupported language
	req2 := httptest.NewRequest("GET", "/?locale=fr", nil)
	result2 := extractor(req2)
	assert.Empty(t, result2)
}
