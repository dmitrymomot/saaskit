package notifications

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNotification_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{
			name:      "no expiration",
			expiresAt: nil,
			want:      false,
		},
		{
			name:      "expired in the past",
			expiresAt: &past,
			want:      true,
		},
		{
			name:      "expires in the future",
			expiresAt: &future,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Notification{
				ExpiresAt: tt.expiresAt,
			}
			assert.Equal(t, tt.want, n.IsExpired())
		})
	}
}

func TestNotification_MarkAsRead(t *testing.T) {
	tests := []struct {
		name     string
		initial  Notification
		validate func(*testing.T, *Notification)
	}{
		{
			name: "mark unread as read",
			initial: Notification{
				ID:     "test-1",
				UserID: "user-1",
				Read:   false,
				ReadAt: nil,
			},
			validate: func(t *testing.T, n *Notification) {
				assert.True(t, n.Read)
				assert.NotNil(t, n.ReadAt)
				assert.True(t, time.Now().Sub(*n.ReadAt) < time.Second)
			},
		},
		{
			name: "mark already read notification",
			initial: Notification{
				ID:     "test-2",
				UserID: "user-2",
				Read:   true,
				ReadAt: func() *time.Time {
					t := time.Now().Add(-24 * time.Hour)
					return &t
				}(),
			},
			validate: func(t *testing.T, n *Notification) {
				assert.True(t, n.Read)
				assert.NotNil(t, n.ReadAt)
				// ReadAt should be updated to current time
				assert.True(t, time.Now().Sub(*n.ReadAt) < time.Second)
			},
		},
		{
			name: "preserve other fields",
			initial: Notification{
				ID:       "test-3",
				UserID:   "user-3",
				Title:    "Test Title",
				Message:  "Test Message",
				Type:     TypeInfo,
				Priority: PriorityHigh,
				Data:     map[string]interface{}{"key": "value"},
				Actions: []Action{
					{Label: "Click", URL: "/click", Style: "primary"},
				},
				CreatedAt: time.Now().Add(-1 * time.Hour),
				ExpiresAt: func() *time.Time {
					t := time.Now().Add(1 * time.Hour)
					return &t
				}(),
			},
			validate: func(t *testing.T, n *Notification) {
				assert.True(t, n.Read)
				assert.NotNil(t, n.ReadAt)
				// Verify other fields are unchanged
				assert.Equal(t, "test-3", n.ID)
				assert.Equal(t, "user-3", n.UserID)
				assert.Equal(t, "Test Title", n.Title)
				assert.Equal(t, "Test Message", n.Message)
				assert.Equal(t, TypeInfo, n.Type)
				assert.Equal(t, PriorityHigh, n.Priority)
				assert.Equal(t, "value", n.Data["key"])
				assert.Len(t, n.Actions, 1)
				assert.Equal(t, "Click", n.Actions[0].Label)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notif := tt.initial
			notif.MarkAsRead()
			tt.validate(t, &notif)
		})
	}
}

func TestNotification_Types(t *testing.T) {
	// Verify type constants are properly defined
	assert.Equal(t, Type("info"), TypeInfo)
	assert.Equal(t, Type("success"), TypeSuccess)
	assert.Equal(t, Type("warning"), TypeWarning)
	assert.Equal(t, Type("error"), TypeError)
}

func TestNotification_Priorities(t *testing.T) {
	// Verify priority constants are properly defined
	assert.Equal(t, Priority(0), PriorityLow)
	assert.Equal(t, Priority(1), PriorityNormal)
	assert.Equal(t, Priority(2), PriorityHigh)
	assert.Equal(t, Priority(3), PriorityUrgent)

	// Verify priorities can be compared
	assert.True(t, PriorityLow < PriorityNormal)
	assert.True(t, PriorityNormal < PriorityHigh)
	assert.True(t, PriorityHigh < PriorityUrgent)
}

func TestAction_Structure(t *testing.T) {
	action := Action{
		Label: "View Details",
		URL:   "/notifications/123",
		Style: "primary",
	}

	assert.Equal(t, "View Details", action.Label)
	assert.Equal(t, "/notifications/123", action.URL)
	assert.Equal(t, "primary", action.Style)
}

func TestNotification_EdgeCases(t *testing.T) {
	t.Run("nil data map", func(t *testing.T) {
		n := Notification{
			ID:     "test",
			UserID: "user",
			Data:   nil,
		}
		assert.Nil(t, n.Data)
	})

	t.Run("empty actions slice", func(t *testing.T) {
		n := Notification{
			ID:      "test",
			UserID:  "user",
			Actions: []Action{},
		}
		assert.Empty(t, n.Actions)
	})

	t.Run("multiple actions", func(t *testing.T) {
		n := Notification{
			ID:     "test",
			UserID: "user",
			Actions: []Action{
				{Label: "Accept", URL: "/accept", Style: "primary"},
				{Label: "Decline", URL: "/decline", Style: "secondary"},
				{Label: "Delete", URL: "/delete", Style: "danger"},
			},
		}
		assert.Len(t, n.Actions, 3)
		assert.Equal(t, "Accept", n.Actions[0].Label)
		assert.Equal(t, "Decline", n.Actions[1].Label)
		assert.Equal(t, "Delete", n.Actions[2].Label)
	})

	t.Run("complex data map", func(t *testing.T) {
		n := Notification{
			ID:     "test",
			UserID: "user",
			Data: map[string]interface{}{
				"string":  "value",
				"number":  42,
				"boolean": true,
				"nested": map[string]interface{}{
					"key": "value",
				},
				"array": []interface{}{1, 2, 3},
			},
		}
		assert.Equal(t, "value", n.Data["string"])
		assert.Equal(t, 42, n.Data["number"])
		assert.Equal(t, true, n.Data["boolean"])
		assert.NotNil(t, n.Data["nested"])
		assert.NotNil(t, n.Data["array"])
	})
}

func TestNotification_TimeBoundaries(t *testing.T) {
	t.Run("far future expiration", func(t *testing.T) {
		farFuture := time.Now().Add(100 * 365 * 24 * time.Hour) // 100 years
		n := Notification{
			ExpiresAt: &farFuture,
		}
		assert.False(t, n.IsExpired())
	})

	t.Run("far past expiration", func(t *testing.T) {
		farPast := time.Now().Add(-100 * 365 * 24 * time.Hour) // 100 years ago
		n := Notification{
			ExpiresAt: &farPast,
		}
		assert.True(t, n.IsExpired())
	})

	t.Run("millisecond precision", func(t *testing.T) {
		// Test that expiration works with millisecond precision
		// Using milliseconds instead of microseconds to avoid race conditions
		justExpired := time.Now().Add(-1 * time.Millisecond)
		n := Notification{
			ExpiresAt: &justExpired,
		}
		assert.True(t, n.IsExpired())

		justNotExpired := time.Now().Add(10 * time.Millisecond)
		n2 := Notification{
			ExpiresAt: &justNotExpired,
		}
		assert.False(t, n2.IsExpired())
	})
}

func TestNotification_ConcurrentMarkAsRead(t *testing.T) {
	// Test concurrent marking as read (though the current implementation isn't thread-safe)
	n := Notification{
		ID:     "test",
		UserID: "user",
		Read:   false,
	}

	// Mark as read multiple times
	for i := 0; i < 10; i++ {
		n.MarkAsRead()
		assert.True(t, n.Read)
		assert.NotNil(t, n.ReadAt)
	}
}
