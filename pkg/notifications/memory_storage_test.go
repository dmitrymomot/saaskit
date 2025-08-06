package notifications

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_Get(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*MemoryStorage)
		userID      string
		notifID     string
		want        *Notification
		wantErr     bool
		expectedErr error
	}{
		{
			name: "successful get",
			setup: func(s *MemoryStorage) {
				notif := Notification{
					ID:        "notif-123",
					UserID:    "user-456",
					Title:     "Test Notification",
					Message:   "Test message",
					CreatedAt: time.Now(),
				}
				_ = s.Create(context.Background(), notif)
			},
			userID:  "user-456",
			notifID: "notif-123",
			want: &Notification{
				ID:      "notif-123",
				UserID:  "user-456",
				Title:   "Test Notification",
				Message: "Test message",
			},
			wantErr: false,
		},
		{
			name:        "user not found",
			setup:       func(s *MemoryStorage) {},
			userID:      "nonexistent-user",
			notifID:     "notif-123",
			want:        nil,
			wantErr:     true,
			expectedErr: ErrNotificationNotFound,
		},
		{
			name: "notification not found",
			setup: func(s *MemoryStorage) {
				notif := Notification{
					ID:     "notif-456",
					UserID: "user-123",
					Title:  "Test",
				}
				_ = s.Create(context.Background(), notif)
			},
			userID:      "user-123",
			notifID:     "notif-999",
			want:        nil,
			wantErr:     true,
			expectedErr: ErrNotificationNotFound,
		},
		{
			name: "get from multiple notifications",
			setup: func(s *MemoryStorage) {
				for i := 0; i < 5; i++ {
					notif := Notification{
						ID:      "notif-" + string(rune('0'+i)),
						UserID:  "user-123",
						Title:   "Test " + string(rune('0'+i)),
						Message: "Message " + string(rune('0'+i)),
					}
					_ = s.Create(context.Background(), notif)
				}
			},
			userID:  "user-123",
			notifID: "notif-3",
			want: &Notification{
				ID:      "notif-3",
				UserID:  "user-123",
				Title:   "Test 3",
				Message: "Message 3",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemoryStorage()
			tt.setup(storage)

			got, err := storage.Get(context.Background(), tt.userID, tt.notifID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != nil {
					assert.Equal(t, tt.expectedErr, err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.ID, got.ID)
				assert.Equal(t, tt.want.UserID, got.UserID)
				assert.Equal(t, tt.want.Title, got.Title)
				assert.Equal(t, tt.want.Message, got.Message)
			}
		})
	}
}

func TestMemoryStorage_Delete(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(*MemoryStorage)
		userID       string
		notifIDs     []string
		validateFunc func(*testing.T, *MemoryStorage)
	}{
		{
			name: "delete single notification",
			setup: func(s *MemoryStorage) {
				_ = s.Create(context.Background(), Notification{ID: "notif-1", UserID: "user-123"})
				_ = s.Create(context.Background(), Notification{ID: "notif-2", UserID: "user-123"})
				_ = s.Create(context.Background(), Notification{ID: "notif-3", UserID: "user-123"})
			},
			userID:   "user-123",
			notifIDs: []string{"notif-2"},
			validateFunc: func(t *testing.T, s *MemoryStorage) {
				// notif-2 should be deleted
				_, err := s.Get(context.Background(), "user-123", "notif-2")
				assert.Equal(t, ErrNotificationNotFound, err)

				// Others should still exist
				_, err = s.Get(context.Background(), "user-123", "notif-1")
				assert.NoError(t, err)
				_, err = s.Get(context.Background(), "user-123", "notif-3")
				assert.NoError(t, err)
			},
		},
		{
			name: "delete multiple notifications",
			setup: func(s *MemoryStorage) {
				for i := 1; i <= 5; i++ {
					_ = s.Create(context.Background(), Notification{
						ID:     "notif-" + string(rune('0'+i)),
						UserID: "user-123",
					})
				}
			},
			userID:   "user-123",
			notifIDs: []string{"notif-1", "notif-3", "notif-5"},
			validateFunc: func(t *testing.T, s *MemoryStorage) {
				// Deleted notifications should not exist
				for _, id := range []string{"notif-1", "notif-3", "notif-5"} {
					_, err := s.Get(context.Background(), "user-123", id)
					assert.Equal(t, ErrNotificationNotFound, err)
				}

				// Remaining should exist
				for _, id := range []string{"notif-2", "notif-4"} {
					_, err := s.Get(context.Background(), "user-123", id)
					assert.NoError(t, err)
				}
			},
		},
		{
			name: "delete all notifications",
			setup: func(s *MemoryStorage) {
				_ = s.Create(context.Background(), Notification{ID: "notif-1", UserID: "user-123"})
				_ = s.Create(context.Background(), Notification{ID: "notif-2", UserID: "user-123"})
			},
			userID:   "user-123",
			notifIDs: []string{"notif-1", "notif-2"},
			validateFunc: func(t *testing.T, s *MemoryStorage) {
				list, err := s.List(context.Background(), "user-123", ListOptions{})
				assert.NoError(t, err)
				assert.Empty(t, list)
			},
		},
		{
			name:     "delete from non-existent user",
			setup:    func(s *MemoryStorage) {},
			userID:   "nonexistent",
			notifIDs: []string{"notif-1"},
			validateFunc: func(t *testing.T, s *MemoryStorage) {
				// Should not error for non-existent user
			},
		},
		{
			name: "delete non-existent notification",
			setup: func(s *MemoryStorage) {
				_ = s.Create(context.Background(), Notification{ID: "notif-1", UserID: "user-123"})
			},
			userID:   "user-123",
			notifIDs: []string{"notif-999"},
			validateFunc: func(t *testing.T, s *MemoryStorage) {
				// Existing notification should remain
				_, err := s.Get(context.Background(), "user-123", "notif-1")
				assert.NoError(t, err)
			},
		},
		{
			name: "empty notification IDs",
			setup: func(s *MemoryStorage) {
				_ = s.Create(context.Background(), Notification{ID: "notif-1", UserID: "user-123"})
			},
			userID:   "user-123",
			notifIDs: []string{},
			validateFunc: func(t *testing.T, s *MemoryStorage) {
				// Nothing should be deleted
				_, err := s.Get(context.Background(), "user-123", "notif-1")
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemoryStorage()
			tt.setup(storage)

			err := storage.Delete(context.Background(), tt.userID, tt.notifIDs...)
			assert.NoError(t, err)

			tt.validateFunc(t, storage)
		})
	}
}

