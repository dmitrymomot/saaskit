package audit

import (
	"context"
	"fmt"
	"time"
)

// Result represents the outcome of an audited action
type Result string

const (
	ResultSuccess Result = "success"
	ResultFailure Result = "failure"
	ResultError   Result = "error"
)

// Event represents a single audit log entry
type Event struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id"`
	UserID     string         `json:"user_id"`
	SessionID  string         `json:"session_id"`
	Action     string         `json:"action"`
	Resource   string         `json:"resource"`
	ResourceID string         `json:"resource_id"`
	Result     Result         `json:"result"`
	Error      string         `json:"error,omitempty"`
	RequestID  string         `json:"request_id,omitempty"`
	IP         string         `json:"ip,omitempty"`
	UserAgent  string         `json:"user_agent,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Validate checks if the event has all required fields
func (e *Event) Validate() error {
	if e.Action == "" {
		return fmt.Errorf("%w: action is required", ErrEventValidation)
	}
	return nil
}

// Storage defines the interface for persisting audit events
type Storage interface {
	Store(ctx context.Context, events ...Event) error
	Query(ctx context.Context, criteria Criteria) ([]Event, error)
}

// StorageCounter is an optional interface that Storage implementations
// can implement to provide optimized COUNT queries without loading all records
type StorageCounter interface {
	Storage
	Count(ctx context.Context, criteria Criteria) (int64, error)
}

// EventOption applies configuration to an Event during creation.
// Used with Log and LogError methods to add metadata, resources, etc.
type EventOption func(*Event)

// Logger defines the interface for recording audit events
type Logger interface {
	Log(ctx context.Context, action string, opts ...EventOption) error
	LogError(ctx context.Context, action string, err error, opts ...EventOption) error
}

// Reader defines the interface for querying audit events
type Reader interface {
	Find(ctx context.Context, criteria Criteria) ([]Event, error)
	FindWithCursor(ctx context.Context, criteria Criteria, cursor string) ([]Event, string, error)
	Count(ctx context.Context, criteria Criteria) (int64, error)
}

// Criteria defines filtering options for querying audit events
type Criteria struct {
	TenantID   string    `json:"tenant_id,omitempty"`
	UserID     string    `json:"user_id,omitempty"`
	SessionID  string    `json:"session_id,omitempty"`
	Action     string    `json:"action,omitempty"`
	Resource   string    `json:"resource,omitempty"`
	ResourceID string    `json:"resource_id,omitempty"`
	Result     Result    `json:"result,omitempty"`
	StartTime  time.Time `json:"start_time,omitzero"`
	EndTime    time.Time `json:"end_time,omitzero"`
	Limit      int       `json:"limit,omitempty"`
	Offset     int       `json:"offset,omitempty"`
	Cursor     string    `json:"cursor,omitempty"`
}
