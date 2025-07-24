package queue

import "errors"

// Common errors
var (
	// ErrRepositoryNil is returned when a nil repository is provided
	ErrRepositoryNil = errors.New("repository cannot be nil")

	// ErrPayloadNil is returned when attempting to enqueue a nil payload
	ErrPayloadNil = errors.New("payload cannot be nil")

	// ErrPayloadMarshal is returned when payload marshaling fails
	ErrPayloadMarshal = errors.New("failed to marshal payload to JSON")

	// ErrTaskCreate is returned when task creation in storage fails
	ErrTaskCreate = errors.New("failed to create task in storage")

	// ErrInvalidPriority is returned when priority is outside valid range
	ErrInvalidPriority = errors.New("priority must be between 0 and 100")

	// ErrNoItemsToEnqueue is returned when batch enqueue is called with empty items
	ErrNoItemsToEnqueue = errors.New("no items to enqueue")

	// ErrHandlerNotFound is returned when no handler is registered for a task
	ErrHandlerNotFound = errors.New("no handler registered for task type")

	// ErrNoHandlers is returned when worker has no handlers registered
	ErrNoHandlers = errors.New("no task handlers registered")

	// ErrInvalidSchedule is returned when schedule format is invalid
	ErrInvalidSchedule = errors.New("invalid schedule format")

	// ErrTaskAlreadyRegistered is returned when trying to register a duplicate task
	ErrTaskAlreadyRegistered = errors.New("task already registered")

	// ErrSchedulerNotConfigured is returned when scheduler has no tasks
	ErrSchedulerNotConfigured = errors.New("scheduler has no registered tasks")

	// ErrNoScheduleSpecified is returned when no schedule is provided for periodic task
	ErrNoScheduleSpecified = errors.New("no schedule specified for periodic task")

	// ErrFailedToGetNextTask is returned when fetching next task fails
	ErrFailedToGetNextTask = errors.New("failed to get next task from storage")

	// ErrFailedToUpdateTaskStatus is returned when task status update fails
	ErrFailedToUpdateTaskStatus = errors.New("failed to update task status")

	// ErrFailedToMoveToDLQ is returned when moving task to DLQ fails
	ErrFailedToMoveToDLQ = errors.New("failed to move task to dead letter queue")
)
