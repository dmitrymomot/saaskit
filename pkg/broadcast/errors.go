package broadcast

import (
	"fmt"
	"time"
)

// ErrHubClosed is returned when operations are attempted on a closed hub
type ErrHubClosed struct{}

func (e ErrHubClosed) Error() string {
	return "broadcast: hub is closed"
}

// ErrSubscriberClosed is returned when operations are attempted on a closed subscriber
type ErrSubscriberClosed struct {
	ID string
}

func (e ErrSubscriberClosed) Error() string {
	return fmt.Sprintf("broadcast: subscriber %s is closed", e.ID)
}

// ErrChannelNotFound is returned when a channel doesn't exist
type ErrChannelNotFound struct {
	Channel string
}

func (e ErrChannelNotFound) Error() string {
	return fmt.Sprintf("broadcast: channel %s not found", e.Channel)
}

// ErrStorageFailure wraps storage errors
type ErrStorageFailure struct {
	Operation string
	Err       error
}

func (e ErrStorageFailure) Error() string {
	return fmt.Sprintf("broadcast: storage %s failed: %v", e.Operation, e.Err)
}

func (e ErrStorageFailure) Unwrap() error {
	return e.Err
}

// ErrPublishTimeout is returned when publishing times out
type ErrPublishTimeout struct {
	Channel string
	Timeout time.Duration
}

func (e ErrPublishTimeout) Error() string {
	return fmt.Sprintf("broadcast: publish to channel %s timed out after %v", e.Channel, e.Timeout)
}

// ErrShutdownTimeout is returned when hub shutdown times out
type ErrShutdownTimeout struct{}

func (e ErrShutdownTimeout) Error() string {
	return "broadcast: shutdown timeout exceeded"
}
