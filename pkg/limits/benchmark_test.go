package limits_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/dmitrymomot/saaskit/pkg/limits"
)

func BenchmarkService_CanCreate(b *testing.B) {
	// Setup
	plans := map[string]limits.Plan{
		"test": {
			ID:   "test",
			Name: "Test Plan",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    100,
				limits.ResourceProjects: 50,
			},
		},
	}

	source := limits.NewInMemSource(plans)
	counters := limits.NewRegistry()
	counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		return 50, nil
	})
	counters.Register(limits.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		return 25, nil
	})

	svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
	if err != nil {
		b.Fatal(err)
	}

	ctx := limits.SetPlanIDToContext(context.Background(), "test")
	tenantID := uuid.New()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = svc.CanCreate(ctx, tenantID, limits.ResourceUsers)
	}
}

func BenchmarkService_HasFeature(b *testing.B) {
	// Setup
	plans := map[string]limits.Plan{
		"test": {
			ID:       "test",
			Name:     "Test Plan",
			Features: []limits.Feature{limits.FeatureAI, limits.FeatureSSO, "feature3", "feature4", "feature5"},
		},
	}

	source := limits.NewInMemSource(plans)
	svc, err := limits.NewLimitsService(context.Background(), source, nil, nil)
	if err != nil {
		b.Fatal(err)
	}

	ctx := limits.SetPlanIDToContext(context.Background(), "test")
	tenantID := uuid.New()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = svc.HasFeature(ctx, tenantID, limits.FeatureSSO)
	}
}

func BenchmarkService_GetUsage(b *testing.B) {
	// Setup
	plans := map[string]limits.Plan{
		"test": {
			ID:   "test",
			Name: "Test Plan",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers: 100,
			},
		},
	}

	source := limits.NewInMemSource(plans)
	counters := limits.NewRegistry()
	counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		// Simulate some work
		sum := int64(0)
		for i := range 10 {
			sum += int64(i)
		}
		return sum, nil
	})

	svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
	if err != nil {
		b.Fatal(err)
	}

	ctx := limits.SetPlanIDToContext(context.Background(), "test")
	tenantID := uuid.New()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _, _ = svc.GetUsage(ctx, tenantID, limits.ResourceUsers)
	}
}

func BenchmarkService_GetAllUsage(b *testing.B) {
	// Setup with multiple resources
	plans := map[string]limits.Plan{
		"test": {
			ID:   "test",
			Name: "Test Plan",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    100,
				limits.ResourceProjects: 50,
				"resource3":             1000,
				"resource4":             2000,
				"resource5":             3000,
			},
		},
	}

	source := limits.NewInMemSource(plans)
	counters := limits.NewRegistry()

	// Register counters for all resources
	for _, res := range []limits.Resource{limits.ResourceUsers, limits.ResourceProjects, "resource3", "resource4", "resource5"} {
		resource := res // Capture loop variable
		counters.Register(resource, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
			return 10, nil
		})
	}

	svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
	if err != nil {
		b.Fatal(err)
	}

	ctx := limits.SetPlanIDToContext(context.Background(), "test")
	tenantID := uuid.New()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = svc.GetAllUsage(ctx, tenantID)
	}
}

func BenchmarkContext_SetGet(b *testing.B) {
	ctx := context.Background()
	planID := "test-plan"

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		ctx := limits.SetPlanIDToContext(ctx, planID)
		_, _ = limits.GetPlanIDFromContext(ctx)
	}
}

func BenchmarkRegistry_Counter(b *testing.B) {
	registry := limits.NewRegistry()
	registry.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		return 42, nil
	})

	ctx := context.Background()
	tenantID := uuid.New()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		counter := registry[limits.ResourceUsers]
		_, _ = counter(ctx, tenantID)
	}
}

func BenchmarkInMemSource_Load(b *testing.B) {
	// Create a source with multiple plans
	plans := make(map[string]limits.Plan)
	for i := range 10 {
		planID := uuid.New().String()
		plans[planID] = limits.Plan{
			ID:   planID,
			Name: "Plan " + planID,
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    int64(i * 10),
				limits.ResourceProjects: int64(i * 5),
			},
			Features:  []limits.Feature{limits.FeatureAI, limits.FeatureSSO},
			TrialDays: i,
		}
	}

	source := limits.NewInMemSource(plans)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = source.Load(ctx)
	}
}

func BenchmarkService_Parallel(b *testing.B) {
	// Setup
	plans := map[string]limits.Plan{
		"test": {
			ID:   "test",
			Name: "Test Plan",
			Limits: map[limits.Resource]int64{
				limits.ResourceUsers:    100,
				limits.ResourceProjects: 50,
			},
			Features: []limits.Feature{limits.FeatureAI, limits.FeatureSSO},
		},
	}

	source := limits.NewInMemSource(plans)
	counters := limits.NewRegistry()
	counters.Register(limits.ResourceUsers, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		return 50, nil
	})
	counters.Register(limits.ResourceProjects, func(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		return 25, nil
	})

	svc, err := limits.NewLimitsService(context.Background(), source, counters, nil)
	if err != nil {
		b.Fatal(err)
	}

	ctx := limits.SetPlanIDToContext(context.Background(), "test")
	tenantID := uuid.New()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Mix of different operations
			_ = svc.CanCreate(ctx, tenantID, limits.ResourceUsers)
			_ = svc.HasFeature(ctx, tenantID, limits.FeatureAI)
			_, _, _ = svc.GetUsage(ctx, tenantID, limits.ResourceProjects)
		}
	})
}
