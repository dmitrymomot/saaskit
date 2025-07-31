package statemachine_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/statemachine"
)

func TestStateMachine(t *testing.T) {
	t.Parallel()
	// Define states
	const (
		Draft     = statemachine.StringState("draft")
		InReview  = statemachine.StringState("in_review")
		Approved  = statemachine.StringState("approved")
		Published = statemachine.StringState("published")
		Rejected  = statemachine.StringState("rejected")
	)

	// Define events
	const (
		Submit   = statemachine.StringEvent("submit")
		Approve  = statemachine.StringEvent("approve")
		Reject   = statemachine.StringEvent("reject")
		Publish  = statemachine.StringEvent("publish")
		Withdraw = statemachine.StringEvent("withdraw")
	)

	t.Run("Basic Transitions", func(t *testing.T) {
		t.Parallel()
		// Create a state machine
		sm := statemachine.MustNew(Draft,
			statemachine.WithTransition(Draft, InReview, Submit),
			statemachine.WithTransition(InReview, Approved, Approve),
		)

		// Initial state should be Draft
		if sm.Current() != Draft {
			t.Fatalf("Expected initial state to be %s, got %s", Draft, sm.Current())
		}

		ctx := context.Background()

		// Test CanFire
		if !sm.CanFire(ctx, Submit, nil) {
			t.Fatal("Expected CanFire to return true for Submit event in Draft state")
		}

		if err := sm.Fire(ctx, Submit, nil); err != nil {
			t.Fatalf("Failed to fire Submit event: %v", err)
		}

		// State should be InReview
		if sm.Current() != InReview {
			t.Fatalf("Expected state to be %s, got %s", InReview, sm.Current())
		}

		if err := sm.Fire(ctx, Approve, nil); err != nil {
			t.Fatalf("Failed to fire Approve event: %v", err)
		}

		// State should be Approved
		if sm.Current() != Approved {
			t.Fatalf("Expected state to be %s, got %s", Approved, sm.Current())
		}

		if err := sm.Reset(); err != nil {
			t.Fatalf("Failed to reset state machine: %v", err)
		}

		// State should be back to Draft
		if sm.Current() != Draft {
			t.Fatalf("Expected state to be %s after reset, got %s", Draft, sm.Current())
		}
	})

	t.Run("Guards", func(t *testing.T) {
		t.Parallel()
		// Define guards
		isAuthorized := func(ctx context.Context, from statemachine.State, event statemachine.Event, data any) bool {
			userData, ok := data.(map[string]any)
			if !ok {
				return false
			}
			isAuth, ok := userData["authorized"].(bool)
			if !ok {
				return false
			}
			return isAuth
		}

		// Create a state machine with guarded transition
		sm := statemachine.MustNew(Draft,
			statemachine.WithTransition(Draft, InReview, Submit,
				statemachine.WithGuard(isAuthorized),
			),
		)

		ctx := context.Background()

		// Test with unauthorized data
		unauthorizedData := map[string]any{
			"authorized": false,
		}

		// CanFire should return false
		if sm.CanFire(ctx, Submit, unauthorizedData) {
			t.Fatal("Expected CanFire to return false for unauthorized data")
		}

		// Fire should fail with unauthorized data
		err := sm.Fire(ctx, Submit, unauthorizedData)
		if !statemachine.IsTransitionRejectedError(err) {
			t.Fatalf("Expected TransitionRejectedError, got: %v", err)
		}

		// Test with authorized data
		authorizedData := map[string]any{
			"authorized": true,
		}

		// CanFire should return true
		if !sm.CanFire(ctx, Submit, authorizedData) {
			t.Fatal("Expected CanFire to return true for authorized data")
		}

		// Fire should succeed with authorized data
		if err := sm.Fire(ctx, Submit, authorizedData); err != nil {
			t.Fatalf("Failed to fire Submit event with authorized data: %v", err)
		}

		// State should be InReview
		if sm.Current() != InReview {
			t.Fatalf("Expected state to be %s, got %s", InReview, sm.Current())
		}
	})

	t.Run("Actions", func(t *testing.T) {
		t.Parallel()
		// Track action execution
		actionExecuted := false
		actionData := ""

		// Define action
		logAction := func(ctx context.Context, from, to statemachine.State, event statemachine.Event, data any) error {
			actionExecuted = true
			if str, ok := data.(string); ok {
				actionData = str
			}
			return nil
		}

		errorAction := func(ctx context.Context, from, to statemachine.State, event statemachine.Event, data any) error {
			return errors.New("action error")
		}

		// Create a state machine with actions
		sm := statemachine.MustNew(Draft,
			statemachine.WithTransition(Draft, InReview, Submit,
				statemachine.WithAction(logAction),
			),
			statemachine.WithTransition(InReview, Rejected, Reject,
				statemachine.WithAction(errorAction),
			),
		)

		ctx := context.Background()
		testData := "test-data"

		// Fire Submit event
		if err := sm.Fire(ctx, Submit, testData); err != nil {
			t.Fatalf("Failed to fire Submit event: %v", err)
		}

		// Check that action was executed
		if !actionExecuted {
			t.Fatal("Expected action to be executed")
		}

		// Check that action received correct data
		if actionData != testData {
			t.Fatalf("Expected action data to be %s, got %s", testData, actionData)
		}

		// Fire Reject event (should fail due to error action)
		err := sm.Fire(ctx, Reject, nil)
		if err == nil {
			t.Fatal("Expected error from action")
		}

		// Check that error message contains "action failed"
		if !strings.Contains(err.Error(), "action failed") {
			t.Fatalf("Expected error to contain 'action failed', got: %v", err)
		}
	})

	t.Run("Error Handling", func(t *testing.T) {
		t.Parallel()
		// Create a state machine with limited transitions
		sm := statemachine.MustNew(Draft,
			statemachine.WithTransition(Draft, InReview, Submit),
		)

		ctx := context.Background()

		// Try to fire an event with no transition
		err := sm.Fire(ctx, Approve, nil)
		if !statemachine.IsNoTransitionAvailableError(err) {
			t.Fatalf("Expected NoTransitionAvailableError, got: %v", err)
		}

		// Test nil initial state
		_, err = statemachine.New(nil)
		if err == nil || !strings.Contains(err.Error(), "initial state cannot be nil") {
			t.Fatalf("Expected error for nil initial state, got: %v", err)
		}

		// Try to fire nil event
		err = sm.Fire(ctx, nil, nil)
		if err != statemachine.ErrInvalidEvent {
			t.Fatalf("Expected ErrInvalidEvent, got: %v", err)
		}
	})

	t.Run("MustNew Panic", func(t *testing.T) {
		t.Parallel()
		// Test that MustNew panics on invalid configuration
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("Expected MustNew to panic with nil initial state")
			}
		}()

		_ = statemachine.MustNew(nil)
	})
}

