# State Machine Package

A flexible, type-safe state machine implementation for Go applications.

## Installation

```bash
go get github.com/dmitrymomot/saaskit/pkg/statemachine
```

## Overview 

The `statemachine` package provides a clean, flexible implementation of the state machine pattern for Go applications. It offers a fluent builder API and type-safety through interfaces, making it ideal for modeling complex workflows, business processes, and application states. This package is thread-safe and suitable for concurrent use in production environments.

## Features

- Fluent builder pattern for intuitive state machine construction
- Type-safe implementation using Go interfaces
- Support for guards (transition conditions) and actions (side effects)
- Thread-safe operation for concurrent access
- String-based or custom state/event implementations
- Comprehensive error handling with specific error types
- Simple API with minimal boilerplate

## Usage

### Basic Example

```go
import (
	"context"
	"github.com/dmitrymomot/saaskit/pkg/statemachine"
)

// Define states using string constants
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

// Create and configure the state machine using the fluent builder API
builder := statemachine.NewBuilder(Draft)
builder.From(Draft).When(Submit).To(InReview).Add()
builder.From(InReview).When(Approve).To(Approved).Add()
builder.From(InReview).When(Reject).To(Rejected).Add()
builder.From(Approved).When(Publish).To(Published).Add()
builder.From(Approved).When(Withdraw).To(Draft).Add()
builder.From(Rejected).When(Submit).To(InReview).Add()

machine := builder.Build()

// Use the state machine
ctx := context.Background()
// Current state: "draft"

// Trigger transitions
machine.Fire(ctx, Submit, nil)
// Current state: "in_review"

machine.Fire(ctx, Approve, nil)
// Current state: "approved"
```

### Guards and Actions

```go
import (
	"context"
	"github.com/dmitrymomot/saaskit/pkg/statemachine"
)

// Add conditional transition with a guard
isAuthorized := func(ctx context.Context, from statemachine.State, event statemachine.Event, data any) bool {
	userData, ok := data.(map[string]any)
	return ok && userData["is_authorized"].(bool)
}

// Add a side effect with an action
logTransition := func(ctx context.Context, from, to statemachine.State, event statemachine.Event, data any) error {
	// Log transition details
	return nil
}

// Create builder
builder := statemachine.NewBuilder(statemachine.StringState("idle"))

// Add a transition with guard and action
builder.From(statemachine.StringState("idle")).
	When(statemachine.StringEvent("start")).
	To(statemachine.StringState("running")).
	WithGuard(isAuthorized).
	WithAction(logTransition).
	Add()

machine := builder.Build()

// Fire event with context data
userData := map[string]any{"is_authorized": true, "user_id": 123}
err := machine.Fire(context.Background(), statemachine.StringEvent("start"), userData)
// If authorized, state changes to "running" and action is executed

// Try with unauthorized data
unauthorizedData := map[string]any{"is_authorized": false, "user_id": 456}
err = machine.Fire(context.Background(), statemachine.StringEvent("start"), unauthorizedData)
// err will be a TransitionRejectedError and state remains unchanged
```

### Custom State and Event Types

```go
import (
	"context"
	"github.com/dmitrymomot/saaskit/pkg/statemachine"
)

// Define custom state type
type OrderState struct {
	code        string
	description string
}

func (s OrderState) Name() string {
	return s.code
}

// Define custom event type
type OrderEvent struct {
	code string
	data map[string]any
}

func (e OrderEvent) Name() string {
	return e.code
}

// Create states and events
new := OrderState{code: "new", description: "Order created"}
processing := OrderState{code: "processing", description: "In process"}
shipped := OrderState{code: "shipped", description: "Shipped"}

process := OrderEvent{code: "process"}
ship := OrderEvent{code: "ship"}

// Configure state machine using standard API
machine := statemachine.NewSimpleStateMachine(new)
machine.AddTransition(new, processing, process, nil, nil)
machine.AddTransition(processing, shipped, ship, nil, nil)

// Use the state machine
// Current state: "new"
machine.Fire(context.Background(), process, nil)
// Current state: "processing"
```

### Error Handling

