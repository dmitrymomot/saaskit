// Package statemachine provides a flexible, type-safe implementation of the
// finite-state-machine (FSM) pattern for Go applications.
//
// The package revolves around two minimal interfaces – State and Event – that
// give you full freedom to model domain specific states and events while the
// library handles:
//  1. Transition validation and lookup
//  2. Optional Guard evaluation to accept or reject transitions
//  3. Execution of side-effect Actions during transitions
//  4. Concurrency-safe access to current state and transition map
//
// Ready-made helpers such as StringState and StringEvent let you get started
// quickly for simple scenarios, while custom struct types can satisfy the
// interfaces when additional data is required.
//
// # Architecture
//
// The SimpleStateMachine implementation uses an in-memory nested map structure
// map[FromState][Event][]Transition for O(1) lookups and guards all access with
// a RWMutex. Configuration uses the functional options pattern for a clean,
// idiomatic Go API.
//
// Rich error types with helper predicates (e.g. IsNoTransitionAvailableError)
// allow callers to differentiate between "transition not defined" and
// "guard rejected" cases.
//
// # Usage
//
// Basic example using the options pattern:
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
//	machine := statemachine.MustNew(Draft,
//	    statemachine.WithTransition(Draft, InReview, Submit),
//	)
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
// SimpleStateMachine uses RWMutex for thread safety, making read operations
// (Current, CanFire) cheap while serializing mutations (AddTransition, Fire, Reset).
//
// # See Also
//
// The README in this directory contains more elaborate examples including
// custom state/event structs and advanced builder usage.
package statemachine
