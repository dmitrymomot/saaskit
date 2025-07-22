package i18n

import (
	"net/http"
	"slices"
	"strings"
)

// maxLangCodeLength is the maximum allowed length for a language code
const maxLangCodeLength = 35 // RFC 5646 recommends 35 characters max

// langValidator validates and normalizes language codes
type langValidator struct {
	supportedLangs []string
}

// newLangValidator creates a new language validator with normalized supported languages
func newLangValidator(supportedLangs []string) *langValidator {
	normalized := make([]string, len(supportedLangs))
	for i, lang := range supportedLangs {
		normalized[i] = strings.ToLower(lang)
	}
	return &langValidator{supportedLangs: normalized}
}

// validate checks if a language code is valid and returns the normalized version
func (v *langValidator) validate(lang string) string {
	if lang == "" {
		return ""
	}

	// Limit language code length for security
	if len(lang) > maxLangCodeLength {
		return ""
	}

	// Always normalize to lowercase for consistency
	normalizedLang := strings.ToLower(lang)

	// If no validation required, return normalized
	if len(v.supportedLangs) == 0 {
		return normalizedLang
	}

	// Check if language is supported
	if slices.Contains(v.supportedLangs, normalizedLang) {
		return normalizedLang
	}
	// Check without region code
	if idx := strings.Index(normalizedLang, "-"); idx > 0 {
		baseLang := normalizedLang[:idx]
		if slices.Contains(v.supportedLangs, baseLang) {
			return baseLang
		}
	}
	return ""
}

// ExtractorConfig holds configuration for the language extractor
type ExtractorConfig struct {
	CookieName     string
	QueryParamName string
	SupportedLangs []string
}

// ExtractorOption configures the language extractor
type ExtractorOption func(*ExtractorConfig)

// WithCookieName sets the cookie name to check for language preference
func WithCookieName(name string) ExtractorOption {
	return func(c *ExtractorConfig) {
		if name == "" {
			return
		}
		c.CookieName = name
	}
}

// WithQueryParamName sets the query parameter name to check for language
func WithQueryParamName(name string) ExtractorOption {
	return func(c *ExtractorConfig) {
		if name == "" {
			return
		}
		c.QueryParamName = name
	}
}

// WithSupportedLanguages sets the list of supported languages for validation
func WithSupportedLanguages(langs ...string) ExtractorOption {
	return func(c *ExtractorConfig) {
		if len(langs) == 0 {
			return
		}
		c.SupportedLangs = langs
	}
}

// DefaultLangExtractor creates a language extractor that checks multiple sources in priority order:
// 1. Cookie (default name: "lang")
// 2. Query parameter (default name: "lang")
// 3. Language header
// 4. Accept-Language header
//
// The extractor returns the first non-empty language code found.
// If SupportedLangs is provided, it will validate the language.
// For Accept-Language headers, it uses ParseAcceptLanguage to find the best match.
func DefaultLangExtractor(opts ...ExtractorOption) LangExtractor {
	config := &ExtractorConfig{
		CookieName:     "lang",
		QueryParamName: "lang",
		SupportedLangs: nil,
	}

	for _, opt := range opts {
		opt(config)
	}

	// Create validator once at initialization time
	validator := newLangValidator(config.SupportedLangs)

	return func(r *http.Request) string {

		// 1. Check cookie
		if config.CookieName != "" {
			if cookie, err := r.Cookie(config.CookieName); err == nil && cookie.Value != "" {
				if lang := strings.TrimSpace(cookie.Value); lang != "" {
					if validated := validator.validate(lang); validated != "" {
						return validated
					}
				}
			}
		}

		// 2. Check query parameter
		if config.QueryParamName != "" {
			if lang := strings.TrimSpace(r.URL.Query().Get(config.QueryParamName)); lang != "" {
				if validated := validator.validate(lang); validated != "" {
					return validated
				}
			}
		}

		// 3. Check Language header (non-standard but sometimes used)
		if lang := strings.TrimSpace(r.Header.Get("Language")); lang != "" {
			if validated := validator.validate(lang); validated != "" {
				return validated
			}
		}

		// 4. Check Accept-Language header
		acceptLang := r.Header.Get("Accept-Language")
		if acceptLang != "" {
			if len(config.SupportedLangs) > 0 {
				// Parse and find best match
				return ParseAcceptLanguage(acceptLang, config.SupportedLangs, "")
			}
			// Return the highest priority language without validation
			langs := parseAcceptLanguageHeader(acceptLang)
			if len(langs) > 0 {
				return langs[0].lang
			}
			return ""
		}

		// Return empty string to let the middleware handle the default
		return ""
	}
}
