package i18n

import (
	"slices"
	"sort"
	"strconv"
	"strings"
)

// DefaultLanguage is the default language code used when no language is detected
const DefaultLanguage = "en"

// maxAcceptLanguageLength is the maximum allowed length for Accept-Language header
const maxAcceptLanguageLength = 4096

// langWithQ represents a language tag with its quality value
type langWithQ struct {
	lang string
	q    float64
}

// parseAcceptLanguageHeader parses an Accept-Language header into a slice of languages with quality values.
// It handles the parsing of quality values and returns the languages sorted by quality (highest first).
func parseAcceptLanguageHeader(header string) []langWithQ {
	if header == "" {
		return nil
	}

	// Limit header length to prevent DoS
	if len(header) > maxAcceptLanguageLength {
		header = header[:maxAcceptLanguageLength]
	}

	var languages []langWithQ

	for part := range strings.SplitSeq(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split by semicolon to separate language from quality
		langAndQ := strings.Split(part, ";")
		lang := strings.TrimSpace(langAndQ[0])
		lang = strings.ToLower(lang) // Normalize to lowercase for consistency
		q := 1.0                     // Default quality

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

	// Sort by quality (highest first)
	sort.Slice(languages, func(i, j int) bool {
		return languages[i].q > languages[j].q
	})

	return languages
}

// ParseAcceptLanguage parses an Accept-Language header and returns the best matching language
// from the list of supported languages. If no match is found, returns the defaultLang.
// It handles quality values and language variants (e.g., "en-US" matching "en").
func ParseAcceptLanguage(header string, supportedLangs []string, defaultLang string) string {
	if header == "" || len(supportedLangs) == 0 {
		return defaultLang
	}

	// Normalize supported languages to lowercase for case-insensitive matching
	normalizedSupported := make([]string, len(supportedLangs))
	for i, lang := range supportedLangs {
		normalizedSupported[i] = strings.ToLower(lang)
	}

	// Parse Accept-Language header
	languages := parseAcceptLanguageHeader(header)

	// Find the first matching supported language
	// Process in quality order, checking exact match then base language
	for _, lq := range languages {
		// Check for exact match first
		if slices.Contains(normalizedSupported, lq.lang) {
			return lq.lang
		}
	}

	// No exact matches found, now check for base language matches
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
