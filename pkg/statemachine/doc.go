// Package statemachine provides a flexible, type-safe implementation of the
// finite-state-machine (FSM) pattern for Go applications.
//
// The package revolves around two minimal interfaces – State and Event – that
// give you full freedom to model domain specific states and events while the
// library takes care of:
//  1. Transition validation and lookup.
//  2. Optional Guard evaluation to accept or reject transitions.
//  3. Execution of side-effect Actions during a transition.
//  4. Concurrency-safe access to the current state and transition map.
//
// Ready-made helpers such as StringState and StringEvent let you get started
// quickly for simple scenarios, while your own struct types can satisfy the
// interfaces when additional data is required.
//
// # Architecture
//
// Internally the package offers a SimpleStateMachine implementation that keeps
// an in-memory map[FromState][Event][]Transition and guards all access with a
// RWMutex. A fluent Builder type is provided to construct that map in a clear
// declarative style.
//
// Errors that can occur during FSM use are exposed via rich error types and
// helper predicates (e.g. IsNoTransitionAvailableError) so that callers can
// differentiate between "transition not defined" and "guard rejected" cases.
//
// # Usage
//
// Basic example using the builder pattern:
//
//	import (
//	    "context"
//	    "github.com/dmitrymomot/saaskit/pkg/statemachine"
//	)
//
//	const (
//	    Draft    = statemachine.StringState("draft")
//	    InReview = statemachine.StringState("in_review")
//	    Submit   = statemachine.StringEvent("submit")
//	)
//
//	machine := statemachine.NewBuilder(Draft).
//	    From(Draft).When(Submit).To(InReview).Add().
//	    Build()
//
//	_ = machine.Fire(context.Background(), Submit, nil)
//
// # Guards and Actions
//
// Guards let you veto a transition based on runtime data:
//
//	isOwner := func(ctx context.Context, from statemachine.State, evt statemachine.Event, data any) bool {
//	    u, ok := data.(map[string]any)
//	    return ok && u["role"] == "owner"
//	}
//
// Actions are executed after all guards succeed and before the state is
// updated:
//
//	logAction := func(ctx context.Context, from, to statemachine.State, evt statemachine.Event, data any) error {
//	    log.Printf("%s -> %s via %s", from.Name(), to.Name(), evt.Name())
//	    return nil
//	}
//
// # Error Handling
//
// When Fire returns an error you can inspect it using helper functions:
//
//	if statemachine.IsNoTransitionAvailableError(err) { /* ... */ }
//	if statemachine.IsTransitionRejectedError(err)   { /* ... */ }
//
// # Concurrency
//
// SimpleStateMachine employs a RWMutex, making read operations (Current,
// CanFire) cheap while serialising mutating operations (AddTransition, Fire,
// Reset).
//
// # See Also
//
// The README in this directory contains more elaborate examples including
// custom state/event structs and advanced builder usage.
package statemachine
