package statemachine

// Builder provides a fluent API for building state machines.
type Builder struct {
	machine      *SimpleStateMachine
	currentFrom  State
	currentEvent Event
	currentTo    State
	guards       []Guard
	actions      []Action
}

// NewBuilder creates a new state machine builder.
func NewBuilder(initialState State) *Builder {
	return &Builder{
		machine: NewSimpleStateMachine(initialState),
	}
}

// From sets the starting state for a transition.
func (b *Builder) From(state State) *Builder {
	b.reset()
	b.currentFrom = state
	return b
}

// When sets the event that triggers a transition.
func (b *Builder) When(event Event) *Builder {
	b.currentEvent = event
	return b
}

// To sets the target state for a transition.
func (b *Builder) To(state State) *Builder {
	b.currentTo = state
	return b
}

// WithGuard adds a guard function to the current transition.
func (b *Builder) WithGuard(guard Guard) *Builder {
	b.guards = append(b.guards, guard)
	return b
}

// WithAction adds an action function to the current transition.
func (b *Builder) WithAction(action Action) *Builder {
	b.actions = append(b.actions, action)
	return b
}

// Add finalizes the current transition and adds it to the state machine.
func (b *Builder) Add() (*Builder, error) {
	if err := b.machine.AddTransition(b.currentFrom, b.currentTo, b.currentEvent, b.guards, b.actions); err != nil {
		return b, err
	}
	b.reset()
	return b, nil
}

// WithTransition is a shorthand method to add a transition in one call.
func (b *Builder) WithTransition(from, to State, event Event, guard Guard, action Action) (*Builder, error) {
	var guards []Guard
	var actions []Action

	if guard != nil {
		guards = append(guards, guard)
	}

	if action != nil {
		actions = append(actions, action)
	}

	if err := b.machine.AddTransition(from, to, event, guards, actions); err != nil {
		return b, err
	}

	return b, nil
}

// Build returns the constructed state machine.
func (b *Builder) Build() StateMachine {
	return b.machine
}

// reset clears the current transition configuration.
func (b *Builder) reset() {
	b.currentFrom = nil
	b.currentEvent = nil
	b.currentTo = nil
	b.guards = nil
	b.actions = nil
}
