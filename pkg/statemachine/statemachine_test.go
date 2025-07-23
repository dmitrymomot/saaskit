package statemachine_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/statemachine"
)

func TestSimpleStateMachine(t *testing.T) {
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
		// Create a state machine
		sm := statemachine.NewSimpleStateMachine(Draft)

		// Add transitions
		if err := sm.AddTransition(Draft, InReview, Submit, nil, nil); err != nil {
			t.Fatalf("Failed to add transition: %v", err)
		}
		if err := sm.AddTransition(InReview, Approved, Approve, nil, nil); err != nil {
			t.Fatalf("Failed to add transition: %v", err)
		}

		// Initial state should be Draft
		if sm.Current() != Draft {
			t.Fatalf("Expected initial state to be %s, got %s", Draft, sm.Current())
		}

		ctx := context.Background()

		// Fire Submit event
		if err := sm.Fire(ctx, Submit, nil); err != nil {
			t.Fatalf("Failed to fire Submit event: %v", err)
		}

		// State should now be InReview
		if sm.Current() != InReview {
			t.Fatalf("Expected state to be %s, got %s", InReview, sm.Current())
		}

		// Fire Approve event
		if err := sm.Fire(ctx, Approve, nil); err != nil {
			t.Fatalf("Failed to fire Approve event: %v", err)
		}

		// State should now be Approved
		if sm.Current() != Approved {
			t.Fatalf("Expected state to be %s, got %s", Approved, sm.Current())
		}

		// Reset the state machine
		if err := sm.Reset(); err != nil {
			t.Fatalf("Failed to reset state machine: %v", err)
		}

		// State should be back to Draft
		if sm.Current() != Draft {
			t.Fatalf("Expected state to be %s after reset, got %s", Draft, sm.Current())
		}
	})

	t.Run("Guards", func(t *testing.T) {
		// Create a state machine
		sm := statemachine.NewSimpleStateMachine(Draft)

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

		// Add transitions with guards
		if err := sm.AddTransition(
			Draft,
			InReview,
			Submit,
			[]statemachine.Guard{isAuthorized},
			nil,
		); err != nil {
			t.Fatalf("Failed to add transition with guard: %v", err)
		}

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
		// Create a state machine
		sm := statemachine.NewSimpleStateMachine(Draft)

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

		// Add transitions with actions
		if err := sm.AddTransition(
			Draft,
			InReview,
			Submit,
			nil,
			[]statemachine.Action{logAction},
		); err != nil {
			t.Fatalf("Failed to add transition with action: %v", err)
		}

		if err := sm.AddTransition(
			InReview,
			Rejected,
			Reject,
			nil,
			[]statemachine.Action{errorAction},
		); err != nil {
			t.Fatalf("Failed to add transition with error action: %v", err)
		}

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

		// Fire Reject event which should trigger an error in the action
		err := sm.Fire(ctx, Reject, nil)
		if err == nil {
			t.Fatal("Expected error from action, got nil")
		}
		if !errors.Is(err, errors.New("action error")) && !strings.Contains(err.Error(), "action error") {
			t.Fatalf("Expected error containing 'action error', got: %v", err)
		}
	})

	t.Run("Invalid Transitions", func(t *testing.T) {
		// Create a state machine
		sm := statemachine.NewSimpleStateMachine(Draft)

		// Add only some transitions
		if err := sm.AddTransition(Draft, InReview, Submit, nil, nil); err != nil {
			t.Fatalf("Failed to add transition: %v", err)
		}

		ctx := context.Background()

		// Try to fire an event with no transition
		err := sm.Fire(ctx, Approve, nil)
		if !statemachine.IsNoTransitionAvailableError(err) {
			t.Fatalf("Expected NoTransitionAvailableError, got: %v", err)
		}

		// Try to add invalid transition (nil values)
		err = sm.AddTransition(nil, InReview, Submit, nil, nil)
		if err != statemachine.ErrInvalidTransition {
			t.Fatalf("Expected ErrInvalidTransition, got: %v", err)
		}

		// Try to fire nil event
		err = sm.Fire(ctx, nil, nil)
		if err != statemachine.ErrInvalidEvent {
			t.Fatalf("Expected ErrInvalidEvent, got: %v", err)
		}
	})
}