```go
import (
	"context"
	"github.com/dmitrymomot/saaskit/pkg/statemachine"
)

// Setup a simple state machine
const (
	Initial = statemachine.StringState("initial")
	Final   = statemachine.StringState("final")
	Event   = statemachine.StringEvent("event")
	InvalidEvent = statemachine.StringEvent("invalid")
)

builder := statemachine.NewBuilder(Initial)
machine := builder.Build()

// Try an invalid event (no transition defined)
err := machine.Fire(context.Background(), InvalidEvent, nil)
if err != nil {
	if statemachine.IsNoTransitionAvailableError(err) {
		// Handle "no transition available" error
		// Error message: "no transition available from state 'initial' for event 'invalid'"
	}
}

// Add a transition with a guard that always rejects
alwaysFalse := func(ctx context.Context, from statemachine.State, event statemachine.Event, data any) bool {
	return false
}

builder.From(Initial).When(Event).To(Final).WithGuard(alwaysFalse).Add()
machine = builder.Build()

// Try a transition that will be rejected by the guard
err = machine.Fire(context.Background(), Event, nil)
if err != nil {
	if statemachine.IsTransitionRejectedError(err) {
		// Handle "transition rejected" error
		// Error message: "transition from state 'initial' for event 'event' was rejected by guards"
	}
}
```

## Best Practices

1. **State Machine Design**:
    - Keep your state machines small and focused on a single responsibility
    - Use descriptive names for states and events
    - Document the allowed transitions in your code
    - Consider using a diagram tool to visualize complex state machines

2. **Guards and Actions**:
    - Keep guards simple - they should only check conditions, not modify state
    - Actions should handle side effects but avoid changing the state machine itself
    - Handle errors from actions appropriately

3. **Thread Safety**:
    - The state machine is thread-safe internally, but ensure your guards and actions are also thread-safe
    - Consider locking if you're accessing shared resources in guards or actions

4. **Error Handling**:
    - Use the error type checking functions rather than comparing error strings
    - Handle each error type appropriately in your application
    - Log state transition errors for debugging

## API Reference

### Types

```go
type State interface {
	Name() string
}
```

Interface for state objects. Implement this for custom states.

```go
type Event interface {
	Name() string
}
```

Interface for event objects. Implement this for custom events.

```go
type Guard func(ctx context.Context, from State, event Event, data any) bool
```

Function type for conditional transitions. Returns true if the transition is allowed.

```go
type Action func(ctx context.Context, from, to State, event Event, data any) error
```

Function type for side effects during transitions. Return an error to abort the transition.

```go
type Transition struct {
	From    State
	To      State
	Event   Event
	Guards  []Guard
	Actions []Action
}
```

Structure representing a possible state change in the state machine.

```go
type StateMachine interface {
	Current() State
	AddTransition(from, to State, event Event, guards []Guard, actions []Action) error
	Fire(ctx context.Context, event Event, data any) error
	CanFire(ctx context.Context, event Event, data any) bool
	Reset() error
}
```

Core interface for state machine implementations.

### Functions

```go
func NewBuilder(initialState State) *Builder
```

Creates a new state machine builder with the specified initial state.

```go
func NewSimpleStateMachine(initialState State) StateMachine
```

Creates a new simple state machine with the specified initial state.

```go
func StringState(name string) State
```

Creates a simple string-based state implementation.

```go
func StringEvent(name string) Event
```

Creates a simple string-based event implementation.

```go
func IsNoTransitionAvailableError(err error) bool
```

Checks if an error is a "no transition available" error.

```go
func IsTransitionRejectedError(err error) bool
```

Checks if an error is a "transition rejected by guard" error.

### Error Types

```go
var ErrInvalidTransition = errors.New("invalid transition: from, to, or event cannot be nil")
var ErrInvalidEvent = errors.New("invalid event: event cannot be nil")
```

Errors returned when attempting to add invalid transitions or fire invalid events.

Custom error types:

- `ErrNoTransitionAvailable` - when there's no transition for the current state and event
- `ErrTransitionRejected` - when all transitions are rejected by their guards
