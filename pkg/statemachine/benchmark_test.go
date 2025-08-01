package statemachine_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/statemachine"
)

func BenchmarkStateMachine_Fire(b *testing.B) {
	ctx := context.Background()

	// Create states
	idle := statemachine.StringState("idle")
	running := statemachine.StringState("running")
	stopped := statemachine.StringState("stopped")

	// Create events
	start := statemachine.StringEvent("start")
	stop := statemachine.StringEvent("stop")

	// Build state machine
	sm := statemachine.MustNew(idle,
		statemachine.WithTransition(idle, running, start),
		statemachine.WithTransition(running, stopped, stop),
		statemachine.WithTransition(stopped, running, start),
	)

	b.ResetTimer()

	for b.Loop() {
		// Cycle through states
		_ = sm.Fire(ctx, start, nil)
		_ = sm.Fire(ctx, stop, nil)
	}
}

func BenchmarkStateMachine_FireWithGuards(b *testing.B) {
	ctx := context.Background()

	// Create states
	idle := statemachine.StringState("idle")
	running := statemachine.StringState("running")

	// Create event
	start := statemachine.StringEvent("start")

	// Create guard that always passes
	guard := func(ctx context.Context, from statemachine.State, event statemachine.Event, data any) bool {
		return true
	}

	// Build state machine with guards
	sm := statemachine.MustNew(idle,
		statemachine.WithTransition(idle, running, start,
			statemachine.WithGuard(guard),
		),
		statemachine.WithTransition(running, idle, start,
			statemachine.WithGuard(guard),
		),
	)

	b.ResetTimer()

	for b.Loop() {
		_ = sm.Fire(ctx, start, nil)
	}
}

func BenchmarkStateMachine_FireWithActions(b *testing.B) {
	ctx := context.Background()

	// Create states
	idle := statemachine.StringState("idle")
	running := statemachine.StringState("running")

	// Create event
	start := statemachine.StringEvent("start")

	// Create action that does minimal work
	action := func(ctx context.Context, from, to statemachine.State, event statemachine.Event, data any) error {
		return nil
	}

	// Build state machine with actions
	sm := statemachine.MustNew(idle,
		statemachine.WithTransition(idle, running, start,
			statemachine.WithAction(action),
		),
		statemachine.WithTransition(running, idle, start,
			statemachine.WithAction(action),
		),
	)

	b.ResetTimer()

	for b.Loop() {
		_ = sm.Fire(ctx, start, nil)
	}
}

func BenchmarkStateMachine_CanFire(b *testing.B) {
	ctx := context.Background()

	// Create states
	idle := statemachine.StringState("idle")
	running := statemachine.StringState("running")
	stopped := statemachine.StringState("stopped")

	// Create events
	start := statemachine.StringEvent("start")
	stop := statemachine.StringEvent("stop")
	pause := statemachine.StringEvent("pause") // Invalid event for current state

	// Build state machine
	sm := statemachine.MustNew(idle,
		statemachine.WithTransition(idle, running, start),
		statemachine.WithTransition(running, stopped, stop),
	)

	b.ResetTimer()

	for b.Loop() {
		// Mix of valid and invalid checks
		_ = sm.CanFire(ctx, start, nil)
		_ = sm.CanFire(ctx, pause, nil)
	}
}

func BenchmarkStateMachine_CanFireWithGuards(b *testing.B) {
	ctx := context.Background()

	// Create states
	idle := statemachine.StringState("idle")
	running := statemachine.StringState("running")

	// Create event
	start := statemachine.StringEvent("start")

	// Create guard that checks data
	guard := func(ctx context.Context, from statemachine.State, event statemachine.Event, data any) bool {
		if data == nil {
			return false
		}
		enabled, ok := data.(bool)
		return ok && enabled
	}

	// Build state machine with guards
	sm := statemachine.MustNew(idle,
		statemachine.WithTransition(idle, running, start,
			statemachine.WithGuard(guard),
		),
	)

	b.ResetTimer()

	for b.Loop() {
		// Alternate between passing and failing guard
		_ = sm.CanFire(ctx, start, true)
		_ = sm.CanFire(ctx, start, false)
	}
}

func BenchmarkStateMachine_ConcurrentReads(b *testing.B) {
	ctx := context.Background()

	// Create states
	idle := statemachine.StringState("idle")
	running := statemachine.StringState("running")

	// Create event
	start := statemachine.StringEvent("start")

	// Build state machine
	sm := statemachine.MustNew(idle,
		statemachine.WithTransition(idle, running, start),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Concurrent read operations
			_ = sm.Current()
			_ = sm.CanFire(ctx, start, nil)
		}
	})
}

