package statemachine

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidTransition is returned when attempting to add an invalid transition.
	ErrInvalidTransition = errors.New("invalid transition: from, to, or event cannot be nil")

	// ErrInvalidEvent is returned when attempting to fire a nil event.
	ErrInvalidEvent = errors.New("invalid event: event cannot be nil")
)

// ErrNoTransitionAvailable is returned when there's no transition available for the current state and event.
type ErrNoTransitionAvailable struct {
	StateName string
	EventName string
}

// Error implements the error interface.
func (e *ErrNoTransitionAvailable) Error() string {
	return fmt.Sprintf("no transition available from state '%s' for event '%s'", e.StateName, e.EventName)
}

// NewErrNoTransitionAvailable creates a new ErrNoTransitionAvailable error.
func NewErrNoTransitionAvailable(stateName, eventName string) *ErrNoTransitionAvailable {
	return &ErrNoTransitionAvailable{
		StateName: stateName,
		EventName: eventName,
	}
}

// ErrTransitionRejected is returned when all transitions are rejected by their guards.
type ErrTransitionRejected struct {
	StateName string
	EventName string
}

// Error implements the error interface.
func (e *ErrTransitionRejected) Error() string {
	return fmt.Sprintf("transition from state '%s' for event '%s' was rejected by guards", e.StateName, e.EventName)
}

// NewErrTransitionRejected creates a new ErrTransitionRejected error.
func NewErrTransitionRejected(stateName, eventName string) *ErrTransitionRejected {
	return &ErrTransitionRejected{
		StateName: stateName,
		EventName: eventName,
	}
}

// IsNoTransitionAvailableError checks if the error is an ErrNoTransitionAvailable.
func IsNoTransitionAvailableError(err error) bool {
	var e *ErrNoTransitionAvailable
	return errors.As(err, &e)
}

// IsTransitionRejectedError checks if the error is an ErrTransitionRejected.
func IsTransitionRejectedError(err error) bool {
	var e *ErrTransitionRejected
	return errors.As(err, &e)
}
