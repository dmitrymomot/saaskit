package i18n

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Custom error types for the i18n package

// ErrLanguageNotSupported indicates that the requested language is not available
type ErrLanguageNotSupported struct {
	Lang string
}

func (e *ErrLanguageNotSupported) Error() string {
	return fmt.Sprintf("language not supported: %s", e.Lang)
}

// Translator represents a struct that handles translation functionality.
// It uses an adapter to load translations from various sources.
type Translator struct {
	translations   map[string]map[string]any
	defaultLang    string
	fallbackToKey  bool
	missingLogMode bool
	logger         *slog.Logger
	mu             sync.RWMutex
	adapter        TranslationAdapter
}

// NewTranslator creates a new Translator instance with the given adapter and options.
func NewTranslator(ctx context.Context, adapter TranslationAdapter, options ...Option) (*Translator, error) {
	if adapter == nil {
		return nil, fmt.Errorf("adapter is nil")
	}

	t := &Translator{
		defaultLang:    DefaultLanguage,
		fallbackToKey:  true,
		missingLogMode: false,
		logger:         slog.New(slog.NewTextHandler(io.Discard, nil)), // Nope-logger by default
		adapter:        adapter,
	}

	// Apply options
	for _, option := range options {
		option(t)
	}

	// Load translations from adapter
	translations, err := adapter.Load(ctx)
	if err != nil {
		return nil, err
	}

	// Validate translations
	if err := t.validateTranslations(translations); err != nil {
		return nil, err
	}

	t.translations = translations
	t.logger.InfoContext(ctx, "Translations loaded", "languages", t.supportedLanguages())
	return t, nil
}

// validateTranslations checks if the translations map has a valid structure.
// It ensures that language codes are valid and that translations are properly formatted.
func (t *Translator) validateTranslations(trans map[string]map[string]any) error {
	if len(trans) == 0 {
		t.logger.Warn("No translations provided")
		return nil
	}

	for lang, translations := range trans {
		if lang == "" {
			return fmt.Errorf("empty language code found")
		}
		if translations == nil {
			return fmt.Errorf("nil translations map for language: %s", lang)
		}
	}
	return nil
}

// supportedLanguages returns a list of language codes that have translations available.
func (t *Translator) supportedLanguages() []string {
	langs := make([]string, 0, len(t.translations))
	for lang := range t.translations {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	return langs
}

// SupportedLanguages returns a list of language codes that have translations available.
func (t *Translator) SupportedLanguages() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.supportedLanguages()
}

// getTranslation traverses a nested map using dot-separated keys.
// For example, key "datetime.days.other" will traverse m["datetime"] then ["days"] then ["other"].
func (t *Translator) getTranslation(m map[string]any, key string) (any, bool) {
	parts := strings.Split(key, ".")
	current := m

	for i, part := range parts {
		if i == len(parts)-1 {
			val, ok := current[part]
			return val, ok
		}

		next, ok := current[part]
		if !ok {
			return nil, false
		}

		currentMap, ok := next.(map[string]any)
		if !ok {
			// Try to convert from map[any]any to map[string]any
			anyMap, isAnyMap := next.(map[any]any)
			if !isAnyMap {
				return nil, false
			}

			currentMap = make(map[string]any, len(anyMap))
			for k, v := range anyMap {
				if ks, ok := k.(string); ok {
					currentMap[ks] = v
				}
			}
		}

		current = currentMap
	}

	return nil, false
}

// HasTranslation checks if a translation exists for the given language and key.
func (t *Translator) HasTranslation(lang, key string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	langMap, ok := t.translations[lang]
	if !ok {
		return false
	}

	_, ok = t.getTranslation(langMap, key)
	return ok
}

// buildParams converts a slice of strings (expected as key, value, key, value, …)
// into a map. If the number of arguments is odd, the last one is ignored.
func (t *Translator) buildParams(args []string) map[string]string {
	params := make(map[string]string)
	for i := 0; i < len(args)-1; i += 2 {
		params[args[i]] = args[i+1]
	}
	return params
}

// sprintf always uses named substitution. It builds a parameter map from the key-value pairs.
func (t *Translator) sprintf(tmpl string, args []string) string {
	params := t.buildParams(args)
	return t.namedSprintf(tmpl, params)
}

// Regex to find named parameters in the form %{name}
var paramRegex = regexp.MustCompile(`%\{([^}]+)\}`)