func TestMemoryStorage_Create_Validation(t *testing.T) {
	tests := []struct {
		name    string
		notif   Notification
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid notification",
			notif: Notification{
				ID:     "notif-123",
				UserID: "user-456",
				Title:  "Test",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			notif: Notification{
				UserID: "user-456",
				Title:  "Test",
			},
			wantErr: true,
			errMsg:  "notification ID is required",
		},
		{
			name: "missing UserID",
			notif: Notification{
				ID:    "notif-123",
				Title: "Test",
			},
			wantErr: true,
			errMsg:  "user ID is required",
		},
		{
			name: "auto-set CreatedAt",
			notif: Notification{
				ID:     "notif-123",
				UserID: "user-456",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemoryStorage()
			err := storage.Create(context.Background(), tt.notif)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)

				// Verify notification was created
				got, err := storage.Get(context.Background(), tt.notif.UserID, tt.notif.ID)
				assert.NoError(t, err)
				assert.NotNil(t, got)

				// Check CreatedAt was set
				if tt.notif.CreatedAt.IsZero() {
					assert.False(t, got.CreatedAt.IsZero())
				}
			}
		})
	}
}

func TestMemoryStorage_List_Filtering(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	tests := []struct {
		name      string
		setup     func(*MemoryStorage)
		userID    string
		opts      ListOptions
		wantCount int
		validate  func(*testing.T, []Notification)
	}{
		{
			name: "filter by unread only",
			setup: func(s *MemoryStorage) {
				_ = s.Create(context.Background(), Notification{ID: "1", UserID: "user", Read: false})
				_ = s.Create(context.Background(), Notification{ID: "2", UserID: "user", Read: true})
				_ = s.Create(context.Background(), Notification{ID: "3", UserID: "user", Read: false})
			},
			userID:    "user",
			opts:      ListOptions{OnlyUnread: true},
			wantCount: 2,
		},
		{
			name: "filter by types",
			setup: func(s *MemoryStorage) {
				_ = s.Create(context.Background(), Notification{ID: "1", UserID: "user", Type: "info"})
				_ = s.Create(context.Background(), Notification{ID: "2", UserID: "user", Type: "warning"})
				_ = s.Create(context.Background(), Notification{ID: "3", UserID: "user", Type: "error"})
				_ = s.Create(context.Background(), Notification{ID: "4", UserID: "user", Type: "info"})
			},
			userID:    "user",
			opts:      ListOptions{Types: []Type{TypeInfo, TypeError}},
			wantCount: 3,
		},
		{
			name: "filter by since time",
			setup: func(s *MemoryStorage) {
				_ = s.Create(context.Background(), Notification{ID: "1", UserID: "user", CreatedAt: yesterday})
				_ = s.Create(context.Background(), Notification{ID: "2", UserID: "user", CreatedAt: now})
				_ = s.Create(context.Background(), Notification{ID: "3", UserID: "user", CreatedAt: tomorrow})
			},
			userID:    "user",
			opts:      ListOptions{Since: &now},
			wantCount: 2,
		},
		{
			name: "filter expired notifications",
			setup: func(s *MemoryStorage) {
				expired := yesterday
				_ = s.Create(context.Background(), Notification{ID: "1", UserID: "user", ExpiresAt: &expired})
				_ = s.Create(context.Background(), Notification{ID: "2", UserID: "user", ExpiresAt: &tomorrow})
				_ = s.Create(context.Background(), Notification{ID: "3", UserID: "user"}) // No expiry
			},
			userID:    "user",
			opts:      ListOptions{},
			wantCount: 2, // Should exclude expired
		},
		{
			name: "pagination with limit and offset",
			setup: func(s *MemoryStorage) {
				for i := 0; i < 10; i++ {
					_ = s.Create(context.Background(), Notification{
						ID:        "notif-" + string(rune('0'+i)),
						UserID:    "user",
						CreatedAt: now.Add(time.Duration(i) * time.Hour),
					})
				}
			},
			userID:    "user",
			opts:      ListOptions{Limit: 3, Offset: 2},
			wantCount: 3,
		},
		{
			name: "sorting by created time (newest first)",
			setup: func(s *MemoryStorage) {
				_ = s.Create(context.Background(), Notification{ID: "old", UserID: "user", CreatedAt: yesterday})
				_ = s.Create(context.Background(), Notification{ID: "new", UserID: "user", CreatedAt: tomorrow})
				_ = s.Create(context.Background(), Notification{ID: "current", UserID: "user", CreatedAt: now})
			},
			userID:    "user",
			opts:      ListOptions{},
			wantCount: 3,
			validate: func(t *testing.T, notifs []Notification) {
				assert.Equal(t, "new", notifs[0].ID)
				assert.Equal(t, "current", notifs[1].ID)
				assert.Equal(t, "old", notifs[2].ID)
			},
		},
		{
			name:      "empty user notifications",
			setup:     func(s *MemoryStorage) {},
			userID:    "user",
			opts:      ListOptions{},
			wantCount: 0,
		},
		{
			name: "combined filters",
			setup: func(s *MemoryStorage) {
				_ = s.Create(context.Background(), Notification{ID: "1", UserID: "user", Type: "info", Read: false, CreatedAt: yesterday})
				_ = s.Create(context.Background(), Notification{ID: "2", UserID: "user", Type: "error", Read: false, CreatedAt: now})
				_ = s.Create(context.Background(), Notification{ID: "3", UserID: "user", Type: "info", Read: true, CreatedAt: tomorrow})
				_ = s.Create(context.Background(), Notification{ID: "4", UserID: "user", Type: "info", Read: false, CreatedAt: tomorrow})
			},
			userID:    "user",
			opts:      ListOptions{OnlyUnread: true, Types: []Type{TypeInfo}, Since: &now},
			wantCount: 1, // Only ID: 4 matches all criteria
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemoryStorage()
			tt.setup(storage)

			got, err := storage.List(context.Background(), tt.userID, tt.opts)
			assert.NoError(t, err)
			assert.Len(t, got, tt.wantCount)

			if tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestMemoryStorage_ConcurrentAccess(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()
	userID := "user-concurrent"

	// Number of concurrent operations
	numGoroutines := 100
	numOperations := 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent creates
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				notif := Notification{
					ID:      "notif-" + string(rune('0'+idx)) + "-" + string(rune('0'+j)),
					UserID:  userID,
					Title:   "Test",
					Message: "Concurrent test",
				}
				_ = storage.Create(ctx, notif)
			}
		}(i)
	}

	wg.Wait()

	// Verify all notifications were created
	notifications, err := storage.List(ctx, userID, ListOptions{})
	require.NoError(t, err)
	assert.Len(t, notifications, numGoroutines*numOperations)

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				notifID := "notif-" + string(rune('0'+idx)) + "-" + string(rune('0'+j))
				_, _ = storage.Get(ctx, userID, notifID)
			}
		}(i)
	}

	wg.Wait()

	// Concurrent mark as read
	wg.Add(numGoroutines / 2)
	for i := 0; i < numGoroutines/2; i++ {
		go func(idx int) {
			defer wg.Done()
			notifIDs := make([]string, numOperations)
			for j := 0; j < numOperations; j++ {
				notifIDs[j] = "notif-" + string(rune('0'+idx)) + "-" + string(rune('0'+j))
			}
			_ = storage.MarkRead(ctx, userID, notifIDs...)
		}(i)
	}

	wg.Wait()

	// Concurrent deletes
	wg.Add(numGoroutines / 4)
	for i := 0; i < numGoroutines/4; i++ {
		go func(idx int) {
			defer wg.Done()
			notifIDs := make([]string, numOperations/2)
			for j := 0; j < numOperations/2; j++ {
				notifIDs[j] = "notif-" + string(rune('0'+idx)) + "-" + string(rune('0'+j))
			}
			_ = storage.Delete(ctx, userID, notifIDs...)
		}(i)
	}

	wg.Wait()

	// Verify state after concurrent operations
	finalNotifications, err := storage.List(ctx, userID, ListOptions{})
	require.NoError(t, err)

	// Should have some notifications deleted
	assert.Less(t, len(finalNotifications), numGoroutines*numOperations)

	// Count unread should work
	unreadCount, err := storage.CountUnread(ctx, userID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, unreadCount, 0)
}

