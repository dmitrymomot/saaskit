package statemachine

import (
	"fmt"
)

// Option configures a state machine during construction.
type Option func(*SimpleStateMachine) error

// TransitionOption configures a single transition with guards and actions.
type TransitionOption func(*transitionConfig)

// TransitionDef defines a transition between states.
type TransitionDef struct {
	From    State
	To      State
	Event   Event
	Guards  []Guard
	Actions []Action
}

type transitionConfig struct {
	guards  []Guard
	actions []Action
}

// New creates a new state machine with the given initial state and options.
func New(initialState State, opts ...Option) (StateMachine, error) {
	if initialState == nil {
		return nil, fmt.Errorf("initial state cannot be nil")
	}

	sm := newSimpleStateMachine(initialState)

	for _, opt := range opts {
		if err := opt(sm); err != nil {
			return nil, err
		}
	}

	return sm, nil
}

// MustNew creates a new state machine with the given initial state and options.
// Panics if any option fails to apply, following SaasKit's fail-fast pattern.
func MustNew(initialState State, opts ...Option) StateMachine {
	sm, err := New(initialState, opts...)
	if err != nil {
		panic(fmt.Sprintf("failed to create state machine: %v", err))
	}
	return sm
}

// WithTransition adds a single transition to the state machine.
func WithTransition(from, to State, event Event, opts ...TransitionOption) Option {
	return func(sm *SimpleStateMachine) error {
		cfg := &transitionConfig{}
		for _, opt := range opts {
			opt(cfg)
		}

		return sm.AddTransition(from, to, event, cfg.guards, cfg.actions)
	}
}

// WithTransitions adds multiple transitions to the state machine at once.
func WithTransitions(transitions []TransitionDef) Option {
	return func(sm *SimpleStateMachine) error {
		for i, t := range transitions {
			if err := sm.AddTransition(t.From, t.To, t.Event, t.Guards, t.Actions); err != nil {
				// Handle nil states/events safely in error message
				fromName := "<nil>"
				toName := "<nil>"
				eventName := "<nil>"

				if t.From != nil {
					fromName = t.From.Name()
				}
				if t.To != nil {
					toName = t.To.Name()
				}
				if t.Event != nil {
					eventName = t.Event.Name()
				}

				return fmt.Errorf("failed to add transition[%d] %s->%s on %s: %w",
					i, fromName, toName, eventName, err)
			}
		}
		return nil
	}
}

// WithGuard adds a single guard to a transition.
func WithGuard(guard Guard) TransitionOption {
	return func(cfg *transitionConfig) {
		if guard != nil {
			cfg.guards = append(cfg.guards, guard)
		}
	}
}

// WithGuards adds multiple guards to a transition.
func WithGuards(guards ...Guard) TransitionOption {
	return func(cfg *transitionConfig) {
		for _, guard := range guards {
			if guard != nil {
				cfg.guards = append(cfg.guards, guard)
			}
		}
	}
}

// WithAction adds a single action to a transition.
func WithAction(action Action) TransitionOption {
	return func(cfg *transitionConfig) {
		if action != nil {
			cfg.actions = append(cfg.actions, action)
		}
	}
}

// WithActions adds multiple actions to a transition.
func WithActions(actions ...Action) TransitionOption {
	return func(cfg *transitionConfig) {
		for _, action := range actions {
			if action != nil {
				cfg.actions = append(cfg.actions, action)
			}
		}
	}
}