func TestOptionsPattern(t *testing.T) {
	t.Parallel()
	// Define states
	const (
		Draft     = statemachine.StringState("draft")
		InReview  = statemachine.StringState("in_review")
		Approved  = statemachine.StringState("approved")
		Published = statemachine.StringState("published")
	)

	// Define events
	const (
		Submit  = statemachine.StringEvent("submit")
		Approve = statemachine.StringEvent("approve")
		Publish = statemachine.StringEvent("publish")
	)

	t.Run("Basic Options", func(t *testing.T) {
		t.Parallel()
		// Create state machine with options
		sm := statemachine.MustNew(Draft,
			statemachine.WithTransition(Draft, InReview, Submit),
			statemachine.WithTransition(InReview, Approved, Approve),
			statemachine.WithTransition(Approved, Published, Publish),
		)

		// Check initial state
		if sm.Current() != Draft {
			t.Fatalf("Expected initial state to be %s, got %s", Draft, sm.Current())
		}

		ctx := context.Background()

		// Execute the full workflow
		if err := sm.Fire(ctx, Submit, nil); err != nil {
			t.Fatalf("Failed to fire Submit event: %v", err)
		}

		if err := sm.Fire(ctx, Approve, nil); err != nil {
			t.Fatalf("Failed to fire Approve event: %v", err)
		}

		if err := sm.Fire(ctx, Publish, nil); err != nil {
			t.Fatalf("Failed to fire Publish event: %v", err)
		}

		// Final state should be Published
		if sm.Current() != Published {
			t.Fatalf("Expected final state to be %s, got %s", Published, sm.Current())
		}
	})

	t.Run("Options with Guards and Actions", func(t *testing.T) {
		t.Parallel()
		// Track execution
		actionExecuted := false

		isAuthorized := func(ctx context.Context, from statemachine.State, event statemachine.Event, data any) bool {
			return data.(bool)
		}

		logAction := func(ctx context.Context, from, to statemachine.State, event statemachine.Event, data any) error {
			actionExecuted = true
			return nil
		}

		// Create state machine with guards and actions
		sm := statemachine.MustNew(Draft,
			statemachine.WithTransition(Draft, InReview, Submit,
				statemachine.WithGuard(isAuthorized),
				statemachine.WithAction(logAction),
			),
		)

		ctx := context.Background()

		// Fire with valid data
		if err := sm.Fire(ctx, Submit, true); err != nil {
			t.Fatalf("Failed to fire Submit event: %v", err)
		}

		// Check that action was executed
		if !actionExecuted {
			t.Fatal("Expected action to be executed")
		}

		// Check state changed
		if sm.Current() != InReview {
			t.Fatalf("Expected state to be %s, got %s", InReview, sm.Current())
		}
	})

	t.Run("WithTransitions Bulk", func(t *testing.T) {
		t.Parallel()
		// Define transitions
		transitions := []statemachine.TransitionDef{
			{From: Draft, To: InReview, Event: Submit},
			{From: InReview, To: Approved, Event: Approve},
			{From: Approved, To: Published, Event: Publish},
		}

		// Create state machine with bulk transitions
		sm := statemachine.MustNew(Draft,
			statemachine.WithTransitions(transitions),
		)

		ctx := context.Background()

		// Execute workflow
		if err := sm.Fire(ctx, Submit, nil); err != nil {
			t.Fatalf("Failed to fire Submit event: %v", err)
		}

		if sm.Current() != InReview {
			t.Fatalf("Expected state to be %s, got %s", InReview, sm.Current())
		}
	})

	t.Run("Multiple Guards", func(t *testing.T) {
		t.Parallel()
		// Define multiple guards
		hasRole := func(ctx context.Context, from statemachine.State, event statemachine.Event, data any) bool {
			userData, ok := data.(map[string]any)
			if !ok {
				return false
			}
			role, ok := userData["role"].(string)
			return ok && role == "admin"
		}

		isEnabled := func(ctx context.Context, from statemachine.State, event statemachine.Event, data any) bool {
			userData, ok := data.(map[string]any)
			if !ok {
				return false
			}
			enabled, ok := userData["enabled"].(bool)
			return ok && enabled
		}

		// Create state machine with multiple guards
		sm := statemachine.MustNew(Draft,
			statemachine.WithTransition(Draft, InReview, Submit,
				statemachine.WithGuards(hasRole, isEnabled),
			),
		)

		ctx := context.Background()

		// Test with partial data (should fail)
		partialData := map[string]any{
			"role": "admin",
		}

		if sm.CanFire(ctx, Submit, partialData) {
			t.Fatal("Expected CanFire to return false with partial data")
		}

		// Test with complete valid data
		validData := map[string]any{
			"role":    "admin",
			"enabled": true,
		}

		if !sm.CanFire(ctx, Submit, validData) {
			t.Fatal("Expected CanFire to return true with valid data")
		}

		// Fire event
		if err := sm.Fire(ctx, Submit, validData); err != nil {
			t.Fatalf("Failed to fire Submit event: %v", err)
		}
	})

	t.Run("Multiple Actions", func(t *testing.T) {
		t.Parallel()
		// Track action execution
		var executionOrder []string

		action1 := func(ctx context.Context, from, to statemachine.State, event statemachine.Event, data any) error {
			executionOrder = append(executionOrder, "action1")
			return nil
		}

		action2 := func(ctx context.Context, from, to statemachine.State, event statemachine.Event, data any) error {
			executionOrder = append(executionOrder, "action2")
			return nil
		}

		// Create state machine with multiple actions
		sm := statemachine.MustNew(Draft,
			statemachine.WithTransition(Draft, InReview, Submit,
				statemachine.WithActions(action1, action2),
			),
		)

		ctx := context.Background()

		// Fire event
		if err := sm.Fire(ctx, Submit, nil); err != nil {
			t.Fatalf("Failed to fire Submit event: %v", err)
		}

		// Check execution order
		if len(executionOrder) != 2 {
			t.Fatalf("Expected 2 actions to be executed, got %d", len(executionOrder))
		}

		if executionOrder[0] != "action1" || executionOrder[1] != "action2" {
			t.Fatalf("Expected execution order [action1, action2], got %v", executionOrder)
		}
	})

	t.Run("Invalid Transition in WithTransitions", func(t *testing.T) {
		t.Parallel()
		// Define transitions with nil state
		transitions := []statemachine.TransitionDef{
			{From: nil, To: InReview, Event: Submit},
		}

		// Should return error
		_, err := statemachine.New(Draft,
			statemachine.WithTransitions(transitions),
		)

		if err == nil {
			t.Fatal("Expected error for invalid transition")
		}

		if !strings.Contains(err.Error(), "failed to add transition") {
			t.Fatalf("Expected error to contain 'failed to add transition', got: %v", err)
		}
	})
}

