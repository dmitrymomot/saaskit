package queue

import "errors"

var (
	ErrRepositoryNil            = errors.New("repository cannot be nil")
	ErrPayloadNil               = errors.New("payload cannot be nil")
	ErrPayloadMarshal           = errors.New("failed to marshal payload to JSON")
	ErrTaskCreate               = errors.New("failed to create task in storage")
	ErrInvalidPriority          = errors.New("priority must be between 0 and 100")
	ErrNoItemsToEnqueue         = errors.New("no items to enqueue")
	ErrHandlerNotFound          = errors.New("no handler registered for task type")
	ErrNoHandlers               = errors.New("no task handlers registered")
	ErrInvalidSchedule          = errors.New("invalid schedule format")
	ErrTaskAlreadyRegistered    = errors.New("task already registered")
	ErrSchedulerNotConfigured   = errors.New("scheduler has no registered tasks")
	ErrNoScheduleSpecified      = errors.New("no schedule specified for periodic task")
	ErrFailedToGetNextTask      = errors.New("failed to get next task from storage")
	ErrFailedToUpdateTaskStatus = errors.New("failed to update task status")
	ErrFailedToMoveToDLQ        = errors.New("failed to move task to dead letter queue")
	ErrNoTaskToClaim            = errors.New("no task available to claim")
)
