package feature_test

import (
	"context"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/feature"
)

func BenchmarkStrategies(b *testing.B) {
	getUserID := func(ctx context.Context) string { return "user-123" }
	getUserGroups := func(ctx context.Context) []string { return []string{"group-1", "group-2"} }
	getEnvironment := func(ctx context.Context) string { return "production" }

	b.Run("AlwaysOnStrategy", func(b *testing.B) {
		strategy := feature.NewAlwaysOnStrategy()
		ctx := context.Background()
		b.ResetTimer()
		for b.Loop() {
			_, _ = strategy.Evaluate(ctx)
		}
	})

	b.Run("AlwaysOffStrategy", func(b *testing.B) {
		strategy := feature.NewAlwaysOffStrategy()
		ctx := context.Background()
		b.ResetTimer()
		for b.Loop() {
			_, _ = strategy.Evaluate(ctx)
		}
	})

	b.Run("UserTargetStrategy", func(b *testing.B) {
		strategy := feature.NewTargetedStrategy(
			feature.TargetCriteria{UserIDs: []string{"user-123", "user-456"}},
			feature.WithUserIDExtractor(getUserID),
		)
		ctx := context.Background()
		b.ResetTimer()
		for b.Loop() {
			_, _ = strategy.Evaluate(ctx)
		}
	})

	b.Run("GroupTargetStrategy", func(b *testing.B) {
		strategy := feature.NewTargetedStrategy(
			feature.TargetCriteria{Groups: []string{"group-1", "group-3"}},
			feature.WithUserGroupsExtractor(getUserGroups),
		)
		ctx := context.Background()
		b.ResetTimer()
		for b.Loop() {
			_, _ = strategy.Evaluate(ctx)
		}
	})

	b.Run("PercentageStrategy", func(b *testing.B) {
		percentage := 50
		strategy := feature.NewTargetedStrategy(
			feature.TargetCriteria{Percentage: &percentage},
			feature.WithUserIDExtractor(getUserID),
		)
		ctx := context.Background()
		b.ResetTimer()
		for b.Loop() {
			_, _ = strategy.Evaluate(ctx)
		}
	})

	b.Run("EnvironmentStrategy", func(b *testing.B) {
		strategy := feature.NewEnvironmentStrategy(
			[]string{"production", "staging"},
			feature.WithEnvironmentExtractor(getEnvironment),
		)
		ctx := context.Background()
		b.ResetTimer()
		for b.Loop() {
			_, _ = strategy.Evaluate(ctx)
		}
	})

	b.Run("TargetedStrategy-Complex", func(b *testing.B) {
		percentage := 50
		criteria := feature.TargetCriteria{
			UserIDs:    []string{"user-123", "user-456"},
			Groups:     []string{"group-1", "group-3"},
			Percentage: &percentage,
			AllowList:  []string{"allowed-user"},
			DenyList:   []string{"denied-user"},
		}
		strategy := feature.NewTargetedStrategy(
			criteria,
			feature.WithUserIDExtractor(getUserID),
			feature.WithUserGroupsExtractor(getUserGroups),
		)
		ctx := context.Background()
		b.ResetTimer()
		for b.Loop() {
			_, _ = strategy.Evaluate(ctx)
		}
	})

	b.Run("CompositeStrategy-And", func(b *testing.B) {
		strategy := feature.NewAndStrategy(
			feature.NewAlwaysOnStrategy(),
			feature.NewEnvironmentStrategy(
				[]string{"production"},
				feature.WithEnvironmentExtractor(getEnvironment),
			),
		)
		ctx := context.Background()
		b.ResetTimer()
		for b.Loop() {
			_, _ = strategy.Evaluate(ctx)
		}
	})

	b.Run("CompositeStrategy-Or", func(b *testing.B) {
		strategy := feature.NewOrStrategy(
			feature.NewAlwaysOffStrategy(),
			feature.NewEnvironmentStrategy(
				[]string{"production"},
				feature.WithEnvironmentExtractor(getEnvironment),
			),
		)
		ctx := context.Background()
		b.ResetTimer()
		for b.Loop() {
			_, _ = strategy.Evaluate(ctx)
		}
	})
}