func TestMemoryStorage_MarkRead(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*MemoryStorage)
		userID   string
		notifIDs []string
		validate func(*testing.T, *MemoryStorage)
	}{
		{
			name: "mark single as read",
			setup: func(s *MemoryStorage) {
				_ = s.Create(context.Background(), Notification{ID: "1", UserID: "user", Read: false})
				_ = s.Create(context.Background(), Notification{ID: "2", UserID: "user", Read: false})
			},
			userID:   "user",
			notifIDs: []string{"1"},
			validate: func(t *testing.T, s *MemoryStorage) {
				n1, _ := s.Get(context.Background(), "user", "1")
				n2, _ := s.Get(context.Background(), "user", "2")
				assert.True(t, n1.Read)
				assert.False(t, n2.Read)
			},
		},
		{
			name: "mark multiple as read",
			setup: func(s *MemoryStorage) {
				for i := 1; i <= 5; i++ {
					_ = s.Create(context.Background(), Notification{
						ID:     "notif-" + string(rune('0'+i)),
						UserID: "user",
						Read:   false,
					})
				}
			},
			userID:   "user",
			notifIDs: []string{"notif-1", "notif-3", "notif-5"},
			validate: func(t *testing.T, s *MemoryStorage) {
				for i := 1; i <= 5; i++ {
					n, _ := s.Get(context.Background(), "user", "notif-"+string(rune('0'+i)))
					if i == 1 || i == 3 || i == 5 {
						assert.True(t, n.Read)
						assert.False(t, n.ReadAt.IsZero())
					} else {
						assert.False(t, n.Read)
					}
				}
			},
		},
		{
			name:     "mark read for non-existent user",
			setup:    func(s *MemoryStorage) {},
			userID:   "nonexistent",
			notifIDs: []string{"1"},
			validate: func(t *testing.T, s *MemoryStorage) {
				// Should not error
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemoryStorage()
			tt.setup(storage)

			err := storage.MarkRead(context.Background(), tt.userID, tt.notifIDs...)
			assert.NoError(t, err)

			tt.validate(t, storage)
		})
	}
}

