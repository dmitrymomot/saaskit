package i18n_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/i18n"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTranslator implements the translator interface for testing
type mockTranslator struct {
	langResponse string
	langCalls    []langCall
}

type langCall struct {
	header        string
	defaultLocale []string
}

func (m *mockTranslator) Lang(header string, defaultLocale ...string) string {
	m.langCalls = append(m.langCalls, langCall{
		header:        header,
		defaultLocale: defaultLocale,
	})
	return m.langResponse
}

func TestSetLocale(t *testing.T) {
	t.Run("sets locale in context", func(t *testing.T) {
		ctx := context.Background()
		locale := "fr"

		newCtx := i18n.SetLocale(ctx, locale)

		assert.NotEqual(t, ctx, newCtx, "Should return a new context")
		retrievedLocale := i18n.GetLocale(newCtx)
		assert.Equal(t, locale, retrievedLocale, "Should retrieve the same locale that was set")
	})

	t.Run("overwrites existing locale", func(t *testing.T) {
		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "en")
		ctx = i18n.SetLocale(ctx, "fr")

		retrievedLocale := i18n.GetLocale(ctx)
		assert.Equal(t, "fr", retrievedLocale, "Should retrieve the last set locale")
	})

}

func TestGetLocale(t *testing.T) {
	t.Run("returns default when no locale set", func(t *testing.T) {
		ctx := context.Background()

		locale := i18n.GetLocale(ctx)
		assert.Equal(t, "en", locale, "Should return default locale 'en' when none is set")
	})

	t.Run("returns set locale", func(t *testing.T) {
		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "es")

		locale := i18n.GetLocale(ctx)
		assert.Equal(t, "es", locale, "Should return the set locale")
	})

	t.Run("handles empty string locale", func(t *testing.T) {
		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "") // Empty string

		locale := i18n.GetLocale(ctx)
		assert.Equal(t, "en", locale, "Should return default locale when value is empty string")
	})

	t.Run("handles context without locale", func(t *testing.T) {
		ctx := context.Background()
		// Don't set any locale in context

		locale := i18n.GetLocale(ctx)
		assert.Equal(t, "en", locale, "Should return default locale when no locale is set")
	})
}

