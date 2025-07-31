package statemachine

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidTransition = errors.New("invalid transition: from, to, or event cannot be nil")
	ErrInvalidEvent      = errors.New("invalid event: event cannot be nil")
)

// ErrNoTransitionAvailable indicates no valid transition exists for the given state/event combination.
type ErrNoTransitionAvailable struct {
	StateName string
	EventName string
}

func (e *ErrNoTransitionAvailable) Error() string {
	return fmt.Sprintf("no transition available from state '%s' for event '%s'", e.StateName, e.EventName)
}

func NewErrNoTransitionAvailable(stateName, eventName string) *ErrNoTransitionAvailable {
	return &ErrNoTransitionAvailable{
		StateName: stateName,
		EventName: eventName,
	}
}

// ErrTransitionRejected indicates all possible transitions were blocked by guard functions.
type ErrTransitionRejected struct {
	StateName string
	EventName string
}

func (e *ErrTransitionRejected) Error() string {
	return fmt.Sprintf("transition from state '%s' for event '%s' was rejected by guards", e.StateName, e.EventName)
}

func NewErrTransitionRejected(stateName, eventName string) *ErrTransitionRejected {
	return &ErrTransitionRejected{
		StateName: stateName,
		EventName: eventName,
	}
}

func IsNoTransitionAvailableError(err error) bool {
	var e *ErrNoTransitionAvailable
	return errors.As(err, &e)
}

func IsTransitionRejectedError(err error) bool {
	var e *ErrTransitionRejected
	return errors.As(err, &e)
}
