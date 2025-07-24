package i18n_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/i18n"

	"github.com/stretchr/testify/assert"
)

func TestParseAcceptLanguage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		header         string
		supportedLangs []string
		defaultLang    string
		expected       string
	}{
		{
			name:           "empty header returns default",
			header:         "",
			supportedLangs: []string{"en", "fr", "de"},
			defaultLang:    "en",
			expected:       "en",
		},
		{
			name:           "exact match",
			header:         "fr",
			supportedLangs: []string{"en", "fr", "de"},
			defaultLang:    "en",
			expected:       "fr",
		},
		{
			name:           "region variant matches base language",
			header:         "fr-CA",
			supportedLangs: []string{"en", "fr", "de"},
			defaultLang:    "en",
			expected:       "fr",
		},
		{
			name:           "quality values respected",
			header:         "en;q=0.5,fr;q=0.9,de;q=0.8",
			supportedLangs: []string{"en", "fr", "de"},
			defaultLang:    "en",
			expected:       "fr",
		},
		{
			name:           "unsupported language falls back to default",
			header:         "ja,ko",
			supportedLangs: []string{"en", "fr", "de"},
			defaultLang:    "en",
			expected:       "en",
		},
		{
			name:           "complex header with multiple matches",
			header:         "ja;q=0.9,en-US;q=0.8,fr;q=0.7",
			supportedLangs: []string{"en", "fr", "de"},
			defaultLang:    "de",
			expected:       "fr", // Exact match preferred over base language fallback
		},
		{
			name:           "whitespace handling",
			header:         "  en-US  ,  fr;q=0.9  ",
			supportedLangs: []string{"en", "fr"},
			defaultLang:    "de",
			expected:       "fr", // fr is exact match with q=0.9, en-US needs fallback to en
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := i18n.ParseAcceptLanguage(tt.header, tt.supportedLangs, tt.defaultLang)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultLangExtractor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		setupRequest func(*http.Request)
		options      []i18n.ExtractorOption
		expected     string
	}{
		{
			name: "cookie takes precedence",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "lang", Value: "fr"})
				req.URL.RawQuery = "lang=de"
				req.Header.Set("Language", "es")
				req.Header.Set("Accept-Language", "en")
			},
			expected: "fr",
		},
		{
			name: "query parameter when no cookie",
			setupRequest: func(req *http.Request) {
				req.URL.RawQuery = "lang=de"
				req.Header.Set("Language", "es")
				req.Header.Set("Accept-Language", "en")
			},
			expected: "de",
		},
		{
			name: "Language header when no cookie or query",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Language", "es")
				req.Header.Set("Accept-Language", "en")
			},
			expected: "es",
		},
		{
			name: "Accept-Language header as last resort",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Accept-Language", "en-US,en;q=0.9")
			},
			expected: "en-us",
		},
		{
			name: "custom cookie name",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "locale", Value: "fr"})
				req.AddCookie(&http.Cookie{Name: "lang", Value: "de"})
			},
			options:  []i18n.ExtractorOption{i18n.WithCookieName("locale")},
			expected: "fr",
		},
		{
			name: "custom query parameter",
			setupRequest: func(req *http.Request) {
				req.URL.RawQuery = "locale=fr&lang=de"
			},
			options:  []i18n.ExtractorOption{i18n.WithQueryParamName("locale")},
			expected: "fr",
		},
		{
			name: "validation with supported languages",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "lang", Value: "invalid"})
				req.URL.RawQuery = "lang=fr"
			},
			options: []i18n.ExtractorOption{
				i18n.WithSupportedLanguages("en", "fr", "de"),
			},
			expected: "fr",
		},
		{
			name: "validation rejects unsupported language",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "lang", Value: "ja"})
				req.Header.Set("Accept-Language", "en")
			},
			options: []i18n.ExtractorOption{
				i18n.WithSupportedLanguages("en", "fr", "de"),
			},
			expected: "en",
		},
		{
			name: "empty result when no language found",
			setupRequest: func(req *http.Request) {
				// No language information
			},
			expected: "",
		},
		{
			name: "whitespace trimming",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "lang", Value: "  fr  "})
			},
			expected: "fr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest("GET", "/", nil)
			tt.setupRequest(req)

			extractor := i18n.DefaultLangExtractor(tt.options...)
			result := extractor(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}
