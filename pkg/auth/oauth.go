package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// OAuthAuthenticator defines the interface for OAuth-based authentication.
type OAuthAuthenticator interface {
	// GetAuthURL generates an OAuth authorization URL with CSRF protection
	GetAuthURL(ctx context.Context) (string, error)

	// Auth handles OAuth callback - authenticates user or links to existing user
	Auth(ctx context.Context, code, state string, linkToUserID *uuid.UUID) (*User, error)

	// Unlink removes the OAuth provider link from a user account
	Unlink(ctx context.Context, userID uuid.UUID) error
}

// OAuthStorage defines the storage interface required by OAuth services.
type OAuthStorage interface {
	// User operations
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error

	// OAuth link operations
	StoreOAuthLink(ctx context.Context, userID uuid.UUID, provider, providerUserID string) error
	GetUserByOAuth(ctx context.Context, provider, providerUserID string) (*User, error)
	RemoveOAuthLink(ctx context.Context, userID uuid.UUID, provider string) error

	// State management for CSRF protection
	StoreState(ctx context.Context, state string, expiresAt time.Time) error
	// ConsumeState atomically checks if state exists and removes it.
	// Returns ErrStateNotFound if state doesn't exist or was already consumed.
	// Must be atomic to prevent race conditions with concurrent requests.
	ConsumeState(ctx context.Context, state string) error
}

// OAuthState represents OAuth state information used for CSRF protection.
type OAuthState struct {
	State       string
	RedirectURL string
	ExpiresAt   time.Time
}

// OAuthLink represents the connection between a local user and an OAuth provider account.
type OAuthLink struct {
	UserID         uuid.UUID
	Provider       string
	ProviderUserID string
	CreatedAt      time.Time
}
