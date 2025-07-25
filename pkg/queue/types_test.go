package queue_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/queue"
)

func TestPriority_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		priority queue.Priority
		valid    bool
	}{
		{"min priority", queue.PriorityMin, true},
		{"low priority", queue.PriorityLow, true},
		{"medium priority", queue.PriorityMedium, true},
		{"high priority", queue.PriorityHigh, true},
		{"max priority", queue.PriorityMax, true},
		{"custom valid priority", queue.Priority(37), true},
		{"boundary min", queue.Priority(0), true},
		{"boundary max", queue.Priority(100), true},
		{"below min", queue.Priority(-1), false},
		{"above max", queue.Priority(101), false},
		{"way below min", queue.Priority(-100), false},
		{"way above max", queue.Priority(127), false}, // int8 max is 127
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.valid, tt.priority.Valid())
		})
	}
}

func TestPriority_Constants(t *testing.T) {
	t.Parallel()

	// Verify constant values
	assert.Equal(t, queue.Priority(0), queue.PriorityMin)
	assert.Equal(t, queue.Priority(25), queue.PriorityLow)
	assert.Equal(t, queue.Priority(50), queue.PriorityMedium)
	assert.Equal(t, queue.Priority(75), queue.PriorityHigh)
	assert.Equal(t, queue.Priority(100), queue.PriorityMax)
	assert.Equal(t, queue.PriorityMedium, queue.PriorityDefault)

	// Verify ordering
	assert.Less(t, int(queue.PriorityMin), int(queue.PriorityLow))
	assert.Less(t, int(queue.PriorityLow), int(queue.PriorityMedium))
	assert.Less(t, int(queue.PriorityMedium), int(queue.PriorityHigh))
	assert.Less(t, int(queue.PriorityHigh), int(queue.PriorityMax))
}

func TestTaskType_Constants(t *testing.T) {
	t.Parallel()

	// Verify task type values
	assert.Equal(t, queue.TaskType("one-time"), queue.TaskTypeOneTime)
	assert.Equal(t, queue.TaskType("periodic"), queue.TaskTypePeriodic)
}

func TestTaskStatus_Constants(t *testing.T) {
	t.Parallel()

	// Verify task status values
	assert.Equal(t, queue.TaskStatus("pending"), queue.TaskStatusPending)
	assert.Equal(t, queue.TaskStatus("processing"), queue.TaskStatusProcessing)
	assert.Equal(t, queue.TaskStatus("completed"), queue.TaskStatusCompleted)
	assert.Equal(t, queue.TaskStatus("failed"), queue.TaskStatusFailed)
}

func TestDefaultQueueName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "default", queue.DefaultQueueName)
}
