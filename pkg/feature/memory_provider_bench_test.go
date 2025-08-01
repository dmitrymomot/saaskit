package feature_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/feature"
)

func BenchmarkMemoryProvider_IsEnabled(b *testing.B) {
	percentage := 50
	provider, err := feature.NewMemoryProvider(
		&feature.Flag{Name: "feature-1", Enabled: true},
		&feature.Flag{Name: "feature-2", Enabled: false},
		&feature.Flag{
			Name:    "feature-3",
			Enabled: true,
			Strategy: feature.NewTargetedStrategy(
				feature.TargetCriteria{Percentage: &percentage},
				feature.WithUserIDExtractor(func(ctx context.Context) string { return "user-123" }),
			),
		},
	)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.Run("enabled-flag", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_, _ = provider.IsEnabled(ctx, "feature-1")
		}
	})

	b.Run("disabled-flag", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_, _ = provider.IsEnabled(ctx, "feature-2")
		}
	})

	b.Run("percentage-strategy", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_, _ = provider.IsEnabled(ctx, "feature-3")
		}
	})

	b.Run("non-existent-flag", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_, _ = provider.IsEnabled(ctx, "non-existent")
		}
	})
}

func BenchmarkMemoryProvider_ListFlags(b *testing.B) {
	flags := make([]*feature.Flag, 100)
	for i := range 100 {
		tags := []string{"tag-common"}
		if i%10 == 0 {
			tags = append(tags, "tag-special")
		}
		if i%5 == 0 {
			tags = append(tags, "tag-frequent")
		}
		flags[i] = &feature.Flag{
			Name:    fmt.Sprintf("feature-%d", i),
			Enabled: i%2 == 0,
			Tags:    tags,
		}
	}

	provider, err := feature.NewMemoryProvider(flags...)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.Run("all-flags", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_, _ = provider.ListFlags(ctx)
		}
	})

	b.Run("filter-common-tag", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_, _ = provider.ListFlags(ctx, "tag-common")
		}
	})

	b.Run("filter-special-tag", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_, _ = provider.ListFlags(ctx, "tag-special")
		}
	})

	b.Run("filter-multiple-tags", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_, _ = provider.ListFlags(ctx, "tag-special", "tag-frequent")
		}
	})
}

func BenchmarkMemoryProvider_ConcurrentAccess(b *testing.B) {
	provider, err := feature.NewMemoryProvider(
		&feature.Flag{Name: "concurrent-1", Enabled: true},
		&feature.Flag{Name: "concurrent-2", Enabled: false},
		&feature.Flag{Name: "concurrent-3", Enabled: true, Tags: []string{"tag1", "tag2"}},
	)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		flags := []string{"concurrent-1", "concurrent-2", "concurrent-3"}
		for pb.Next() {
			_, _ = provider.IsEnabled(ctx, flags[i%len(flags)])
			i++
		}
	})
}
