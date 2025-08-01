package randomname_test

import (
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/randomname"
)

func BenchmarkGenerate(b *testing.B) {
	b.Run("Simple", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = randomname.Simple()
		}
	})

	b.Run("WithSuffix", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = randomname.WithSuffix()
		}
	})

	b.Run("Descriptive", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = randomname.Descriptive()
		}
	})

	b.Run("Full", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = randomname.Full()
		}
	})
}

func BenchmarkGenerateWithOptions(b *testing.B) {
	b.Run("DefaultOptions", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = randomname.Generate(nil)
		}
	})

	b.Run("CustomPattern", func(b *testing.B) {
		opts := &randomname.Options{
			Pattern: []randomname.WordType{randomname.Size, randomname.Color, randomname.Noun},
		}
		b.ReportAllocs()
		for b.Loop() {
			_ = randomname.Generate(opts)
		}
	})

	b.Run("WithValidator", func(b *testing.B) {
		opts := &randomname.Options{
			Validator: func(s string) bool { return true },
		}
		b.ReportAllocs()
		for b.Loop() {
			_ = randomname.Generate(opts)
		}
	})

	b.Run("CustomWords", func(b *testing.B) {
		opts := &randomname.Options{
			Words: map[randomname.WordType][]string{
				randomname.Adjective: {"fast", "quick", "speedy"},
				randomname.Noun:      {"benchmark", "test", "measure"},
			},
		}
		b.ReportAllocs()
		for b.Loop() {
			_ = randomname.Generate(opts)
		}
	})
}

func BenchmarkSuffixTypes(b *testing.B) {
	suffixes := []struct {
		name   string
		suffix randomname.SuffixType
	}{
		{"NoSuffix", randomname.NoSuffix},
		{"Hex6", randomname.Hex6},
		{"Hex8", randomname.Hex8},
		{"Numeric4", randomname.Numeric4},
	}

	for _, s := range suffixes {
		b.Run(s.name, func(b *testing.B) {
			opts := &randomname.Options{
				Suffix: s.suffix,
			}
			b.ReportAllocs()
			for b.Loop() {
				_ = randomname.Generate(opts)
			}
		})
	}
}

func BenchmarkPatternComplexity(b *testing.B) {
	patterns := []struct {
		name    string
		pattern []randomname.WordType
	}{
		{"1Word", []randomname.WordType{randomname.Noun}},
		{"2Words", []randomname.WordType{randomname.Adjective, randomname.Noun}},
		{"3Words", []randomname.WordType{randomname.Adjective, randomname.Color, randomname.Noun}},
		{"4Words", []randomname.WordType{randomname.Size, randomname.Adjective, randomname.Color, randomname.Noun}},
		{"5Words", []randomname.WordType{randomname.Origin, randomname.Size, randomname.Adjective, randomname.Color, randomname.Noun}},
	}

	for _, p := range patterns {
		b.Run(p.name, func(b *testing.B) {
			opts := &randomname.Options{
				Pattern: p.pattern,
			}
			b.ReportAllocs()
			for b.Loop() {
				_ = randomname.Generate(opts)
			}
		})
	}
}

func BenchmarkConcurrentGeneration(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = randomname.Generate(&randomname.Options{
				Pattern: []randomname.WordType{randomname.Adjective, randomname.Noun},
				Suffix:  randomname.Hex6,
			})
		}
	})
}

func BenchmarkValidatorRejection(b *testing.B) {
	b.Run("AcceptFirst", func(b *testing.B) {
		opts := &randomname.Options{
			Validator: func(s string) bool { return true },
		}
		b.ReportAllocs()
		for b.Loop() {
			_ = randomname.Generate(opts)
		}
	})

	b.Run("RejectFirst", func(b *testing.B) {
		opts := &randomname.Options{
			Validator: func(s string) bool {
				// Accept on second attempt
				return len(s) > 10
			},
		}
		b.ReportAllocs()
		for b.Loop() {
			_ = randomname.Generate(opts)
		}
	})

	b.Run("RejectMany", func(b *testing.B) {
		count := 0
		opts := &randomname.Options{
			Validator: func(s string) bool {
				count++
				// Accept every 5th attempt
				return count%5 == 0
			},
		}
		b.ReportAllocs()
		for b.Loop() {
			count = 0
			_ = randomname.Generate(opts)
		}
	})
}
