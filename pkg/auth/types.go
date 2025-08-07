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

// User represents an authenticated user
type User struct {
	ID         uuid.UUID
	Email      string
	AuthMethod string // "password", "google", "magic_link", etc.
	IsVerified bool
	CreatedAt  time.Time
}