func TestBuilder(t *testing.T) {
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

	t.Run("Basic Builder", func(t *testing.T) {
		// Create a builder
		builder := statemachine.NewBuilder(Draft)

		// Define transitions
		if _, err := builder.From(Draft).When(Submit).To(InReview).Add(); err != nil {
			t.Fatalf("Failed to add transition: %v", err)
		}

		if _, err := builder.From(InReview).When(Approve).To(Approved).Add(); err != nil {
			t.Fatalf("Failed to add transition: %v", err)
		}

		if _, err := builder.From(Approved).When(Publish).To(Published).Add(); err != nil {
			t.Fatalf("Failed to add transition: %v", err)
		}

		// Build the state machine
		machine := builder.Build()

		// Check initial state
		if machine.Current() != Draft {
			t.Fatalf("Expected initial state to be %s, got %s", Draft, machine.Current())
		}

		ctx := context.Background()

		// Execute the full workflow
		if err := machine.Fire(ctx, Submit, nil); err != nil {
			t.Fatalf("Failed to fire Submit event: %v", err)
		}

		if err := machine.Fire(ctx, Approve, nil); err != nil {
			t.Fatalf("Failed to fire Approve event: %v", err)
		}

		if err := machine.Fire(ctx, Publish, nil); err != nil {
			t.Fatalf("Failed to fire Publish event: %v", err)
		}

		// Check final state
		if machine.Current() != Published {
			t.Fatalf("Expected final state to be %s, got %s", Published, machine.Current())
		}
	})

	t.Run("Builder with Guards and Actions", func(t *testing.T) {
		// Create a builder
		builder := statemachine.NewBuilder(Draft)

		// Track action execution
		actionExecuted := false

		// Define guard and action
		isAuthorized := func(ctx context.Context, from statemachine.State, event statemachine.Event, data any) bool {
			return data.(bool)
		}

		logAction := func(ctx context.Context, from, to statemachine.State, event statemachine.Event, data any) error {
			actionExecuted = true
			return nil
		}

		// Define transition with guard and action
		if _, err := builder.From(Draft).When(Submit).To(InReview).
			WithGuard(isAuthorized).
			WithAction(logAction).
			Add(); err != nil {
			t.Fatalf("Failed to add transition: %v", err)
		}

		// Build the state machine
		machine := builder.Build()

		ctx := context.Background()

		// Try with unauthorized data
		err := machine.Fire(ctx, Submit, false)
		if !statemachine.IsTransitionRejectedError(err) {
			t.Fatalf("Expected TransitionRejectedError, got: %v", err)
		}

		// Check that action was not executed
		if actionExecuted {
			t.Fatal("Expected action not to be executed")
		}

		// Try with authorized data
		if err := machine.Fire(ctx, Submit, true); err != nil {
			t.Fatalf("Failed to fire Submit event: %v", err)
		}

		// Check that action was executed
		if !actionExecuted {
			t.Fatal("Expected action to be executed")
		}

		// Check state
		if machine.Current() != InReview {
			t.Fatalf("Expected state to be %s, got %s", InReview, machine.Current())
		}
	})

	t.Run("WithTransition Shorthand", func(t *testing.T) {
		// Create a builder
		builder := statemachine.NewBuilder(Draft)

		// Use the shorthand method
		if _, err := builder.WithTransition(
			Draft,
			InReview,
			Submit,
			func(ctx context.Context, from statemachine.State, event statemachine.Event, data any) bool {
				return true
			},
			func(ctx context.Context, from, to statemachine.State, event statemachine.Event, data any) error {
				return nil
			},
		); err != nil {
			t.Fatalf("Failed to add transition: %v", err)
		}

		// Build the state machine
		machine := builder.Build()

		ctx := context.Background()

		// Fire event
		if err := machine.Fire(ctx, Submit, nil); err != nil {
			t.Fatalf("Failed to fire Submit event: %v", err)
		}

		// Check state
		if machine.Current() != InReview {
			t.Fatalf("Expected state to be %s, got %s", InReview, machine.Current())
		}
	})
}
