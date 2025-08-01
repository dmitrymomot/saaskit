package i18n

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
)

// DefaultLanguage is the default language code used when no language is detected
const DefaultLanguage = "en"

// maxAcceptLanguageLength prevents DoS attacks through oversized Accept-Language headers.
// RFC 7231 doesn't specify a limit, but 4KB is generous for legitimate headers while
// preventing memory exhaustion from malicious requests.
const maxAcceptLanguageLength = 4096

// langWithQ represents a language tag with its quality value
type langWithQ struct {
	lang string
	q    float64
}

// parseAcceptLanguageHeader parses Accept-Language headers according to RFC 7231.
// Uses quality values to prioritize user preferences, handling malformed entries gracefully.
// Truncates oversized headers to prevent DoS while preserving most user preferences.
func parseAcceptLanguageHeader(header string) []langWithQ {
	if header == "" {
		return nil
	}

	if len(header) > maxAcceptLanguageLength {
		header = header[:maxAcceptLanguageLength]
	}

	var languages []langWithQ

	for part := range strings.SplitSeq(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		langAndQ := strings.Split(part, ";")
		lang := strings.TrimSpace(langAndQ[0])
		lang = strings.ToLower(lang) // Case-insensitive matching per RFC 7231
		q := 1.0

		// Parse quality value if present
		if len(langAndQ) > 1 {
			qPart := strings.TrimSpace(langAndQ[1])
			if strings.HasPrefix(qPart, "q=") {
				if qVal, err := strconv.ParseFloat(qPart[2:], 64); err == nil && qVal >= 0 && qVal <= 1 {
					q = qVal
				}
			}
		}

		if lang != "" {
			languages = append(languages, langWithQ{lang: lang, q: q})
		}
	}

	// Sort by quality score descending to respect user preferences
	slices.SortFunc(languages, func(a, b langWithQ) int {
		return cmp.Compare(b.q, a.q) // Reversed for descending order
	})

	return languages
}

// ParseAcceptLanguage implements RFC 7231 Accept-Language negotiation with fallback strategy.
// First attempts exact matches (en-US), then base language matches (en-US -> en).
// This two-phase approach balances user preferences with practical language support.
func ParseAcceptLanguage(header string, supportedLangs []string, defaultLang string) string {
	if header == "" || len(supportedLangs) == 0 {
		return defaultLang
	}

	normalizedSupported := make([]string, len(supportedLangs))
	for i, lang := range supportedLangs {
		normalizedSupported[i] = strings.ToLower(lang)
	}

	languages := parseAcceptLanguageHeader(header)

	// Phase 1: Exact matches (en-US matches en-US)
	for _, lq := range languages {
		if slices.Contains(normalizedSupported, lq.lang) {
			return lq.lang
		}
	}

	// Phase 2: Base language fallback (en-US matches en)
	// Only after all exact matches are exhausted to respect quality ordering
	for _, lq := range languages {
		if idx := strings.Index(lq.lang, "-"); idx > 0 {
			baseLang := lq.lang[:idx]
			if slices.Contains(normalizedSupported, baseLang) {
				return baseLang
			}
		}
	}

	return defaultLang
}
