package statemachine

import (
	"context"
	"fmt"
	"sync"
)

// SimpleStateMachine provides a thread-safe in-memory state machine implementation.
// Uses a nested map structure for O(1) transition lookups: [fromState][event][]Transition
type SimpleStateMachine struct {
	initialState State
	currentState State
	transitions  map[string]map[string][]Transition
	mu           sync.RWMutex
}

func newSimpleStateMachine(initialState State) *SimpleStateMachine {
	sm := &SimpleStateMachine{
		initialState: initialState,
		currentState: initialState,
		transitions:  make(map[string]map[string][]Transition),
	}
	return sm
}

func (sm *SimpleStateMachine) Current() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

func (sm *SimpleStateMachine) AddTransition(from, to State, event Event, guards []Guard, actions []Action) error {
	if from == nil || to == nil || event == nil {
		return ErrInvalidTransition
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	fromStateName := from.Name()
	eventName := event.Name()

	if _, ok := sm.transitions[fromStateName]; !ok {
		sm.transitions[fromStateName] = make(map[string][]Transition)
	}

	transition := Transition{
		From:    from,
		To:      to,
		Event:   event,
		Guards:  guards,
		Actions: actions,
	}

	// Multiple transitions allowed for same from/event to support guard-based branching
	sm.transitions[fromStateName][eventName] = append(sm.transitions[fromStateName][eventName], transition)
	return nil
}

func (sm *SimpleStateMachine) Fire(ctx context.Context, event Event, data any) error {
	if event == nil {
		return ErrInvalidEvent
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	currentStateName := sm.currentState.Name()
	eventName := event.Name()

	if _, ok := sm.transitions[currentStateName]; !ok {
		return NewErrNoTransitionAvailable(currentStateName, eventName)
	}

	transitions, ok := sm.transitions[currentStateName][eventName]
	if !ok || len(transitions) == 0 {
		return NewErrNoTransitionAvailable(currentStateName, eventName)
	}

	// First transition with passing guards wins (enables priority ordering)
	var validTransition *Transition
	for i, t := range transitions {
		allGuardsPassed := true
		for _, guard := range t.Guards {
			if guard != nil && !guard(ctx, sm.currentState, event, data) {
				allGuardsPassed = false
				break
			}
		}
		if allGuardsPassed {
			validTransition = &transitions[i]
			break
		}
	}

	if validTransition == nil {
		return NewErrTransitionRejected(currentStateName, eventName)
	}

	// Execute actions before state change; any failure aborts transition
	for _, action := range validTransition.Actions {
		if action != nil {
			if err := action(ctx, sm.currentState, validTransition.To, event, data); err != nil {
				return fmt.Errorf("action failed: %w", err)
			}
		}
	}

	sm.currentState = validTransition.To
	return nil
}

func (sm *SimpleStateMachine) CanFire(ctx context.Context, event Event, data any) bool {
	if event == nil {
		return false
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	currentStateName := sm.currentState.Name()
	eventName := event.Name()

	if _, ok := sm.transitions[currentStateName]; !ok {
		return false
	}

	transitions, ok := sm.transitions[currentStateName][eventName]
	if !ok || len(transitions) == 0 {
		return false
	}

	// Return true if any transition's guards would allow it
	for _, t := range transitions {
		allGuardsPassed := true
		for _, guard := range t.Guards {
			if guard != nil && !guard(ctx, sm.currentState, event, data) {
				allGuardsPassed = false
				break
			}
		}
		if allGuardsPassed {
			return true
		}
	}

	return false
}

func (sm *SimpleStateMachine) Reset() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.currentState = sm.initialState
	return nil
}