// Custom state implementation for testing
type OrderState struct {
	status string
	code   int
}

func (s OrderState) Name() string {
	return s.status
}

// Custom event implementation for testing
type OrderEvent struct {
	action string
	user   string
}

func (e OrderEvent) Name() string {
	return e.action
}

func TestCustomStateAndEvent(t *testing.T) {
	t.Parallel()

	// Define states
	pending := OrderState{status: "pending", code: 1}
	processing := OrderState{status: "processing", code: 2}
	completed := OrderState{status: "completed", code: 3}

	// Define events
	startProcessing := OrderEvent{action: "start_processing", user: "system"}
	completeOrder := OrderEvent{action: "complete", user: "system"}

	// Create state machine
	sm := statemachine.MustNew(pending,
		statemachine.WithTransition(pending, processing, startProcessing),
		statemachine.WithTransition(processing, completed, completeOrder),
	)

	ctx := context.Background()

	// Test transitions
	if err := sm.Fire(ctx, startProcessing, nil); err != nil {
		t.Fatalf("Failed to fire startProcessing event: %v", err)
	}

	currentState := sm.Current().(OrderState)
	if currentState.status != "processing" || currentState.code != 2 {
		t.Fatalf("Expected processing state, got %+v", currentState)
	}

	if err := sm.Fire(ctx, completeOrder, nil); err != nil {
		t.Fatalf("Failed to fire completeOrder event: %v", err)
	}

	finalState := sm.Current().(OrderState)
	if finalState.status != "completed" || finalState.code != 3 {
		t.Fatalf("Expected completed state, got %+v", finalState)
	}
}