// namedSprintf performs substitution of named placeholders in the form "%{key}"
// using the provided map.
func (t *Translator) namedSprintf(tmpl string, params map[string]string) string {
	result := paramRegex.ReplaceAllStringFunc(tmpl, func(match string) string {
		// Extract parameter name
		name := match[2 : len(match)-1]
		// Replace with parameter value if exists
		if val, ok := params[name]; ok {
			return val
		}
		// Keep original placeholder if parameter not found
		return match
	})
	return result
}

// T translates a key for the given language.
// It supports formatting with additional arguments provided as key-value pairs.
// For example: translator.T("en", "welcome", "name", "John") will substitute "%{name}" in the template.
//
// If the requested translation is not found and FallbackToKey is true, the function returns
// the key as a fallback. Otherwise, it returns an empty string and logs the error if
// missingLogMode is enabled.
//
// Example:
//
//	// With translation "welcome": "Hello, %{name}!"
//	msg := translator.T("en", "welcome", "name", "John")
//	// Returns: "Hello, John!"
//
//	// With nested translation using dot notation
//	msg := translator.T("en", "messages.greeting", "name", "Alice")
//	// Returns corresponding nested translation with "Alice" substituted
func (t *Translator) T(lang, key string, args ...string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Check if the language is supported
	langMap, ok := t.translations[lang]
	if !ok {
		if t.missingLogMode {
			t.logger.Warn("Language not supported", "lang", lang, "key", key)
		}
		if t.fallbackToKey {
			return t.sprintf(key, args)
		}
		return ""
	}

	// Get the translation
	val, ok := t.getTranslation(langMap, key)
	if !ok {
		if t.missingLogMode {
			t.logger.Warn("Translation not found", "lang", lang, "key", key)
		}
		if t.fallbackToKey {
			return t.sprintf(key, args)
		}
		return ""
	}

	// Handle different types of translation values
	switch v := val.(type) {
	case string:
		return t.sprintf(v, args)
	case map[string]any, map[any]any:
		if t.missingLogMode {
			t.logger.Warn("Translation is not a string", "lang", lang, "key", key, "type", fmt.Sprintf("%T", v))
		}
		if t.fallbackToKey {
			return t.sprintf(key, args)
		}
	default:
		// Try to convert to string
		if s, ok := val.(fmt.Stringer); ok {
			return t.sprintf(s.String(), args)
		}

		if t.missingLogMode {
			t.logger.Warn("Translation is not a string", "lang", lang, "key", key, "type", fmt.Sprintf("%T", v))
		}
		if t.fallbackToKey {
			return t.sprintf(key, args)
		}
	}

	return ""
}

// N translates a key with pluralization for the given language.
// The parameter n is used to select the plural form. It supports formatting with additional
// arguments provided as key-value pairs.
//
// The function first tries the exact key with the appropriate plural suffix:
// - For n=0, it tries key+".zero" first, falling back to key+".other"
// - For n=1, it tries key+".one"
// - For all other values, it uses key+".other"
//
// If no translation is found and fallbackToKey is true, it falls back to the key itself.
// Otherwise, it returns an empty string and logs the error if missingLogMode is enabled.
//
// Example:
//
//	// With translations:
//	// "items.zero": "No items"
//	// "items.one": "%{count} item"
//	// "items.other": "%{count} items"
//
//	msg := translator.N("en", "items", 0, "count", "0")
//	// Returns: "No items"
//
//	msg := translator.N("en", "items", 1, "count", "1")
//	// Returns: "1 item"
//
//	msg := translator.N("en", "items", 5, "count", "5")
//	// Returns: "5 items"
func (t *Translator) N(lang, key string, n int, args ...string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Check if the language is supported
	langMap, ok := t.translations[lang]
	if !ok {
		if t.missingLogMode {
			t.logger.Warn("Language not supported", "lang", lang, "key", key, "n", n)
		}
		if t.fallbackToKey {
			return t.sprintf(key, args)
		}
		return ""
	}

	// Try to get the translation with appropriate plural form
	var val any
	var found bool

	// For n=0, try "zero" form first
	if n == 0 {
		val, found = t.getTranslation(langMap, key+".zero")
		if found {
			goto translate
		}
		// Fall back to "other" form for n=0
		val, found = t.getTranslation(langMap, key+".other")
		if found {
			goto translate
		}
	}

	// For n=1, try "one" form
	if n == 1 {
		val, found = t.getTranslation(langMap, key+".one")
		if found {
			goto translate
		}
	}

	// For n>1, use "other" form
	if n != 0 && n != 1 {
		val, found = t.getTranslation(langMap, key+".other")
		if found {
			goto translate
		}
	}

	// Try the key itself (might be a string with embedded pluralization logic)
	val, found = t.getTranslation(langMap, key)
	if !found {
		if t.missingLogMode {
			t.logger.Warn("Pluralization not found", "lang", lang, "key", key, "n", n)
		}
		if t.fallbackToKey {
			return t.sprintf(key, args)
		}
		return ""
	}

translate:
	switch v := val.(type) {
	case string:
		// Always include the count in args if not already present
		hasCount := false
		for i := 0; i < len(args)-1; i += 2 {
			if args[i] == "count" {
				hasCount = true
				break
			}
		}
		if !hasCount {
			newArgs := make([]string, len(args)+2)
			copy(newArgs, args)
			newArgs[len(args)] = "count"
			newArgs[len(args)+1] = strconv.Itoa(n)
			args = newArgs
		}
		return t.sprintf(v, args)
	default:
		if t.missingLogMode {
			t.logger.Warn("Pluralization translation is not a string", "lang", lang, "key", key, "n", n, "type", fmt.Sprintf("%T", v))
		}
		if t.fallbackToKey {
			return t.sprintf(key, args)
		}
		return ""
	}
}

