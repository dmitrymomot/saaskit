package auth

import (
	"time"

	"github.com/google/uuid"
)

// Predefined auth methods
const (
	MethodPassword    = "password"
	MethodMagicLink   = "magic_link"
	MethodOAuthGoogle = "oauth_google"
	MethodOAuthGithub = "oauth_github"
)

// Identity represents an authenticated user identity
type Identity struct {
	ID         uuid.UUID
	Email      string
	AuthMethod string // "password", "google", "magic_link", etc.
	IsVerified bool
	CreatedAt  time.Time
}