func TestConcurrency(t *testing.T) {
	t.Parallel()
	// Define states
	const (
		State1 = statemachine.StringState("state1")
		State2 = statemachine.StringState("state2")
		State3 = statemachine.StringState("state3")
	)

	// Define events
	const (
		Event1 = statemachine.StringEvent("event1")
		Event2 = statemachine.StringEvent("event2")
		Event3 = statemachine.StringEvent("event3")
	)

	// Create state machine
	sm := statemachine.MustNew(State1,
		statemachine.WithTransition(State1, State2, Event1),
		statemachine.WithTransition(State2, State3, Event2),
		statemachine.WithTransition(State3, State1, Event3),
	)

	ctx := context.Background()

	// Run concurrent operations
	done := make(chan bool)

	// Multiple readers
	for range 5 {
		go func() {
			for range 100 {
				_ = sm.Current()
				_ = sm.CanFire(ctx, Event1, nil)
			}
			done <- true
		}()
	}

	// Multiple writers
	for range 2 {
		go func() {
			for range 50 {
				_ = sm.Fire(ctx, Event1, nil)
				_ = sm.Fire(ctx, Event2, nil)
				_ = sm.Fire(ctx, Event3, nil)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 7 {
		<-done
	}

	// Verify state machine is still functional
	if err := sm.Reset(); err != nil {
		t.Fatalf("Failed to reset after concurrent operations: %v", err)
	}

	if sm.Current() != State1 {
		t.Fatalf("Expected state to be %s after reset, got %s", State1, sm.Current())
	}
}

func TestEdgeCases(t *testing.T) {
	t.Parallel()
	// Define states
	const (
		State1 = statemachine.StringState("state1")
		State2 = statemachine.StringState("state2")
	)

	// Define events
	const (
		Event1 = statemachine.StringEvent("event1")
	)

	t.Run("Nil Guard Handling", func(t *testing.T) {
		t.Parallel()
		// Create a state machine with nil guard
		sm := statemachine.MustNew(State1,
			statemachine.WithTransition(State1, State2, Event1,
				statemachine.WithGuard(nil),
			),
		)

		ctx := context.Background()

		// Should handle nil guard gracefully (treated as always passing)
		if !sm.CanFire(ctx, Event1, nil) {
			t.Fatal("Expected CanFire to return true with nil guard")
		}

		if err := sm.Fire(ctx, Event1, nil); err != nil {
			t.Fatalf("Failed to fire event with nil guard: %v", err)
		}

		if sm.Current() != State2 {
			t.Fatalf("Expected state to be %s, got %s", State2, sm.Current())
		}
	})

	t.Run("Nil Action Handling", func(t *testing.T) {
		t.Parallel()
		// Create a state machine with nil action
		sm := statemachine.MustNew(State1,
			statemachine.WithTransition(State1, State2, Event1,
				statemachine.WithAction(nil),
			),
		)

		ctx := context.Background()

		// Should handle nil action gracefully (no-op)
		if err := sm.Fire(ctx, Event1, nil); err != nil {
			t.Fatalf("Failed to fire event with nil action: %v", err)
		}

		if sm.Current() != State2 {
			t.Fatalf("Expected state to be %s, got %s", State2, sm.Current())
		}
	})

	t.Run("Empty Payload Handling", func(t *testing.T) {
		t.Parallel()
		// Create a state machine with guard and action that check payload
		guardCalled := false
		actionCalled := false

		sm := statemachine.MustNew(State1,
			statemachine.WithTransition(State1, State2, Event1,
				statemachine.WithGuard(func(ctx context.Context, from statemachine.State, event statemachine.Event, data any) bool {
					guardCalled = true
					// Should handle nil payload gracefully
					return true
				}),
				statemachine.WithAction(func(ctx context.Context, from, to statemachine.State, event statemachine.Event, data any) error {
					actionCalled = true
					// Should handle nil payload gracefully
					return nil
				}),
			),
		)

		ctx := context.Background()

		// Fire with nil payload
		if err := sm.Fire(ctx, Event1, nil); err != nil {
			t.Fatalf("Failed to fire event with nil payload: %v", err)
		}

		if !guardCalled {
			t.Fatal("Guard was not called")
		}

		if !actionCalled {
			t.Fatal("Action was not called")
		}

		if sm.Current() != State2 {
			t.Fatalf("Expected state to be %s, got %s", State2, sm.Current())
		}
	})
}
