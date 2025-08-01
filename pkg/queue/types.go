package queue

import (
	"time"

	"github.com/google/uuid"
)

// DefaultQueueName is the default queue name used when no queue is specified
const DefaultQueueName = "default"

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeOneTime  TaskType = "one-time"
	TaskTypePeriodic TaskType = "periodic"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
)

// Priority represents task priority (0-100, higher is more important)
// Using int8 provides sufficient range while keeping memory footprint minimal
type Priority int8

// Priority constants
const (
	PriorityMin     Priority = 0
	PriorityLow     Priority = 25
	PriorityMedium  Priority = 50
	PriorityHigh    Priority = 75
	PriorityMax     Priority = 100
	PriorityDefault Priority = PriorityMedium
)

// Valid checks if the priority is within valid range
func (p Priority) Valid() bool {
	return p >= PriorityMin && p <= PriorityMax
}

// Task represents a task in the queue
type Task struct {
	ID          uuid.UUID  `json:"id"`
	Queue       string     `json:"queue"`
	TaskType    TaskType   `json:"task_type"`
	TaskName    string     `json:"task_name"`
	Payload     []byte     `json:"payload,omitempty"`
	Status      TaskStatus `json:"status"`
	Priority    Priority   `json:"priority"`
	RetryCount  int8       `json:"retry_count"`
	MaxRetries  int8       `json:"max_retries"`
	ScheduledAt time.Time  `json:"scheduled_at"`
	LockedUntil *time.Time `json:"locked_until,omitempty"`
	LockedBy    *uuid.UUID `json:"locked_by,omitempty"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	Error       *string    `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// TasksDlq represents a task in the dead letter queue
// Stores failed tasks that exhausted all retries for manual inspection and recovery
type TasksDlq struct {
	ID         uuid.UUID `json:"id"`
	TaskID     uuid.UUID `json:"task_id"`
	Queue      string    `json:"queue"`
	TaskType   TaskType  `json:"task_type"`
	TaskName   string    `json:"task_name"`
	Payload    []byte    `json:"payload,omitempty"`
	Priority   Priority  `json:"priority"`
	Error      string    `json:"error"`
	RetryCount int8      `json:"retry_count"`
	FailedAt   time.Time `json:"failed_at"`
	CreatedAt  time.Time `json:"created_at"`
}