func TestMiddleware(t *testing.T) {
	t.Run("uses default extractor when none provided", func(t *testing.T) {
		// Arrange
		middleware := i18n.Middleware(nil)

		// Create a test handler that checks the locale in context
		var capturedLocale string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedLocale = i18n.GetLocale(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		// Create request with Accept-Language header
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en;q=0.8")
		rec := httptest.NewRecorder()

		// Act
		middleware(handler).ServeHTTP(rec, req)

		// Assert
		assert.Equal(t, "fr-fr", capturedLocale, "Should extract language from Accept-Language header")
	})

	t.Run("uses extractor when provided", func(t *testing.T) {
		// Arrange
		// Custom extractor that returns German
		extractor := func(r *http.Request) string {
			return "de"
		}

		middleware := i18n.Middleware(extractor)

		// Create a test handler that checks the locale in context
		var capturedLocale string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedLocale = i18n.GetLocale(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		// Create request
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en;q=0.8")
		rec := httptest.NewRecorder()

		// Act
		middleware(handler).ServeHTTP(rec, req)

		// Assert
		assert.Equal(t, "de", capturedLocale, "Should use language from extractor")
	})

	t.Run("falls back to 'en' when extractor returns empty", func(t *testing.T) {
		// Arrange
		// Extractor that returns empty string
		extractor := func(r *http.Request) string {
			return ""
		}

		middleware := i18n.Middleware(extractor)

		// Create a test handler that checks the locale in context
		var capturedLocale string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedLocale = i18n.GetLocale(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		// Create request
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Language", "es-ES,es;q=0.9")
		rec := httptest.NewRecorder()

		// Act
		middleware(handler).ServeHTTP(rec, req)

		// Assert
		assert.Equal(t, "en", capturedLocale, "Should fall back to 'en' when extractor returns empty")
	})

	t.Run("handles missing Accept-Language header", func(t *testing.T) {
		// Arrange
		middleware := i18n.Middleware(nil)

		// Create a test handler that checks the locale in context
		var capturedLocale string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedLocale = i18n.GetLocale(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		// Create request without Accept-Language header
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		// Act
		middleware(handler).ServeHTTP(rec, req)

		// Assert
		assert.Equal(t, "en", capturedLocale, "Should use default language 'en'")
	})

	t.Run("preserves original request context", func(t *testing.T) {
		// Arrange
		extractor := func(r *http.Request) string {
			return "ja"
		}
		middleware := i18n.Middleware(extractor)

		// Add some value to the original context
		originalKey := struct{ name string }{name: "test"}
		originalValue := "test-value"
		ctx := context.WithValue(context.Background(), originalKey, originalValue)

		// Create a test handler that checks both original and new context values
		var capturedLocale string
		var capturedOriginalValue string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedLocale = i18n.GetLocale(r.Context())
			if val, ok := r.Context().Value(originalKey).(string); ok {
				capturedOriginalValue = val
			}
			w.WriteHeader(http.StatusOK)
		})

		// Create request with context
		req := httptest.NewRequest("GET", "/test", nil)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		// Act
		middleware(handler).ServeHTTP(rec, req)

		// Assert
		assert.Equal(t, "ja", capturedLocale, "Should set new locale")
		assert.Equal(t, "test-value", capturedOriginalValue, "Should preserve original context values")
	})
}

func TestMiddlewareWithRealTranslator(t *testing.T) {
	// Create a real translator for integration testing
	translations := map[string]map[string]any{
		"en": {"hello": "Hello"},
		"fr": {"hello": "Bonjour"},
		"de": {"hello": "Hallo"},
		"es": {"hello": "Hola"},
	}

	adapter := &i18n.MapAdapter{Data: translations}
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	require.NoError(t, err)

	t.Run("integration test with real translator", func(t *testing.T) {
		// Use DefaultLangExtractor with supported languages for proper validation
		middleware := i18n.Middleware(i18n.DefaultLangExtractor(
			i18n.WithSupportedLanguages(translator.SupportedLanguages()...),
		))

		// Create a test handler that uses context-based translation
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			greeting := translator.Tc(r.Context(), "hello")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(greeting))
		})

		tests := []struct {
			name           string
			acceptLanguage string
			expectedBody   string
		}{
			{
				name:           "French preference",
				acceptLanguage: "fr-FR,fr;q=0.9,en;q=0.8",
				expectedBody:   "Bonjour",
			},
			{
				name:           "German preference",
				acceptLanguage: "de-DE,de;q=0.9,en;q=0.8",
				expectedBody:   "Hallo",
			},
			{
				name:           "Spanish preference",
				acceptLanguage: "es-ES,es;q=0.9,en;q=0.8",
				expectedBody:   "Hola",
			},
			{
				name:           "Unsupported language falls back to English",
				acceptLanguage: "zh-CN,zh;q=0.9",
				expectedBody:   "Hello",
			},
			{
				name:           "No Accept-Language header",
				acceptLanguage: "",
				expectedBody:   "Hello",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", "/test", nil)
				if tt.acceptLanguage != "" {
					req.Header.Set("Accept-Language", tt.acceptLanguage)
				}
				rec := httptest.NewRecorder()

				middleware(handler).ServeHTTP(rec, req)

				assert.Equal(t, http.StatusOK, rec.Code)
				assert.Equal(t, tt.expectedBody, rec.Body.String())
			})
		}
	})

	t.Run("custom extractor integration test", func(t *testing.T) {
		// Custom extractor that uses query parameter
		extractor := func(r *http.Request) string {
			return r.URL.Query().Get("lang")
		}

		middleware := i18n.Middleware(extractor)

		// Create a test handler that uses context-based translation
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			greeting := translator.Tc(r.Context(), "hello")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(greeting))
		})

		tests := []struct {
			name         string
			url          string
			expectedBody string
		}{
			{
				name:         "French from query parameter",
				url:          "/test?lang=fr",
				expectedBody: "Bonjour",
			},
			{
				name:         "German from query parameter",
				url:          "/test?lang=de",
				expectedBody: "Hallo",
			},
			{
				name:         "No language parameter falls back to default",
				url:          "/test",
				expectedBody: "Hello", // Falls back to 'en'
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", tt.url, nil)
				req.Header.Set("Accept-Language", "es-ES,es;q=0.9,en;q=0.8")
				rec := httptest.NewRecorder()

				middleware(handler).ServeHTTP(rec, req)

				assert.Equal(t, http.StatusOK, rec.Code)
				assert.Equal(t, tt.expectedBody, rec.Body.String())
			})
		}
	})
}

func TestMiddlewareChaining(t *testing.T) {
	t.Run("chains with other middleware", func(t *testing.T) {
		// Create an extractor that returns French
		extractor := func(r *http.Request) string {
			return "fr"
		}
		i18nMiddleware := i18n.Middleware(extractor)

		// Create a logging middleware for testing
		var loggedPath string
		loggingMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				loggedPath = r.URL.Path
				next.ServeHTTP(w, r)
			})
		}

		// Create final handler
		var capturedLocale string
		finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedLocale = i18n.GetLocale(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		// Chain middleware: logging -> i18n -> final handler
		handler := loggingMiddleware(i18nMiddleware(finalHandler))

		// Create request
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Accept-Language", "fr")
		rec := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rec, req)

		// Assert
		assert.Equal(t, "/api/test", loggedPath, "Logging middleware should work")
		assert.Equal(t, "fr", capturedLocale, "I18n middleware should work")
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