func BenchmarkStateMachine_ConcurrentWrites(b *testing.B) {
	ctx := context.Background()

	// Create states
	idle := statemachine.StringState("idle")
	running := statemachine.StringState("running")

	// Create events
	start := statemachine.StringEvent("start")
	stop := statemachine.StringEvent("stop")

	// Build state machine
	sm := statemachine.MustNew(idle,
		statemachine.WithTransition(idle, running, start),
		statemachine.WithTransition(running, idle, stop),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Alternate between start and stop to cycle states
			_ = sm.Fire(ctx, start, nil)
			_ = sm.Fire(ctx, stop, nil)
		}
	})
}

func BenchmarkStateMachine_MixedConcurrentAccess(b *testing.B) {
	ctx := context.Background()

	// Create states
	idle := statemachine.StringState("idle")
	running := statemachine.StringState("running")
	paused := statemachine.StringState("paused")

	// Create events
	start := statemachine.StringEvent("start")
	pause := statemachine.StringEvent("pause")
	resume := statemachine.StringEvent("resume")
	stop := statemachine.StringEvent("stop")

	// Build state machine
	sm := statemachine.MustNew(idle,
		statemachine.WithTransition(idle, running, start),
		statemachine.WithTransition(running, paused, pause),
		statemachine.WithTransition(paused, running, resume),
		statemachine.WithTransition(running, idle, stop),
		statemachine.WithTransition(paused, idle, stop),
	)

	b.ResetTimer()

	// Create multiple goroutines with different access patterns
	var wg sync.WaitGroup

	// Writers
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < b.N/4; j++ {
				_ = sm.Fire(ctx, start, nil)
				_ = sm.Fire(ctx, pause, nil)
				_ = sm.Fire(ctx, resume, nil)
				_ = sm.Fire(ctx, stop, nil)
			}
		}()
	}

	// Readers
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < b.N/2; j++ {
				_ = sm.Current()
				_ = sm.CanFire(ctx, start, nil)
				_ = sm.CanFire(ctx, pause, nil)
			}
		}()
	}

	wg.Wait()
}

func BenchmarkOptions_Construction(b *testing.B) {
	// Create states
	idle := statemachine.StringState("idle")
	running := statemachine.StringState("running")
	stopped := statemachine.StringState("stopped")

	// Create events
	start := statemachine.StringEvent("start")
	stop := statemachine.StringEvent("stop")
	reset := statemachine.StringEvent("reset")

	for b.Loop() {
		// Build a complete state machine
		_ = statemachine.MustNew(idle,
			statemachine.WithTransition(idle, running, start),
			statemachine.WithTransition(running, stopped, stop),
			statemachine.WithTransition(stopped, idle, reset),
		)
	}
}

func BenchmarkOptions_ConstructionWithGuardsAndActions(b *testing.B) {
	// Create states
	idle := statemachine.StringState("idle")
	running := statemachine.StringState("running")

	// Create event
	start := statemachine.StringEvent("start")

	// Create guard
	guard := func(ctx context.Context, from statemachine.State, event statemachine.Event, data any) bool {
		return true
	}

	// Create action
	action := func(ctx context.Context, from, to statemachine.State, event statemachine.Event, data any) error {
		return nil
	}

	for b.Loop() {
		// Build state machine with guards and actions
		_ = statemachine.MustNew(idle,
			statemachine.WithTransition(idle, running, start,
				statemachine.WithGuard(guard),
				statemachine.WithAction(action),
			),
		)
	}
}

func BenchmarkStateMachine_LargeTransitionTable(b *testing.B) {
	ctx := context.Background()

	// Create many states
	states := make([]statemachine.State, 10)
	for i := range 10 {
		states[i] = statemachine.StringState(fmt.Sprintf("state%d", i))
	}

	// Create events
	events := make([]statemachine.Event, 5)
	for i := range 5 {
		events[i] = statemachine.StringEvent(fmt.Sprintf("event%d", i))
	}

	// Build state machine with many transitions
	var transitions []statemachine.TransitionDef
	for i := range 9 {
		for j := range 5 {
			nextState := states[(i+j+1)%10]
			transitions = append(transitions, statemachine.TransitionDef{
				From:  states[i],
				To:    nextState,
				Event: events[j],
			})
		}
	}
	sm := statemachine.MustNew(states[0],
		statemachine.WithTransitions(transitions),
	)

	b.ResetTimer()

	// Cycle through different events
	eventIndex := 0
	for b.Loop() {
		_ = sm.Fire(ctx, events[eventIndex%5], nil)
		eventIndex++
	}
}
