package i18n_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/i18n"
)

func BenchmarkJSONParserLarge(b *testing.B) {
	// Build a very large JSON structure
	var builder strings.Builder
	builder.WriteString(`{"en": {`)
	for i := 0; i < 10000; i++ {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(fmt.Sprintf(`"key_%d": "value_%d"`, i, i))
	}
	builder.WriteString(`}}`)
	content := builder.String()

	parser := i18n.NewJSONParser()
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, err := parser.Parse(ctx, content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONParserDeepNested(b *testing.B) {
	// Create deeply nested structure
	content := `{"en": {`
	for i := range 100 {
		content += fmt.Sprintf(`"level_%d": {`, i)
	}
	content += `"deep_key": "deep_value"`
	for range 100 {
		content += `}`
	}
	content += `}}`

	parser := i18n.NewJSONParser()
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, err := parser.Parse(ctx, content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkYAMLParserLarge(b *testing.B) {
	var builder strings.Builder
	languages := []string{"en", "fr", "es", "de", "it"}

	for _, lang := range languages {
		builder.WriteString(fmt.Sprintf("%s:\n", lang))
		for i := 0; i < 1000; i++ {
			builder.WriteString(fmt.Sprintf("  key_%d: \"Value %d for %s\"\n", i, i, lang))
		}
	}
	content := builder.String()

	parser := i18n.NewYAMLParser()
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, err := parser.Parse(ctx, content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkYAMLParserDeepNested(b *testing.B) {
	var builder strings.Builder
	builder.WriteString("en:\n")

	// Create deeply nested structure
	for i := 0; i < 50; i++ {
		builder.WriteString(strings.Repeat("  ", i+1))
		builder.WriteString(fmt.Sprintf("level_%d:\n", i))
	}
	builder.WriteString(strings.Repeat("  ", 51))
	builder.WriteString("deep_key: deep_value\n")
	content := builder.String()

	parser := i18n.NewYAMLParser()
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, err := parser.Parse(ctx, content)
		if err != nil {
			b.Fatal(err)
		}
	}
}
