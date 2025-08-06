package notifications

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockStorage for testing Manager
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Create(ctx context.Context, notif Notification) error {
	args := m.Called(ctx, notif)
	return args.Error(0)
}

func (m *MockStorage) Get(ctx context.Context, userID, notifID string) (*Notification, error) {
	args := m.Called(ctx, userID, notifID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Notification), args.Error(1)
}

func (m *MockStorage) List(ctx context.Context, userID string, opts ListOptions) ([]Notification, error) {
	args := m.Called(ctx, userID, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Notification), args.Error(1)
}

func (m *MockStorage) MarkRead(ctx context.Context, userID string, notifIDs ...string) error {
	args := m.Called(ctx, userID, notifIDs)
	return args.Error(0)
}

func (m *MockStorage) Delete(ctx context.Context, userID string, notifIDs ...string) error {
	args := m.Called(ctx, userID, notifIDs)
	return args.Error(0)
}

func (m *MockStorage) CountUnread(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

// MockDeliverer for testing Manager
type MockDeliverer struct {
	mock.Mock
}

func (m *MockDeliverer) Deliver(ctx context.Context, notif Notification) error {
	args := m.Called(ctx, notif)
	return args.Error(0)
}

func (m *MockDeliverer) DeliverBatch(ctx context.Context, notifications []Notification) error {
	args := m.Called(ctx, notifications)
	return args.Error(0)
}

func TestManager_Get(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		notifID   string
		setupMock func(*MockStorage)
		want      *Notification
		wantErr   bool
	}{
		{
			name:    "successful get",
			userID:  "user-123",
			notifID: "notif-456",
			setupMock: func(ms *MockStorage) {
				expected := &Notification{
					ID:      "notif-456",
					UserID:  "user-123",
					Title:   "Test Notification",
					Message: "Test message",
				}
				ms.On("Get", mock.Anything, "user-123", "notif-456").Return(expected, nil)
			},
			want: &Notification{
				ID:      "notif-456",
				UserID:  "user-123",
				Title:   "Test Notification",
				Message: "Test message",
			},
			wantErr: false,
		},
		{
			name:    "notification not found",
			userID:  "user-123",
			notifID: "nonexistent",
			setupMock: func(ms *MockStorage) {
				ms.On("Get", mock.Anything, "user-123", "nonexistent").Return(nil, errors.New("not found"))
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "storage error",
			userID:  "user-123",
			notifID: "notif-456",
			setupMock: func(ms *MockStorage) {
				ms.On("Get", mock.Anything, "user-123", "notif-456").Return(nil, errors.New("storage error"))
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(MockStorage)
			tt.setupMock(mockStorage)

			manager := NewManager(mockStorage, nil)
			got, err := manager.Get(context.Background(), tt.userID, tt.notifID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			mockStorage.AssertExpectations(t)
		})
	}
}

func TestManager_Delete(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		notifIDs  []string
		setupMock func(*MockStorage)
		wantErr   bool
	}{
		{
			name:     "successful delete single",
			userID:   "user-123",
			notifIDs: []string{"notif-456"},
			setupMock: func(ms *MockStorage) {
				ms.On("Delete", mock.Anything, "user-123", []string{"notif-456"}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "successful delete multiple",
			userID:   "user-123",
			notifIDs: []string{"notif-456", "notif-789", "notif-101"},
			setupMock: func(ms *MockStorage) {
				ms.On("Delete", mock.Anything, "user-123", []string{"notif-456", "notif-789", "notif-101"}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "delete with empty IDs",
			userID:   "user-123",
			notifIDs: []string{},
			setupMock: func(ms *MockStorage) {
				ms.On("Delete", mock.Anything, "user-123", []string{}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "storage error",
			userID:   "user-123",
			notifIDs: []string{"notif-456"},
			setupMock: func(ms *MockStorage) {
				ms.On("Delete", mock.Anything, "user-123", []string{"notif-456"}).Return(errors.New("storage error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(MockStorage)
			tt.setupMock(mockStorage)

			manager := NewManager(mockStorage, nil)
			err := manager.Delete(context.Background(), tt.userID, tt.notifIDs...)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockStorage.AssertExpectations(t)
		})
	}
}

func TestManager_SendBatch(t *testing.T) {
	tests := []struct {
		name          string
		notifications []Notification
		setupMocks    func(*MockStorage, *MockDeliverer)
		wantErr       bool
		validateCalls func(*testing.T, *MockStorage, *MockDeliverer)
	}{
		{
			name: "successful batch send with IDs",
			notifications: []Notification{
				{ID: "notif-1", UserID: "user-1", Title: "Title 1", CreatedAt: time.Now()},
				{ID: "notif-2", UserID: "user-2", Title: "Title 2", CreatedAt: time.Now()},
			},
			setupMocks: func(ms *MockStorage, md *MockDeliverer) {
				ms.On("Create", mock.Anything, mock.AnythingOfType("Notification")).Return(nil).Times(2)
				md.On("DeliverBatch", mock.Anything, mock.AnythingOfType("[]notifications.Notification")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "successful batch send without IDs",
			notifications: []Notification{
				{UserID: "user-1", Title: "Title 1"},
				{UserID: "user-2", Title: "Title 2"},
			},
			setupMocks: func(ms *MockStorage, md *MockDeliverer) {
				ms.On("Create", mock.Anything, mock.AnythingOfType("Notification")).Return(nil).Times(2)
				md.On("DeliverBatch", mock.Anything, mock.AnythingOfType("[]notifications.Notification")).Return(nil)
			},
			wantErr: false,
			validateCalls: func(t *testing.T, ms *MockStorage, md *MockDeliverer) {
				// Verify that IDs were generated
				calls := ms.Calls
				for _, call := range calls {
					if call.Method == "Create" {
						notif := call.Arguments[1].(Notification)
						assert.NotEmpty(t, notif.ID, "ID should be generated")
						assert.False(t, notif.CreatedAt.IsZero(), "CreatedAt should be set")
					}
				}
			},
		},
		{
			name: "storage error on first notification",
			notifications: []Notification{
				{UserID: "user-1", Title: "Title 1"},
				{UserID: "user-2", Title: "Title 2"},
			},
			setupMocks: func(ms *MockStorage, md *MockDeliverer) {
				ms.On("Create", mock.Anything, mock.AnythingOfType("Notification")).Return(errors.New("storage error")).Once()
			},
			wantErr: true,
		},
		{
			name: "storage error on second notification",
			notifications: []Notification{
				{UserID: "user-1", Title: "Title 1"},
				{UserID: "user-2", Title: "Title 2"},
			},
			setupMocks: func(ms *MockStorage, md *MockDeliverer) {
				ms.On("Create", mock.Anything, mock.AnythingOfType("Notification")).Return(nil).Once()
				ms.On("Create", mock.Anything, mock.AnythingOfType("Notification")).Return(errors.New("storage error")).Once()
			},
			wantErr: true,
		},
		{
			name: "delivery error doesn't fail operation",
			notifications: []Notification{
				{UserID: "user-1", Title: "Title 1"},
				{UserID: "user-2", Title: "Title 2"},
			},
			setupMocks: func(ms *MockStorage, md *MockDeliverer) {
				ms.On("Create", mock.Anything, mock.AnythingOfType("Notification")).Return(nil).Times(2)
				md.On("DeliverBatch", mock.Anything, mock.AnythingOfType("[]notifications.Notification")).Return(errors.New("delivery error"))
			},
			wantErr: false, // Should not fail even if delivery fails
		},
		{
			name:          "empty batch",
			notifications: []Notification{},
			setupMocks: func(ms *MockStorage, md *MockDeliverer) {
				// DeliverBatch is still called even for empty batch
				md.On("DeliverBatch", mock.Anything, mock.AnythingOfType("[]notifications.Notification")).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(MockStorage)
			mockDeliverer := new(MockDeliverer)
			tt.setupMocks(mockStorage, mockDeliverer)

			manager := NewManager(mockStorage, mockDeliverer)
			err := manager.SendBatch(context.Background(), tt.notifications)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.validateCalls != nil {
				tt.validateCalls(t, mockStorage, mockDeliverer)
			}

			mockStorage.AssertExpectations(t)
			mockDeliverer.AssertExpectations(t)
		})
	}
}

func TestManager_MarkAllRead(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		setupMock func(*MockStorage)
		wantErr   bool
	}{
		{
			name:   "successful mark all read with notifications",
			userID: "user-123",
			setupMock: func(ms *MockStorage) {
				notifications := []Notification{
					{ID: "notif-1", UserID: "user-123", Read: false},
					{ID: "notif-2", UserID: "user-123", Read: false},
					{ID: "notif-3", UserID: "user-123", Read: false},
				}
				ms.On("List", mock.Anything, "user-123", ListOptions{OnlyUnread: true}).Return(notifications, nil)
				ms.On("MarkRead", mock.Anything, "user-123", []string{"notif-1", "notif-2", "notif-3"}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "no unread notifications",
			userID: "user-123",
			setupMock: func(ms *MockStorage) {
				ms.On("List", mock.Anything, "user-123", ListOptions{OnlyUnread: true}).Return([]Notification{}, nil)
				// MarkRead should not be called when there are no notifications
			},
			wantErr: false,
		},
		{
			name:   "list error",
			userID: "user-123",
			setupMock: func(ms *MockStorage) {
				ms.On("List", mock.Anything, "user-123", ListOptions{OnlyUnread: true}).Return(nil, errors.New("list error"))
			},
			wantErr: true,
		},
		{
			name:   "mark read error",
			userID: "user-123",
			setupMock: func(ms *MockStorage) {
				notifications := []Notification{
					{ID: "notif-1", UserID: "user-123", Read: false},
				}
				ms.On("List", mock.Anything, "user-123", ListOptions{OnlyUnread: true}).Return(notifications, nil)
				ms.On("MarkRead", mock.Anything, "user-123", []string{"notif-1"}).Return(errors.New("mark read error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(MockStorage)
			tt.setupMock(mockStorage)

			manager := NewManager(mockStorage, nil)
			err := manager.MarkAllRead(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockStorage.AssertExpectations(t)
		})
	}
}

func TestManager_SendToUsers(t *testing.T) {
	tests := []struct {
		name      string
		userIDs   []string
		template  Notification
		setupMock func(*MockStorage, *MockDeliverer)
		wantErr   bool
	}{
		{
			name:    "successful send to multiple users",
			userIDs: []string{"user-1", "user-2", "user-3"},
			template: Notification{
				Title:   "Broadcast Message",
				Message: "Important update",
				Type:    "info",
			},
			setupMock: func(ms *MockStorage, md *MockDeliverer) {
				ms.On("Create", mock.Anything, mock.AnythingOfType("Notification")).Return(nil).Times(3)
				md.On("DeliverBatch", mock.Anything, mock.AnythingOfType("[]notifications.Notification")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "storage error on second user",
			userIDs: []string{"user-1", "user-2", "user-3"},
			template: Notification{
				Title: "Broadcast Message",
			},
			setupMock: func(ms *MockStorage, md *MockDeliverer) {
				ms.On("Create", mock.Anything, mock.AnythingOfType("Notification")).Return(nil).Once()
				ms.On("Create", mock.Anything, mock.AnythingOfType("Notification")).Return(errors.New("storage error")).Once()
			},
			wantErr: true,
		},
		{
			name:    "delivery error doesn't fail operation",
			userIDs: []string{"user-1", "user-2"},
			template: Notification{
				Title: "Broadcast Message",
			},
			setupMock: func(ms *MockStorage, md *MockDeliverer) {
				ms.On("Create", mock.Anything, mock.AnythingOfType("Notification")).Return(nil).Times(2)
				md.On("DeliverBatch", mock.Anything, mock.AnythingOfType("[]notifications.Notification")).Return(errors.New("delivery error"))
			},
			wantErr: false,
		},
		{
			name:     "empty user list",
			userIDs:  []string{},
			template: Notification{Title: "Test"},
			setupMock: func(ms *MockStorage, md *MockDeliverer) {
				// DeliverBatch is still called even for empty user list
				md.On("DeliverBatch", mock.Anything, mock.AnythingOfType("[]notifications.Notification")).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(MockStorage)
			mockDeliverer := new(MockDeliverer)
			tt.setupMock(mockStorage, mockDeliverer)

			manager := NewManager(mockStorage, mockDeliverer)
			err := manager.SendToUsers(context.Background(), tt.userIDs, tt.template)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockStorage.AssertExpectations(t)
			mockDeliverer.AssertExpectations(t)
		})
	}
}

func TestManager_Send_WithNilDeliverer(t *testing.T) {
	mockStorage := new(MockStorage)
	mockStorage.On("Create", mock.Anything, mock.AnythingOfType("Notification")).Return(nil)

	// Create manager with nil deliverer
	manager := NewManager(mockStorage, nil)

	notif := Notification{
		UserID:  "user-123",
		Title:   "Test",
		Message: "Test message",
	}

	err := manager.Send(context.Background(), notif)
	require.NoError(t, err)

	mockStorage.AssertExpectations(t)
}

func TestManager_CountUnread(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		setupMock func(*MockStorage)
		want      int
		wantErr   bool
	}{
		{
			name:   "successful count",
			userID: "user-123",
			setupMock: func(ms *MockStorage) {
				ms.On("CountUnread", mock.Anything, "user-123").Return(5, nil)
			},
			want:    5,
			wantErr: false,
		},
		{
			name:   "zero unread",
			userID: "user-123",
			setupMock: func(ms *MockStorage) {
				ms.On("CountUnread", mock.Anything, "user-123").Return(0, nil)
			},
			want:    0,
			wantErr: false,
		},
		{
			name:   "storage error",
			userID: "user-123",
			setupMock: func(ms *MockStorage) {
				ms.On("CountUnread", mock.Anything, "user-123").Return(0, errors.New("storage error"))
			},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(MockStorage)
			tt.setupMock(mockStorage)

			manager := NewManager(mockStorage, nil)
			got, err := manager.CountUnread(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			mockStorage.AssertExpectations(t)
		})
	}
}
