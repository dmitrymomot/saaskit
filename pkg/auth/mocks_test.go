package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockMagicLinkStorage is a mock implementation of MagicLinkStorage.
type MockMagicLinkStorage struct {
	mock.Mock
}

func (m *MockMagicLinkStorage) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockMagicLinkStorage) CreateUser(ctx context.Context, user *User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockMagicLinkStorage) UpdateUserVerified(ctx context.Context, id uuid.UUID, verified bool) error {
	args := m.Called(ctx, id, verified)
	return args.Error(0)
}

func (m *MockMagicLinkStorage) ConsumeToken(ctx context.Context, tokenID string, ttl time.Duration) error {
	args := m.Called(ctx, tokenID, ttl)
	return args.Error(0)
}

// MockPasswordStorage is a mock implementation of PasswordStorage.
type MockPasswordStorage struct {
	mock.Mock
}

func (m *MockPasswordStorage) CreateUser(ctx context.Context, user *User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockPasswordStorage) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockPasswordStorage) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockPasswordStorage) DeleteUser(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPasswordStorage) StorePasswordHash(ctx context.Context, userID uuid.UUID, hash []byte) error {
	args := m.Called(ctx, userID, hash)
	return args.Error(0)
}

func (m *MockPasswordStorage) GetPasswordHash(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// MockUserStorage is a mock implementation of UserStorage.
type MockUserStorage struct {
	mock.Mock
}

func (m *MockUserStorage) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockUserStorage) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockUserStorage) UpdateUserEmail(ctx context.Context, id uuid.UUID, email string) error {
	args := m.Called(ctx, id, email)
	return args.Error(0)
}

func (m *MockUserStorage) GetPasswordHash(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockUserStorage) UpdatePasswordHash(ctx context.Context, userID uuid.UUID, hash []byte) error {
	args := m.Called(ctx, userID, hash)
	return args.Error(0)
}

// MockOAuthStorage is a mock implementation of OAuthStorage.
type MockOAuthStorage struct {
	mock.Mock
}

func (m *MockOAuthStorage) CreateUser(ctx context.Context, user *User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockOAuthStorage) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockOAuthStorage) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockOAuthStorage) DeleteUser(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOAuthStorage) StoreOAuthLink(ctx context.Context, userID uuid.UUID, provider, providerUserID string) error {
	args := m.Called(ctx, userID, provider, providerUserID)
	return args.Error(0)
}

func (m *MockOAuthStorage) GetUserByOAuth(ctx context.Context, provider, providerUserID string) (*User, error) {
	args := m.Called(ctx, provider, providerUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockOAuthStorage) RemoveOAuthLink(ctx context.Context, userID uuid.UUID, provider string) error {
	args := m.Called(ctx, userID, provider)
	return args.Error(0)
}

func (m *MockOAuthStorage) StoreState(ctx context.Context, state string, expiresAt time.Time) error {
	args := m.Called(ctx, state, expiresAt)
	return args.Error(0)
}

func (m *MockOAuthStorage) ConsumeState(ctx context.Context, state string) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

// MockProviderAdapter is a mock implementation of ProviderAdapter.
type MockProviderAdapter struct {
	mock.Mock
}

func (m *MockProviderAdapter) ProviderID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockProviderAdapter) AuthURL(state string) (string, error) {
	args := m.Called(state)
	return args.String(0), args.Error(1)
}

func (m *MockProviderAdapter) ResolveProfile(ctx context.Context, code string) (ProviderProfile, error) {
	args := m.Called(ctx, code)
	return args.Get(0).(ProviderProfile), args.Error(1)
}
