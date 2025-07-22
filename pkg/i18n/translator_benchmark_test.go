package i18n_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/i18n"
)

func BenchmarkTranslatorLargeDataset(b *testing.B) {
	// Create a large number of translations
	const numTranslations = 1000
	translations := make(map[string]map[string]any)

	for lang := range []string{"en", "fr", "es", "de", "it"} {
		langCode := []string{"en", "fr", "es", "de", "it"}[lang]
		translations[langCode] = make(map[string]any)

		for i := range numTranslations {
			key := fmt.Sprintf("key_%d", i)
			value := fmt.Sprintf("Value %d in %s", i, langCode)
			translations[langCode][key] = value
		}
	}

	adapter := &i18n.MapAdapter{Data: translations}
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	// Benchmark translation lookups
	for b.Loop() {
		// Access different keys to avoid caching effects
		for i := range 100 {
			key := fmt.Sprintf("key_%d", i*10)
			translator.T("en", key)
		}
	}
}

func BenchmarkTranslatorNestedKeys(b *testing.B) {
	// Create deeply nested translations
	translations := map[string]map[string]any{
		"en": {
			"level1": map[string]any{
				"level2": map[string]any{
					"level3": map[string]any{
						"level4": map[string]any{
							"level5": map[string]any{
								"key": "Deep value",
							},
						},
					},
				},
			},
		},
	}

	adapter := &i18n.MapAdapter{Data: translations}
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for b.Loop() {
		translator.T("en", "level1.level2.level3.level4.level5.key")
	}
}

func BenchmarkTranslatorWithParameters(b *testing.B) {
	translations := map[string]map[string]any{
		"en": {
			"greeting": "Hello %{name}, you have %{count} messages from %{sender}",
		},
	}

	adapter := &i18n.MapAdapter{Data: translations}
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for b.Loop() {
		translator.T("en", "greeting",
			"name", "John",
			"count", "5",
			"sender", "Alice")
	}
}

func BenchmarkTranslatorPluralization(b *testing.B) {
	translations := map[string]map[string]any{
		"en": {
			"items": map[string]any{
				"zero":  "No items",
				"one":   "%{count} item",
				"other": "%{count} items",
			},
		},
	}

	adapter := &i18n.MapAdapter{Data: translations}
	translator, err := i18n.NewTranslator(context.Background(), adapter)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	counts := []int{0, 1, 5, 10, 100}
	i := 0

	for b.Loop() {
		count := counts[i%len(counts)]
		translator.N("en", "items", count, "count", fmt.Sprintf("%d", count))
		i++
	}
}
