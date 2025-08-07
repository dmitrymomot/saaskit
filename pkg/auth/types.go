package auth

import (
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