func TestMemoryStorage_CountUnread(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*MemoryStorage)
		userID string
		want   int
	}{
		{
			name: "count all unread",
			setup: func(s *MemoryStorage) {
				_ = s.Create(context.Background(), Notification{ID: "1", UserID: "user", Read: false})
				_ = s.Create(context.Background(), Notification{ID: "2", UserID: "user", Read: false})
				_ = s.Create(context.Background(), Notification{ID: "3", UserID: "user", Read: true})
			},
			userID: "user",
			want:   2,
		},
		{
			name: "exclude expired notifications",
			setup: func(s *MemoryStorage) {
				expired := time.Now().Add(-1 * time.Hour)
				future := time.Now().Add(1 * time.Hour)
				_ = s.Create(context.Background(), Notification{ID: "1", UserID: "user", Read: false, ExpiresAt: &expired})
				_ = s.Create(context.Background(), Notification{ID: "2", UserID: "user", Read: false, ExpiresAt: &future})
				_ = s.Create(context.Background(), Notification{ID: "3", UserID: "user", Read: false})
			},
			userID: "user",
			want:   2, // Excludes expired
		},
		{
			name:   "non-existent user",
			setup:  func(s *MemoryStorage) {},
			userID: "nonexistent",
			want:   0,
		},
		{
			name: "all read",
			setup: func(s *MemoryStorage) {
				_ = s.Create(context.Background(), Notification{ID: "1", UserID: "user", Read: true})
				_ = s.Create(context.Background(), Notification{ID: "2", UserID: "user", Read: true})
			},
			userID: "user",
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemoryStorage()
			tt.setup(storage)

			got, err := storage.CountUnread(context.Background(), tt.userID)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
