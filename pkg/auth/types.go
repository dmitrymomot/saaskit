package auth

import (
	"time"

	"github.com/google/uuid"
)

// Authentication method identifiers used to track how users authenticate.
const (
	MethodPassword    = "password"
	MethodMagicLink   = "magic_link"
	MethodOAuthGoogle = "oauth_google"
	MethodOAuthGithub = "oauth_github"
)

// User represents a user account in the authentication system.
type User struct {
	ID         uuid.UUID
	Email      string
	AuthMethod string
	IsVerified bool
	CreatedAt  time.Time
}