// Duration converts a time.Duration to a localized string representation.
// It converts the duration to days, hours, or minutes based on the duration length,
// rounding up to the next unit if close to the threshold.
// If no locale is found, it returns the default Duration.String().
//
// The function uses the following rounding rules:
// - Days: rounds up if more than 20 hours remain
// - Hours: rounds up if more than 30 minutes remain
// - Minutes: rounds up if more than 30 seconds remain
//
// Returns a localized string in the format:
// - "X days" for durations >= 1 day
// - "X hours" for durations >= 1 hour
// - "X minutes" for durations < 1 hour
// - "less than a minute" for durations < 1 minute
//
// Example:
//
//	// For 25 hours with English translations:
//	translator.Duration("en", 25 * time.Hour) // Returns: "1 day"
//
//	// For 90 minutes with English translations:
//	translator.Duration("en", 90 * time.Minute) // Returns: "2 hours"
func (t *Translator) Duration(lang string, d time.Duration) string {
	// Convert to total minutes and seconds
	totalSeconds := int(d.Seconds())
	totalMinutes := totalSeconds / 60
	totalHours := totalMinutes / 60
	totalDays := totalHours / 24

	// Calculate remainders for rounding (used in special cases)
	remainderMinutes := totalMinutes % 60
	remainderSeconds := totalSeconds % 60

	// Try to get a translation, if it fails, return the default Duration.String()
	tryTranslate := func(key string, n int) string {
		result := t.N(lang, key, n, "count", fmt.Sprintf("%d", n))
		if result == key || result == "" {
			return d.String()
		}
		return result
	}

	// Round up to the nearest unit for better UX
	// Days: round up if >= 23.5 hours
	if totalHours >= 23 && remainderMinutes >= 30 {
		totalDays = totalHours/24 + 1
	}

	// Hours: round up if >= 59.5 minutes
	if totalMinutes >= 59 && remainderSeconds >= 30 {
		totalHours = totalMinutes/60 + 1
		totalMinutes = 0
	}

	// Minutes: round up if >= 30 seconds remain
	if totalMinutes > 0 && remainderSeconds >= 30 {
		totalMinutes++
	}

	// Handle days
	if totalDays > 0 {
		return tryTranslate("datetime.days", totalDays)
	}

	// Handle hours
	if totalHours > 0 {
		return tryTranslate("datetime.hours", totalHours)
	}

	// Handle minutes
	if totalMinutes > 0 {
		return tryTranslate("datetime.minutes", totalMinutes)
	}

	// Handle less than a minute
	result := t.T(lang, "datetime.minutes.zero")
	if result == "datetime.minutes.zero" || result == "" {
		return d.String()
	}
	return result
}

