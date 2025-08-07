package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	MethodPassword    = "password"
	MethodMagicLink   = "magic_link"
	MethodOAuthGoogle = "oauth_google"
	MethodOAuthGithub = "oauth_github"
)

// User represents an authenticated user in the system
type User struct {
	ID         uuid.UUID
	Email      string
	AuthMethod string
	IsVerified bool
	CreatedAt  time.Time
}

// OAuthStorage defines storage operations for OAuth authentication (provider-agnostic)
type OAuthStorage interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	StoreOAuthLink(ctx context.Context, userID uuid.UUID, provider, providerUserID string) error
	GetUserByOAuth(ctx context.Context, provider, providerUserID string) (*User, error)
	RemoveOAuthLink(ctx context.Context, userID uuid.UUID, provider string) error
	StoreState(ctx context.Context, state string, expiresAt time.Time) error
	ConsumeState(ctx context.Context, state string) error
}
