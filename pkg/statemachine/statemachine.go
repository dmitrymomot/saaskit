package statemachine

import (
	"context"
)

// State represents a state in the state machine.
type State interface {
	Name() string
}

// Event represents an event that can trigger a state transition.
type Event interface {
	Name() string
}

// Action executes side effects during state transitions. Returning an error prevents the transition.
type Action func(ctx context.Context, from, to State, event Event, data any) error

// Guard evaluates whether a transition should be allowed based on runtime conditions.
type Guard func(ctx context.Context, from State, event Event, data any) bool

// Transition defines a state change triggered by an event, with optional guards and actions.
type Transition struct {
	From    State
	To      State
	Event   Event
	Guards  []Guard  // All must pass for transition to proceed
	Actions []Action // Executed in order before state change
}

// StateMachine defines the core finite state machine operations.
type StateMachine interface {
	Current() State
	AddTransition(from, to State, event Event, guards []Guard, actions []Action) error
	Fire(ctx context.Context, event Event, data any) error
	CanFire(ctx context.Context, event Event, data any) bool
	Reset() error
}

// StringState provides a simple string-based state implementation for basic use cases.
type StringState string

func (s StringState) Name() string {
	return string(s)
}

// StringEvent provides a simple string-based event implementation for basic use cases.
type StringEvent string

func (e StringEvent) Name() string {
	return string(e)
}
