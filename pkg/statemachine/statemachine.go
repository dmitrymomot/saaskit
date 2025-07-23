package statemachine

import (
	"context"
)

// State is an interface that represents a state in the state machine.
type State interface {
	// Name returns the name of the state.
	Name() string
}

// Event is an interface that represents an event that can trigger a state transition.
type Event interface {
	// Name returns the name of the event.
	Name() string
}

// Action is a function that is executed during a state transition.
type Action func(ctx context.Context, from, to State, event Event, data any) error

// Guard is a function that determines if a transition can occur.
type Guard func(ctx context.Context, from State, event Event, data any) bool

// Transition represents a possible transition between states.
type Transition struct {
	From    State
	To      State
	Event   Event
	Guards  []Guard
	Actions []Action
}

// StateMachine is an interface that defines the behavior of a state machine.
type StateMachine interface {
	// Current returns the current state of the state machine.
	Current() State

	// AddTransition adds a new transition to the state machine.
	AddTransition(from, to State, event Event, guards []Guard, actions []Action) error

	// Fire triggers an event in the state machine, potentially causing a state transition.
	Fire(ctx context.Context, event Event, data any) error

	// CanFire checks if an event can be fired in the current state.
	CanFire(ctx context.Context, event Event, data any) bool

	// Reset resets the state machine to its initial state.
	Reset() error
}

// StringState is a simple implementation of the State interface.
type StringState string

// Name returns the name of the string state.
func (s StringState) Name() string {
	return string(s)
}

// StringEvent is a simple implementation of the Event interface.
type StringEvent string

// Name returns the name of the string event.
func (e StringEvent) Name() string {
	return string(e)
}
