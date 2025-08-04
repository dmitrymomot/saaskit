package audit

import (
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

// Event represents a single audit log entry with comprehensive context.
// All fields except Action are optional, though TenantID/UserID are typically required for SaaS apps.
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

// Validate ensures the event meets minimum requirements for storage.
// Only Action is required - other validations are application-specific.
func (e *Event) Validate() error {
	if e.Action == "" {
		return fmt.Errorf("%w: action is required", ErrEventValidation)
	}
	return nil
}

// EventOption applies configuration to an Event during creation.
// Used with Log and LogError methods to add metadata, resources, etc.
type EventOption func(*Event)
