package queue

import (
	"context"
	"encoding/json"
)

type (
	Handler interface {
		Name() string
		Handle(ctx context.Context, payload json.RawMessage) error
	}

	TaskHandlerFunc[T any]  func(ctx context.Context, payload T) error
	PeriodicTaskHandlerFunc func(ctx context.Context) error
)

func NewTaskHandler[T any](handler TaskHandlerFunc[T]) Handler {
	var payload T
	return &oneTimeTaskHandler[T]{
		name:    qualifiedStructName(payload),
		handler: handler,
	}
}

func NewPeriodicTaskHandler(name string, handler PeriodicTaskHandlerFunc) Handler {
	return &periodicTaskHandler{
		name:    name,
		handler: handler,
	}
}

type oneTimeTaskHandler[T any] struct {
	name    string
	handler TaskHandlerFunc[T]
}

func (h *oneTimeTaskHandler[T]) Name() string {
	return h.name
}

func (h *oneTimeTaskHandler[T]) Handle(ctx context.Context, payload json.RawMessage) error {
	var t T
	if err := json.Unmarshal(payload, &t); err != nil {
		return err
	}
	return h.handler(ctx, t)
}

type periodicTaskHandler struct {
	name    string
	handler PeriodicTaskHandlerFunc
}

func (h *periodicTaskHandler) Name() string {
	return h.name
}

func (h *periodicTaskHandler) Handle(ctx context.Context, _ json.RawMessage) error {
	return h.handler(ctx)
}
