package i18n_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/i18n"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTranslator(t *testing.T) {
	// Create a simple translation map
	translations := map[string]map[string]any{
		"en": {
			"hello":   "Hello",
			"welcome": "Welcome, %{name}!",
			"items": map[string]any{
				"zero":  "No items",
				"one":   "%{count} item",
				"other": "%{count} items",
			},
			"nested": map[string]any{
				"greeting": "Nested greeting",
			},
		},
		"fr": {
			"hello":   "Bonjour",
			"welcome": "Bienvenue, %{name}!",
			"items": map[string]any{
				"zero":  "Aucun élément",
				"one":   "%{count} élément",
				"other": "%{count} éléments",
			},
		},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator
	translator, err := i18n.NewTranslator(context.Background(), adapter)

	// Assert that no error occurred
	require.NoError(t, err)
	require.NotNil(t, translator)
}

func TestTranslatorSupportedLanguages(t *testing.T) {
	// Create a simple translation map
	translations := map[string]map[string]any{
		"en": {"hello": "Hello"},
		"fr": {"hello": "Bonjour"},
		"es": {"hello": "Hola"},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	require.NoError(t, err)

	// Get supported languages
	languages := translator.SupportedLanguages()

	// Assert that the supported languages are correct
	assert.Len(t, languages, 3)
	assert.Contains(t, languages, "en")
	assert.Contains(t, languages, "fr")
	assert.Contains(t, languages, "es")
}

func TestTranslatorHasTranslation(t *testing.T) {
	// Create a simple translation map
	translations := map[string]map[string]any{
		"en": {
			"hello": "Hello",
			"nested": map[string]any{
				"greeting": "Nested greeting",
			},
		},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	require.NoError(t, err)

	// Test existing translations
	assert.True(t, translator.HasTranslation("en", "hello"))
	assert.True(t, translator.HasTranslation("en", "nested.greeting"))

	// Test non-existing translations
	assert.False(t, translator.HasTranslation("en", "goodbye"))
	assert.False(t, translator.HasTranslation("fr", "hello"))
	assert.False(t, translator.HasTranslation("en", "nested.missing"))
}

func TestTranslatorT(t *testing.T) {
	// Create a simple translation map
	translations := map[string]map[string]any{
		"en": {
			"hello":   "Hello",
			"welcome": "Welcome, %{name}!",
			"nested": map[string]any{
				"greeting": "Hello, %{name}!",
			},
		},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	require.NoError(t, err)

	// Test simple translation
	result := translator.T("en", "hello")
	assert.Equal(t, "Hello", result)

	// Test translation with parameters
	result = translator.T("en", "welcome", "name", "John")
	assert.Equal(t, "Welcome, John!", result)

	// Test nested translation
	result = translator.T("en", "nested.greeting", "name", "Alice")
	assert.Equal(t, "Hello, Alice!", result)

	// Test fallback to key
	result = translator.T("en", "missing")
	assert.Equal(t, "missing", result)

	// Test non-existing language
	result = translator.T("fr", "hello")
	assert.Equal(t, "hello", result)
}

func TestTranslatorTWithComplexCases(t *testing.T) {
	// Create a translation map with complex structures
	translations := map[string]map[string]any{
		"en": {
			"complex": map[string]any{
				"nested": map[string]any{
					"deep": map[string]any{
						"value": "Deep nested value with %{param}",
					},
				},
			},
			"mixed_types": map[any]any{
				"key": "Value with %{param}",
			},
			"non_string": 123,
		},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator with missing translations logging
	translator, err := i18n.NewTranslator(
		context.Background(),
		adapter,
		i18n.WithMissingTranslationsLogging(true),
	)
	require.NoError(t, err)

	// Test deeply nested translation
	result := translator.T("en", "complex.nested.deep.value", "param", "test")
	assert.Equal(t, "Deep nested value with test", result)

	// Test translation with map[any]any
	result = translator.T("en", "mixed_types.key", "param", "mixed")
	assert.Equal(t, "Value with mixed", result)

	// Test with non-string value
	result = translator.T("en", "non_string")
	assert.Equal(t, "non_string", result)

	// Test with invalid path format
	result = translator.T("en", "complex.invalid.path")
	assert.Equal(t, "complex.invalid.path", result)

	// Test with odd number of parameters
	result = translator.T("en", "complex.nested.deep.value", "param")
	assert.Equal(t, "Deep nested value with %{param}", result)

	// Create a translator with fallback to key disabled
	translator, err = i18n.NewTranslator(
		context.Background(),
		adapter,
		i18n.WithFallbackToKey(false),
	)
	require.NoError(t, err)

	// Test missing translation with fallback disabled
	result = translator.T("en", "missing.key")
	assert.Equal(t, "", result)
}

func TestTranslatorN(t *testing.T) {
	// Create a simple translation map
	translations := map[string]map[string]any{
		"en": {
			"items": map[string]any{
				"zero":  "No items",
				"one":   "%{count} item",
				"other": "%{count} items",
			},
		},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	require.NoError(t, err)

	// Test zero case
	result := translator.N("en", "items", 0, "count", "0")
	assert.Equal(t, "No items", result)

	// Test one case
	result = translator.N("en", "items", 1, "count", "1")
	assert.Equal(t, "1 item", result)

	// Test other case
	result = translator.N("en", "items", 5, "count", "5")
	assert.Equal(t, "5 items", result)

	// Test fallback to other when specific form is missing
	translations = map[string]map[string]any{
		"en": {
			"items": map[string]any{
				"one":   "%{count} item",
				"other": "%{count} items",
			},
		},
	}

	adapter = &i18n.MapAdapter{
		Data: translations,
	}

	translator, err = i18n.NewTranslator(context.Background(), adapter)
	require.NoError(t, err)

	// Test fallback from zero to other
	result = translator.N("en", "items", 0, "count", "0")
	assert.Equal(t, "0 items", result)
}

func TestTranslatorNWithComplexCases(t *testing.T) {
	// Create a translation map with realistic nested plural structures
	translations := map[string]map[string]any{
		"en": {
			"nested": map[string]any{
				"items": map[string]any{
					"zero":  "No nested items",
					"one":   "%{count} nested item",
					"other": "%{count} nested items",
				},
			},
			"products": map[string]any{
				"inventory": map[string]any{
					"zero":  "Out of stock",
					"one":   "Last item available (%{count} remaining)",
					"other": "%{count} items in stock",
				},
			},
		},
		"fr": {
			"nested": map[string]any{
				"items": map[string]any{
					"zero":  "Aucun élément imbriqué",
					"one":   "%{count} élément imbriqué",
					"other": "%{count} éléments imbriqués",
				},
			},
		},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator
	translator, err := i18n.NewTranslator(
		context.Background(),
		adapter,
	)
	require.NoError(t, err)

	t.Run("Basic nested plural functionality", func(t *testing.T) {
		// Test zero case
		result := translator.N("en", "nested.items", 0, "count", "0")
		assert.Equal(t, "No nested items", result)

		// Test one case
		result = translator.N("en", "nested.items", 1, "count", "1")
		assert.Equal(t, "1 nested item", result)

		// Test other case
		result = translator.N("en", "nested.items", 5, "count", "5")
		assert.Equal(t, "5 nested items", result)
	})

	t.Run("Deeply nested plural translations", func(t *testing.T) {
		// Test zero case
		result := translator.N("en", "products.inventory", 0, "count", "0")
		assert.Equal(t, "Out of stock", result)

		// Test one case
		result = translator.N("en", "products.inventory", 1, "count", "1")
		assert.Equal(t, "Last item available (1 remaining)", result)

		// Test other case
		result = translator.N("en", "products.inventory", 10, "count", "10")
		assert.Equal(t, "10 items in stock", result)
	})

	t.Run("Different languages support", func(t *testing.T) {
		// Test French translations
		result := translator.N("fr", "nested.items", 0, "count", "0")
		assert.Equal(t, "Aucun élément imbriqué", result)

		result = translator.N("fr", "nested.items", 1, "count", "1")
		assert.Equal(t, "1 élément imbriqué", result)

		result = translator.N("fr", "nested.items", 5, "count", "5")
		assert.Equal(t, "5 éléments imbriqués", result)
	})
}

func TestTranslatorNErrorHandling(t *testing.T) {
	// Create a translation map with invalid structures to test error handling
	translations := map[string]map[string]any{
		"en": {
			"invalid_plural": map[string]any{
				"zero": 123,  // Invalid: number instead of string
				"one":  true, // Invalid: boolean instead of string
			},
			"string_plural": "Not a map but a string", // Invalid for pluralization
		},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator with missing translations logging
	translator, err := i18n.NewTranslator(
		context.Background(),
		adapter,
		i18n.WithMissingTranslationsLogging(true),
	)
	require.NoError(t, err)

	t.Run("Invalid plural format", func(t *testing.T) {
		// When a plural key contains non-string values
		result := translator.N("en", "invalid_plural", 0, "count", "0")
		// Should fallback to key
		assert.Equal(t, "invalid_plural", result)
	})

	t.Run("String used for plural translation", func(t *testing.T) {
		// When a string is used instead of a map for pluralization
		result := translator.N("en", "string_plural", 0, "count", "0")
		// Should return the string as-is
		assert.Equal(t, "Not a map but a string", result)
	})

	t.Run("Missing translation key", func(t *testing.T) {
		// When translation key doesn't exist
		result := translator.N("en", "missing.key", 0, "count", "0")
		// Should fallback to key
		assert.Equal(t, "missing.key", result)
	})

	t.Run("Missing language", func(t *testing.T) {
		// When language isn't available
		result := translator.N("de", "invalid_plural", 0, "count", "0")
		// Should fallback to key
		assert.Equal(t, "invalid_plural", result)
	})

	t.Run("Fallback to key disabled", func(t *testing.T) {
		// Create a translator with fallback to key disabled
		noFallbackTranslator, err := i18n.NewTranslator(
			context.Background(),
			adapter,
			i18n.WithFallbackToKey(false),
		)
		require.NoError(t, err)

		// When translation is missing and fallback is disabled
		result := noFallbackTranslator.N("en", "missing.key", 0, "count", "0")
		// Should return empty string
		assert.Equal(t, "", result)
	})
}

func TestTranslatorParameterHandling(t *testing.T) {
	translations := map[string]map[string]any{
		"en": {
			"greeting": map[string]any{
				"zero":  "Hello, %{name}! You have no messages.",
				"one":   "Hello, %{name}! You have %{count} message.",
				"other": "Hello, %{name}! You have %{count} messages.",
			},
		},
	}

	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	translator, err := i18n.NewTranslator(context.Background(), adapter)
	require.NoError(t, err)

	t.Run("Multiple parameters", func(t *testing.T) {
		// Test with multiple parameters
		result := translator.N("en", "greeting", 0, "name", "John", "count", "0")
		assert.Equal(t, "Hello, John! You have no messages.", result)

		result = translator.N("en", "greeting", 1, "name", "John", "count", "1")
		assert.Equal(t, "Hello, John! You have 1 message.", result)

		result = translator.N("en", "greeting", 5, "name", "John", "count", "5")
		assert.Equal(t, "Hello, John! You have 5 messages.", result)
	})
}

func TestTranslatorWithEmptyLanguageSet(t *testing.T) {
	// Create an empty adapter
	emptyAdapter := &i18n.MapAdapter{
		Data: make(map[string]map[string]any),
	}

	// Create a translator with no languages
	translator, err := i18n.NewTranslator(context.Background(), emptyAdapter)
	require.NoError(t, err)

	// Test that supported languages is empty
	langs := translator.SupportedLanguages()
	assert.Empty(t, langs, "Should have no supported languages")

	// Test translation falls back to key when no languages available
	result := translator.T("en", "test.key")
	assert.Equal(t, "test.key", result, "Should fallback to key when no translations available")
}

func TestTranslatorWithOptions(t *testing.T) {
	// Create a simple translation map
	translations := map[string]map[string]any{
		"en": {"hello": "Hello"},
		"fr": {"hello": "Bonjour"},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator with options
	translator, err := i18n.NewTranslator(
		context.Background(),
		adapter,
		i18n.WithDefaultLanguage("fr"),
		i18n.WithFallbackToKey(false),
	)
	require.NoError(t, err)

	// Test fallback to key is disabled
	result := translator.T("en", "missing")
	assert.Equal(t, "", result)
}

func TestTranslatorWithLoggerOptions(t *testing.T) {
	// Create a simple translation map
	translations := map[string]map[string]any{
		"en": {"hello": "Hello"},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a custom logger
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create a new translator with logger option
	translator, err := i18n.NewTranslator(
		context.Background(),
		adapter,
		i18n.WithLogger(logger),
	)
	require.NoError(t, err)

	// Test a translation to ensure the translator works
	result := translator.T("en", "hello")
	assert.Equal(t, "Hello", result)

	// Create a new translator with missing translations logging
	translator, err = i18n.NewTranslator(
		context.Background(),
		adapter,
		i18n.WithMissingTranslationsLogging(true),
	)
	require.NoError(t, err)

	// Test a missing translation
	result = translator.T("en", "missing")
	assert.Equal(t, "missing", result)

	// Create a new translator with no logging
	translator, err = i18n.NewTranslator(
		context.Background(),
		adapter,
		i18n.WithNoLogging(),
	)
	require.NoError(t, err)

	// Test a translation to ensure the translator works
	result = translator.T("en", "hello")
	assert.Equal(t, "Hello", result)

	// Test with nil logger (should use default)
	translator, err = i18n.NewTranslator(
		context.Background(),
		adapter,
		i18n.WithLogger(nil),
	)
	require.NoError(t, err)

	// Test a translation to ensure the translator works
	result = translator.T("en", "hello")
	assert.Equal(t, "Hello", result)
}

func TestTranslatorWithEmptyAdapter(t *testing.T) {
	// Create an empty adapter
	adapter := &i18n.MapAdapter{
		Data: nil,
	}

	// Create a new translator
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	require.NoError(t, err)

	// Test that supported languages is empty
	languages := translator.SupportedLanguages()
	assert.Empty(t, languages)

	// Test that translations are not found
	assert.False(t, translator.HasTranslation("en", "hello"))

	// Test fallback to key
	result := translator.T("en", "hello")
	assert.Equal(t, "hello", result)
}

func TestTranslatorWithInvalidTranslations(t *testing.T) {
	// Create a translation map with an empty language code
	translations := map[string]map[string]any{
		"": {"hello": "Hello"},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator
	_, err := i18n.NewTranslator(context.Background(), adapter)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty language code found")

	// Create a translation map with a nil translations map
	translations = map[string]map[string]any{
		"en": nil,
	}

	adapter = &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator
	_, err = i18n.NewTranslator(context.Background(), adapter)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil translations map for language: en")
}

func TestTranslatorDuration(t *testing.T) {
	// Create a translation map with duration translations
	translations := map[string]map[string]any{
		"en": {
			"datetime": map[string]any{
				"days": map[string]any{
					"one":   "%{count} day",
					"other": "%{count} days",
				},
				"hours": map[string]any{
					"one":   "%{count} hour",
					"other": "%{count} hours",
				},
				"minutes": map[string]any{
					"one":   "%{count} minute",
					"other": "%{count} minutes",
					"zero":  "less than a minute",
				},
			},
		},
		"fr": {
			"datetime": map[string]any{
				"days": map[string]any{
					"one":   "%{count} jour",
					"other": "%{count} jours",
				},
				"hours": map[string]any{
					"one":   "%{count} heure",
					"other": "%{count} heures",
				},
				"minutes": map[string]any{
					"one":   "%{count} minute",
					"other": "%{count} minutes",
					"zero":  "moins d'une minute",
				},
			},
		},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	require.NoError(t, err)

	tests := []struct {
		name     string
		lang     string
		duration time.Duration
		expected string
	}{
		// Basic tests for days
		{name: "1 day in English", lang: "en", duration: 24 * time.Hour, expected: "1 day"},
		{name: "2 days in English", lang: "en", duration: 48 * time.Hour, expected: "2 days"},
		{name: "1 day in French", lang: "fr", duration: 24 * time.Hour, expected: "1 jour"},
		{name: "3 days in French", lang: "fr", duration: 72 * time.Hour, expected: "3 jours"},

		// Basic tests for hours
		{name: "1 hour in English", lang: "en", duration: time.Hour, expected: "1 hour"},
		{name: "5 hours in English", lang: "en", duration: 5 * time.Hour, expected: "5 hours"},
		{name: "1 hour in French", lang: "fr", duration: time.Hour, expected: "1 heure"},
		{name: "6 hours in French", lang: "fr", duration: 6 * time.Hour, expected: "6 heures"},

		// Basic tests for minutes
		{name: "1 minute in English", lang: "en", duration: time.Minute, expected: "1 minute"},
		{name: "15 minutes in English", lang: "en", duration: 15 * time.Minute, expected: "15 minutes"},
		{name: "1 minute in French", lang: "fr", duration: time.Minute, expected: "1 minute"},
		{name: "30 minutes in French", lang: "fr", duration: 30 * time.Minute, expected: "30 minutes"},

		// Test for less than a minute
		{name: "30 seconds in English", lang: "en", duration: 30 * time.Second, expected: "less than a minute"},
		{name: "30 seconds in French", lang: "fr", duration: 30 * time.Second, expected: "moins d'une minute"},

		// Rounding tests - days
		{name: "23 hours (not rounded to days)", lang: "en", duration: 23 * time.Hour, expected: "23 hours"},
		{name: "23.5 hours (rounded to days)", lang: "en", duration: 23*time.Hour + 30*time.Minute, expected: "1 day"},

		// Rounding tests - hours
		{name: "59 minutes (not rounded to hours)", lang: "en", duration: 59 * time.Minute, expected: "59 minutes"},
		{name: "59.5 minutes (rounded to hours)", lang: "en", duration: 59*time.Minute + 30*time.Second, expected: "1 hour"},

		// Rounding tests - minutes
		{name: "59 seconds (not rounded to minutes)", lang: "en", duration: 59 * time.Second, expected: "less than a minute"},
		{name: "90 seconds (rounded to minutes)", lang: "en", duration: 90 * time.Second, expected: "2 minutes"},

		// Edge case - unsupported language
		{name: "Unsupported language", lang: "es", duration: time.Hour, expected: "1h0m0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translator.Duration(tt.lang, tt.duration)
			assert.Equal(t, tt.expected, result, "Duration did not match expected output")
		})
	}
}

func TestTranslatorTimeSince(t *testing.T) {
	// Create a translation map with time since translations
	translations := map[string]map[string]any{
		"en": {
			"datetime": map[string]any{
				"days": map[string]any{
					"ago": map[string]any{
						"one":   "%{count} day ago",
						"other": "%{count} days ago",
					},
				},
				"hours": map[string]any{
					"ago": map[string]any{
						"one":   "%{count} hour ago",
						"other": "%{count} hours ago",
					},
				},
				"minutes": map[string]any{
					"ago": map[string]any{
						"one":   "%{count} minute ago",
						"other": "%{count} minutes ago",
					},
					"zero": map[string]any{
						"ago": "just now",
					},
				},
			},
		},
		"fr": {
			"datetime": map[string]any{
				"days": map[string]any{
					"ago": map[string]any{
						"one":   "il y a %{count} jour",
						"other": "il y a %{count} jours",
					},
				},
				"hours": map[string]any{
					"ago": map[string]any{
						"one":   "il y a %{count} heure",
						"other": "il y a %{count} heures",
					},
				},
				"minutes": map[string]any{
					"ago": map[string]any{
						"one":   "il y a %{count} minute",
						"other": "il y a %{count} minutes",
					},
					"zero": map[string]any{
						"ago": "à l'instant",
					},
				},
			},
		},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	require.NoError(t, err)

	// Current time for our tests
	now := time.Now()

	tests := []struct {
		name     string
		lang     string
		time     time.Time
		expected string
	}{
		// Basic tests for days
		{name: "1 day ago in English", lang: "en", time: now.Add(-24 * time.Hour), expected: "1 day ago"},
		{name: "2 days ago in English", lang: "en", time: now.Add(-48 * time.Hour), expected: "2 days ago"},
		{name: "1 day ago in French", lang: "fr", time: now.Add(-24 * time.Hour), expected: "il y a 1 jour"},
		{name: "3 days ago in French", lang: "fr", time: now.Add(-72 * time.Hour), expected: "il y a 3 jours"},

		// Basic tests for hours
		{name: "1 hour ago in English", lang: "en", time: now.Add(-time.Hour), expected: "1 hour ago"},
		{name: "5 hours ago in English", lang: "en", time: now.Add(-5 * time.Hour), expected: "5 hours ago"},
		{name: "1 hour ago in French", lang: "fr", time: now.Add(-time.Hour), expected: "il y a 1 heure"},
		{name: "6 hours ago in French", lang: "fr", time: now.Add(-6 * time.Hour), expected: "il y a 6 heures"},

		// Basic tests for minutes
		{name: "1 minute ago in English", lang: "en", time: now.Add(-time.Minute), expected: "1 minute ago"},
		{name: "15 minutes ago in English", lang: "en", time: now.Add(-15 * time.Minute), expected: "15 minutes ago"},
		{name: "1 minute ago in French", lang: "fr", time: now.Add(-time.Minute), expected: "il y a 1 minute"},
		{name: "30 minutes ago in French", lang: "fr", time: now.Add(-30 * time.Minute), expected: "il y a 30 minutes"},

		// Test for less than a minute ago
		{name: "30 seconds ago in English", lang: "en", time: now.Add(-30 * time.Second), expected: "just now"},
		{name: "30 seconds ago in French", lang: "fr", time: now.Add(-30 * time.Second), expected: "à l'instant"},

		// Rounding tests - days
		{name: "23 hours ago (not rounded to days)", lang: "en", time: now.Add(-23 * time.Hour), expected: "23 hours ago"},
		{name: "23.5 hours ago (rounded to days)", lang: "en", time: now.Add(-23*time.Hour - 30*time.Minute), expected: "1 day ago"},

		// Rounding tests - hours
		{name: "59 minutes ago (not rounded to hours)", lang: "en", time: now.Add(-59 * time.Minute), expected: "59 minutes ago"},
		{name: "59.5 minutes ago (rounded to hours)", lang: "en", time: now.Add(-59*time.Minute - 30*time.Second), expected: "1 hour ago"},

		// Edge case - unsupported language
		{name: "Unsupported language", lang: "es", time: now.Add(-time.Hour), expected: "1h0m0s ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translator.TimeSince(tt.lang, tt.time)
			assert.Equal(t, tt.expected, result, "TimeSince did not match expected output")
		})
	}
}

func TestTranslatorDurationWithMissingTranslations(t *testing.T) {
	// Create a translation map with incomplete translations
	translations := map[string]map[string]any{
		"en": {
			"datetime": map[string]any{
				// Only days translation exists
				"days": map[string]any{
					"one":   "%{count} day",
					"other": "%{count} days",
				},
				// Missing hours and minutes
			},
		},
		"minimal": {
			// Completely empty language for testing
		},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator with logging enabled
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	translator, err := i18n.NewTranslator(
		context.Background(),
		adapter,
		i18n.WithMissingTranslationsLogging(true),
		i18n.WithLogger(logger),
	)
	require.NoError(t, err)

	tests := []struct {
		name     string
		lang     string
		duration time.Duration
		expected string
	}{
		// Test with incomplete translations - should fall back to Duration.String()
		{name: "Missing hours translation", lang: "en", duration: 5 * time.Hour, expected: "5h0m0s"},
		{name: "Missing minutes translation", lang: "en", duration: 30 * time.Minute, expected: "30m0s"},
		{name: "Days translation works", lang: "en", duration: 3 * 24 * time.Hour, expected: "3 days"},

		// Test with completely missing translations
		{name: "Empty language - days", lang: "minimal", duration: 2 * 24 * time.Hour, expected: "48h0m0s"},
		{name: "Empty language - hours", lang: "minimal", duration: 4 * time.Hour, expected: "4h0m0s"},
		{name: "Empty language - minutes", lang: "minimal", duration: 15 * time.Minute, expected: "15m0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translator.Duration(tt.lang, tt.duration)
			assert.Equal(t, tt.expected, result, "Duration did not handle missing translations correctly")
		})
	}
}

func TestTranslatorTimeSinceWithMissingTranslations(t *testing.T) {
	// Create a translation map with incomplete translations
	translations := map[string]map[string]any{
		"en": {
			"datetime": map[string]any{
				// Only days translation exists
				"days": map[string]any{
					"ago": map[string]any{
						"one":   "%{count} day ago",
						"other": "%{count} days ago",
					},
				},
				// Missing hours and minutes
			},
		},
		"minimal": {
			// Completely empty language for testing
		},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator with logging enabled
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	translator, err := i18n.NewTranslator(
		context.Background(),
		adapter,
		i18n.WithMissingTranslationsLogging(true),
		i18n.WithLogger(logger),
	)
	require.NoError(t, err)

	// Current time for our tests
	now := time.Now()

	tests := []struct {
		name     string
		lang     string
		time     time.Time
		expected string
	}{
		// Test with incomplete translations
		{name: "Missing hours translation", lang: "en", time: now.Add(-5 * time.Hour), expected: "5h0m0s ago"},
		{name: "Missing minutes translation", lang: "en", time: now.Add(-30 * time.Minute), expected: "30m0s ago"},
		{name: "Days translation works", lang: "en", time: now.Add(-3 * 24 * time.Hour), expected: "3 days ago"},

		// Test with completely missing translations
		{name: "Empty language - days", lang: "minimal", time: now.Add(-2 * 24 * time.Hour), expected: "48h0m0s ago"},
		{name: "Empty language - hours", lang: "minimal", time: now.Add(-4 * time.Hour), expected: "4h0m0s ago"},
		{name: "Empty language - minutes", lang: "minimal", time: now.Add(-15 * time.Minute), expected: "15m0s ago"},
		{name: "Empty language - seconds", lang: "minimal", time: now.Add(-30 * time.Second), expected: "30s ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translator.TimeSince(tt.lang, tt.time)
			assert.Equal(t, tt.expected, result, "TimeSince did not handle missing translations correctly")
		})
	}
}

func TestTranslatorExportJSON(t *testing.T) {
	// Create a translation map with various structures
	translations := map[string]map[string]any{
		"en": {
			"hello":   "Hello",
			"welcome": "Welcome, %{name}!",
			"nested": map[string]any{
				"key": "Nested value",
			},
			"items": map[string]any{
				"zero":  "No items",
				"one":   "One item",
				"other": "%{count} items",
			},
		},
		"fr": {
			"hello": "Bonjour",
		},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	require.NoError(t, err)

	t.Run("successful export", func(t *testing.T) {
		// Export translations for English
		jsonStr, err := translator.ExportJSON("en")
		require.NoError(t, err)
		assert.NotEmpty(t, jsonStr)

		// Verify the JSON content contains expected keys
		assert.Contains(t, jsonStr, "hello")
		assert.Contains(t, jsonStr, "Welcome, %{name}!")
		assert.Contains(t, jsonStr, "nested")
		assert.Contains(t, jsonStr, "items")

		// Verify for French (should be a simpler structure)
		jsonStr, err = translator.ExportJSON("fr")
		require.NoError(t, err)
		assert.Contains(t, jsonStr, "Bonjour")
	})

	t.Run("language not supported", func(t *testing.T) {
		// Try to export translations for a non-existent language
		_, err := translator.ExportJSON("es")
		require.Error(t, err)
		assert.IsType(t, &i18n.ErrLanguageNotSupported{}, err)
		assert.Contains(t, err.Error(), "language not supported")
	})
}

func TestTranslatorTd(t *testing.T) {
	// Create a translation map
	translations := map[string]map[string]any{
		"en": {
			"hello":      "Hello",
			"welcome":    "Welcome, %{name}!",
			"non_string": 123,
		},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	require.NoError(t, err)

	t.Run("existing translation", func(t *testing.T) {
		// Test with existing key
		result := translator.Td("en", "hello", "Default Hello")
		assert.Equal(t, "Hello", result)

		// Test with parameters
		result = translator.Td("en", "welcome", "Default Welcome, %{name}!", "name", "John")
		assert.Equal(t, "Welcome, John!", result)
	})

	t.Run("missing translation", func(t *testing.T) {
		// Test with missing key
		result := translator.Td("en", "missing", "Default Text")
		assert.Equal(t, "Default Text", result)

		// Test with parameters in default text
		result = translator.Td("en", "missing", "Default with %{param}", "param", "value")
		assert.Equal(t, "Default with value", result)
	})

	t.Run("non-string translation", func(t *testing.T) {
		// Test with non-string value
		result := translator.Td("en", "non_string", "Default for non-string")
		assert.Equal(t, "Default for non-string", result)
	})

	t.Run("language not supported", func(t *testing.T) {
		// Test with non-existent language
		result := translator.Td("fr", "hello", "Default Hello")
		assert.Equal(t, "Default Hello", result)
	})
}

func TestTranslatorTc(t *testing.T) {
	// Create a translation map
	translations := map[string]map[string]any{
		"en": {
			"hello":   "Hello",
			"welcome": "Welcome, %{name}!",
			"nested": map[string]any{
				"greeting": "Hello, %{name}!",
			},
		},
		"fr": {
			"hello":   "Bonjour",
			"welcome": "Bienvenue, %{name}!",
		},
		"es": {
			"hello": "Hola",
		},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	require.NoError(t, err)

	t.Run("uses language from context", func(t *testing.T) {
		// Test with English context
		ctx := i18n.SetLocale(context.Background(), "en")
		result := translator.Tc(ctx, "hello")
		assert.Equal(t, "Hello", result)

		// Test with French context
		ctx = i18n.SetLocale(context.Background(), "fr")
		result = translator.Tc(ctx, "hello")
		assert.Equal(t, "Bonjour", result)

		// Test with Spanish context
		ctx = i18n.SetLocale(context.Background(), "es")
		result = translator.Tc(ctx, "hello")
		assert.Equal(t, "Hola", result)
	})

	t.Run("handles parameters from context", func(t *testing.T) {
		ctx := i18n.SetLocale(context.Background(), "en")
		result := translator.Tc(ctx, "welcome", "name", "John")
		assert.Equal(t, "Welcome, John!", result)

		ctx = i18n.SetLocale(context.Background(), "fr")
		result = translator.Tc(ctx, "welcome", "name", "Marie")
		assert.Equal(t, "Bienvenue, Marie!", result)
	})

	t.Run("handles nested keys from context", func(t *testing.T) {
		ctx := i18n.SetLocale(context.Background(), "en")
		result := translator.Tc(ctx, "nested.greeting", "name", "Alice")
		assert.Equal(t, "Hello, Alice!", result)
	})

	t.Run("falls back to default when no locale in context", func(t *testing.T) {
		// Context without locale should use default language
		ctx := context.Background()
		result := translator.Tc(ctx, "hello")
		assert.Equal(t, "Hello", result) // Should use default 'en'
	})

	t.Run("handles missing translation in context language", func(t *testing.T) {
		// Spanish doesn't have 'welcome' translation
		ctx := i18n.SetLocale(context.Background(), "es")
		result := translator.Tc(ctx, "welcome")
		assert.Equal(t, "welcome", result) // Should fallback to key
	})

	t.Run("handles unsupported language in context", func(t *testing.T) {
		ctx := i18n.SetLocale(context.Background(), "de")
		result := translator.Tc(ctx, "hello")
		assert.Equal(t, "hello", result) // Should fallback to key
	})

	t.Run("preserves other context values", func(t *testing.T) {
		// Add custom value to context
		type contextKey string
		customKey := contextKey("custom")
		ctx := context.WithValue(context.Background(), customKey, "custom-value")
		ctx = i18n.SetLocale(ctx, "fr")

		// Verify translation works
		result := translator.Tc(ctx, "hello")
		assert.Equal(t, "Bonjour", result)

		// Verify custom context value is preserved
		customValue := ctx.Value(customKey)
		assert.Equal(t, "custom-value", customValue)
	})
}

func TestTranslatorNc(t *testing.T) {
	// Create a translation map with pluralization
	translations := map[string]map[string]any{
		"en": {
			"items": map[string]any{
				"zero":  "No items",
				"one":   "%{count} item",
				"other": "%{count} items",
			},
			"messages": map[string]any{
				"zero":  "No messages for %{user}",
				"one":   "%{count} message for %{user}",
				"other": "%{count} messages for %{user}",
			},
		},
		"fr": {
			"items": map[string]any{
				"zero":  "Aucun élément",
				"one":   "%{count} élément",
				"other": "%{count} éléments",
			},
		},
		"es": {
			"items": map[string]any{
				"one":   "%{count} artículo",
				"other": "%{count} artículos",
			},
		},
	}

	// Create a MapAdapter with the translations
	adapter := &i18n.MapAdapter{
		Data: translations,
	}

	// Create a new translator
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	require.NoError(t, err)

	t.Run("uses language from context for pluralization", func(t *testing.T) {
		// Test with English context
		ctx := i18n.SetLocale(context.Background(), "en")

		result := translator.Nc(ctx, "items", 0)
		assert.Equal(t, "No items", result)

		result = translator.Nc(ctx, "items", 1)
		assert.Equal(t, "1 item", result)

		result = translator.Nc(ctx, "items", 5)
		assert.Equal(t, "5 items", result)

		// Test with French context
		ctx = i18n.SetLocale(context.Background(), "fr")

		result = translator.Nc(ctx, "items", 0)
		assert.Equal(t, "Aucun élément", result)

		result = translator.Nc(ctx, "items", 1)
		assert.Equal(t, "1 élément", result)

		result = translator.Nc(ctx, "items", 3)
		assert.Equal(t, "3 éléments", result)
	})

	t.Run("handles additional parameters from context", func(t *testing.T) {
		ctx := i18n.SetLocale(context.Background(), "en")

		result := translator.Nc(ctx, "messages", 0, "user", "John")
		assert.Equal(t, "No messages for John", result)

		result = translator.Nc(ctx, "messages", 1, "user", "Alice")
		assert.Equal(t, "1 message for Alice", result)

		result = translator.Nc(ctx, "messages", 3, "user", "Bob")
		assert.Equal(t, "3 messages for Bob", result)
	})

	t.Run("falls back when zero form missing", func(t *testing.T) {
		// Spanish doesn't have 'zero' form, should fallback to 'other'
		ctx := i18n.SetLocale(context.Background(), "es")

		result := translator.Nc(ctx, "items", 0)
		assert.Equal(t, "0 artículos", result) // Falls back to 'other' form

		result = translator.Nc(ctx, "items", 1)
		assert.Equal(t, "1 artículo", result)

		result = translator.Nc(ctx, "items", 2)
		assert.Equal(t, "2 artículos", result)
	})

	t.Run("falls back to default when no locale in context", func(t *testing.T) {
		// Context without locale should use default language
		ctx := context.Background()

		result := translator.Nc(ctx, "items", 1)
		assert.Equal(t, "1 item", result) // Should use default 'en'

		result = translator.Nc(ctx, "items", 2)
		assert.Equal(t, "2 items", result)
	})

	t.Run("handles missing pluralization in context language", func(t *testing.T) {
		// Spanish doesn't have 'messages' translation
		ctx := i18n.SetLocale(context.Background(), "es")
		result := translator.Nc(ctx, "messages", 1, "user", "Carlos")
		assert.Equal(t, "messages", result) // Should fallback to key
	})

	t.Run("handles unsupported language in context", func(t *testing.T) {
		ctx := i18n.SetLocale(context.Background(), "de")
		result := translator.Nc(ctx, "items", 1)
		assert.Equal(t, "items", result) // Should fallback to key
	})

	t.Run("preserves other context values", func(t *testing.T) {
		// Add custom value to context
		type contextKey string
		customKey := contextKey("custom")
		ctx := context.WithValue(context.Background(), customKey, "test-data")
		ctx = i18n.SetLocale(ctx, "fr")

		// Verify pluralization works
		result := translator.Nc(ctx, "items", 2)
		assert.Equal(t, "2 éléments", result)

		// Verify custom context value is preserved
		customValue := ctx.Value(customKey)
		assert.Equal(t, "test-data", customValue)
	})
}

func TestTranslatorContextIntegration(t *testing.T) {
	// Create a comprehensive translation map
	translations := map[string]map[string]any{
		"en": {
			"app": map[string]any{
				"title": "My Application",
				"user": map[string]any{
					"greeting": "Hello, %{name}!",
					"items": map[string]any{
						"zero":  "You have no items",
						"one":   "You have %{count} item",
						"other": "You have %{count} items",
					},
				},
			},
		},
		"fr": {
			"app": map[string]any{
				"title": "Mon Application",
				"user": map[string]any{
					"greeting": "Bonjour, %{name}!",
					"items": map[string]any{
						"zero":  "Vous n'avez aucun élément",
						"one":   "Vous avez %{count} élément",
						"other": "Vous avez %{count} éléments",
					},
				},
			},
		},
	}

	adapter := &i18n.MapAdapter{Data: translations}
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	require.NoError(t, err)

	t.Run("complex nested context-based translations", func(t *testing.T) {
		// English context
		ctx := i18n.SetLocale(context.Background(), "en")

		title := translator.Tc(ctx, "app.title")
		assert.Equal(t, "My Application", title)

		greeting := translator.Tc(ctx, "app.user.greeting", "name", "John")
		assert.Equal(t, "Hello, John!", greeting)

		items := translator.Nc(ctx, "app.user.items", 3)
		assert.Equal(t, "You have 3 items", items)

		// French context
		ctx = i18n.SetLocale(context.Background(), "fr")

		title = translator.Tc(ctx, "app.title")
		assert.Equal(t, "Mon Application", title)

		greeting = translator.Tc(ctx, "app.user.greeting", "name", "Marie")
		assert.Equal(t, "Bonjour, Marie!", greeting)

		items = translator.Nc(ctx, "app.user.items", 1)
		assert.Equal(t, "Vous avez 1 élément", items)
	})

	t.Run("context switching between translations", func(t *testing.T) {
		// Start with English
		ctx := i18n.SetLocale(context.Background(), "en")
		result1 := translator.Tc(ctx, "app.title")
		assert.Equal(t, "My Application", result1)

		// Switch to French using same context variable
		ctx = i18n.SetLocale(ctx, "fr")
		result2 := translator.Tc(ctx, "app.title")
		assert.Equal(t, "Mon Application", result2)

		// Switch back to English
		ctx = i18n.SetLocale(ctx, "en")
		result3 := translator.Tc(ctx, "app.title")
		assert.Equal(t, "My Application", result3)
	})

	t.Run("mixed context and direct translations", func(t *testing.T) {
		ctx := i18n.SetLocale(context.Background(), "fr")

		// Context-based translation
		contextResult := translator.Tc(ctx, "app.title")
		assert.Equal(t, "Mon Application", contextResult)

		// Direct translation (ignores context)
		directResult := translator.T("en", "app.title")
		assert.Equal(t, "My Application", directResult)

		// Context-based should still work
		contextResult2 := translator.Tc(ctx, "app.title")
		assert.Equal(t, "Mon Application", contextResult2)
	})
}

func TestTranslatorConcurrency(t *testing.T) {
	// Create a translation map
	translations := map[string]map[string]any{
		"en": {
			"counter": "Count: %{value}",
			"items": map[string]any{
				"zero":  "No items",
				"one":   "%{count} item",
				"other": "%{count} items",
			},
		},
		"fr": {
			"counter": "Compte: %{value}",
			"items": map[string]any{
				"zero":  "Aucun élément",
				"one":   "%{count} élément",
				"other": "%{count} éléments",
			},
		},
	}

	adapter := &i18n.MapAdapter{Data: translations}
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	require.NoError(t, err)

	t.Run("concurrent translations are thread-safe", func(t *testing.T) {
		const numGoroutines = 100
		const iterationsPerGoroutine = 50

		// Channel to collect results
		results := make(chan string, numGoroutines*iterationsPerGoroutine)

		// Start multiple goroutines performing translations
		for g := 0; g < numGoroutines; g++ {
			go func(goroutineID int) {
				for i := 0; i < iterationsPerGoroutine; i++ {
					// Alternate between languages
					lang := "en"
					if (goroutineID+i)%2 == 0 {
						lang = "fr"
					}

					// Perform translation
					result := translator.T(lang, "counter", "value", fmt.Sprintf("%d-%d", goroutineID, i))
					results <- result
				}
			}(g)
		}

		// Collect all results
		var allResults []string
		for i := 0; i < numGoroutines*iterationsPerGoroutine; i++ {
			allResults = append(allResults, <-results)
		}

		// Verify results
		assert.Len(t, allResults, numGoroutines*iterationsPerGoroutine)

		// Count English and French results
		enCount, frCount := 0, 0
		for _, result := range allResults {
			if strings.Contains(result, "Count:") {
				enCount++
			} else if strings.Contains(result, "Compte:") {
				frCount++
			}
		}

		// Both languages should be represented
		assert.Greater(t, enCount, 0, "Should have English translations")
		assert.Greater(t, frCount, 0, "Should have French translations")
		assert.Equal(t, numGoroutines*iterationsPerGoroutine, enCount+frCount, "All results should be valid")
	})

	t.Run("concurrent context-based translations", func(t *testing.T) {
		const numGoroutines = 50

		// Channel to collect results
		results := make(chan string, numGoroutines)

		// Start multiple goroutines with different contexts
		for g := 0; g < numGoroutines; g++ {
			go func(goroutineID int) {
				// Create context with different language
				lang := "en"
				if goroutineID%2 == 0 {
					lang = "fr"
				}
				ctx := i18n.SetLocale(context.Background(), lang)

				// Perform context-based translation
				result := translator.Tc(ctx, "counter", "value", fmt.Sprintf("g%d", goroutineID))
				results <- result
			}(g)
		}

		// Collect all results
		var allResults []string
		for i := 0; i < numGoroutines; i++ {
			allResults = append(allResults, <-results)
		}

		// Verify results
		assert.Len(t, allResults, numGoroutines)

		// All results should be valid translations
		for _, result := range allResults {
			assert.True(t,
				strings.Contains(result, "Count:") || strings.Contains(result, "Compte:"),
				"Result should be a valid translation: %s", result)
		}
	})
}