// TimeSince converts a time.Time to a human-readable "X time ago" format.
// It calculates the duration between the provided time and the current time,
// then uses that duration to generate a localized string.
//
// Returns strings like:
// - "X days ago"
// - "X hours ago"
// - "X minutes ago"
// - "less than a minute ago"
//
// Example:
//
//	// For a time 5 hours in the past:
//	translator.TimeSince("en", time.Now().Add(-5 * time.Hour)) // Returns: "5 hours ago"
//
//	// For a time 2 days in the past:
//	translator.TimeSince("en", time.Now().Add(-48 * time.Hour)) // Returns: "2 days ago"
func (t *Translator) TimeSince(lang string, tm time.Time) string {
	duration := time.Since(tm)

	// Convert to total minutes and seconds
	totalSeconds := int(duration.Seconds())
	totalMinutes := totalSeconds / 60
	totalHours := totalMinutes / 60
	totalDays := totalHours / 24

	// Calculate remainders for rounding (used in special cases)
	remainderMinutes := totalMinutes % 60
	remainderSeconds := totalSeconds % 60

	// Try to get a translation, if it fails, return the default format
	tryTranslate := func(key string, n int) string {
		result := t.N(lang, key, n, "count", fmt.Sprintf("%d", n))
		if result == key || result == "" {
			return fmt.Sprintf("%v ago", duration.Round(time.Second))
		}
		return result
	}

	// Round up to the nearest unit for better UX
	// Days: round up if >= 23.5 hours
	if totalHours >= 23 && remainderMinutes >= 30 {
		totalDays = totalHours/24 + 1
	}

	// Hours: round up if >= 59.5 minutes
	if totalMinutes >= 59 && remainderSeconds >= 30 {
		totalHours = totalMinutes/60 + 1
		totalMinutes = 0
	}

	// Handle days ago
	if totalDays > 0 {
		return tryTranslate("datetime.days.ago", totalDays)
	}

	// Handle hours ago
	if totalHours > 0 {
		return tryTranslate("datetime.hours.ago", totalHours)
	}

	// Handle minutes ago
	if totalMinutes > 0 {
		return tryTranslate("datetime.minutes.ago", totalMinutes)
	}

	// Handle less than a minute ago
	result := t.T(lang, "datetime.minutes.zero.ago")
	if result == "datetime.minutes.zero.ago" || result == "" {
		return fmt.Sprintf("%v ago", duration.Round(time.Second))
	}
	return result
}

// Tc translates a key using language from context
// Uses middleware-injected language from the request context
func (t *Translator) Tc(ctx context.Context, key string, args ...string) string {
	lang := GetLocale(ctx)
	return t.T(lang, key, args...)
}

// Nc translates a plural key using language from context
func (t *Translator) Nc(ctx context.Context, key string, n int, args ...string) string {
	lang := GetLocale(ctx)
	return t.N(lang, key, n, args...)
}

// ExportJSON returns all translations for a language as a JSON string
// Useful for client-side translation in web applications
func (t *Translator) ExportJSON(lang string) (string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Check if the language is supported
	translations, ok := t.translations[lang]
	if !ok {
		return "", &ErrLanguageNotSupported{Lang: lang}
	}

	// Convert map to JSON
	bytes, err := json.Marshal(translations)
	if err != nil {
		return "", errors.Join(ErrFailedToMarshalJSON, err)
	}

	return string(bytes), nil
}

// Td translates a key with a default fallback if not found
// Provides an explicit fallback rather than using the key itself
func (t *Translator) Td(lang, key, defaultValue string, args ...string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Check if the language is supported
	langMap, ok := t.translations[lang]
	if !ok {
		if t.missingLogMode {
			t.logger.Warn("Language not supported", "lang", lang, "key", key)
		}
		return t.sprintf(defaultValue, args)
	}

	// Try to get the translation
	val, ok := t.getTranslation(langMap, key)
	if !ok {
		if t.missingLogMode {
			t.logger.Warn("Translation not found", "lang", lang, "key", key)
		}
		return t.sprintf(defaultValue, args)
	}

	// If the value is not a string, fallback to the default value
	strVal, ok := val.(string)
	if !ok {
		if t.missingLogMode {
			t.logger.Warn("Translation is not a string", "lang", lang, "key", key, "type", fmt.Sprintf("%T", val))
		}
		return t.sprintf(defaultValue, args)
	}

	// Format the translation with the args
	return t.sprintf(strVal, args)
}
