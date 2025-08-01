package i18n

import (
	"net/http"
	"slices"
	"strings"
)

// maxLangCodeLength enforces RFC 5646 compliance and prevents buffer overflow attacks.
// Language tags longer than 35 characters are malformed or malicious (legitimate tags
// like "zh-Hans-CN-x-private-tag" are well under this limit).
const maxLangCodeLength = 35

// langValidator validates and normalizes language codes
type langValidator struct {
	supportedLangs []string
}

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

	if len(lang) > maxLangCodeLength {
		return ""
	}

	normalizedLang := strings.ToLower(lang)

	if len(v.supportedLangs) == 0 {
		return normalizedLang
	}

	// Exact match first (en-US matches en-US)
	if slices.Contains(v.supportedLangs, normalizedLang) {
		return normalizedLang
	}

	// Fallback to base language (en-US matches en)
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

// DefaultLangExtractor implements security-conscious language detection with fallback hierarchy.
// Priority order reflects security vs usability: explicit user choice (cookie, query) before
// implicit browser preferences (headers). This prevents language injection while respecting
// user preferences. Cookie takes precedence as it represents persistent user choice.
//
// Priority order:
// 1. Cookie - explicit user preference, persistent across sessions
// 2. Query parameter - explicit per-request override
// 3. Language header - non-standard but sometimes used by APIs
// 4. Accept-Language header - browser preferences with quality values
func DefaultLangExtractor(opts ...ExtractorOption) LangExtractor {
	config := &ExtractorConfig{
		CookieName:     "lang",
		QueryParamName: "lang",
		SupportedLangs: nil,
	}

	for _, opt := range opts {
		opt(config)
	}

	validator := newLangValidator(config.SupportedLangs)

	return func(r *http.Request) string {
		// 1. Cookie - persistent user preference
		if config.CookieName != "" {
			if cookie, err := r.Cookie(config.CookieName); err == nil && cookie.Value != "" {
				if lang := strings.TrimSpace(cookie.Value); lang != "" {
					if validated := validator.validate(lang); validated != "" {
						return validated
					}
				}
			}
		}

		// 2. Query parameter - per-request override
		if config.QueryParamName != "" {
			if lang := strings.TrimSpace(r.URL.Query().Get(config.QueryParamName)); lang != "" {
				if validated := validator.validate(lang); validated != "" {
					return validated
				}
			}
		}

		// 3. Language header - non-standard but used by some APIs
		if lang := strings.TrimSpace(r.Header.Get("Language")); lang != "" {
			if validated := validator.validate(lang); validated != "" {
				return validated
			}
		}

		// 4. Accept-Language header - browser preferences
		acceptLang := r.Header.Get("Accept-Language")
		if acceptLang != "" {
			if len(config.SupportedLangs) > 0 {
				return ParseAcceptLanguage(acceptLang, config.SupportedLangs, "")
			}
			// Return highest quality language without validation
			langs := parseAcceptLanguageHeader(acceptLang)
			if len(langs) > 0 {
				return langs[0].lang
			}
			return ""
		}

		return ""
	}
}
