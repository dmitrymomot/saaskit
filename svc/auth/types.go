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

// Token subjects used in JWT tokens for various authentication operations.
const (
	SubjectPasswordReset = "password_reset"
	SubjectEmailVerify   = "email_verify" // for future use
	SubjectEmailChange   = "email_change" // for email update verification
	SubjectMagicLink     = "magic_link"
)

// User represents a user account in the authentication system.
type User struct {
	ID         uuid.UUID
	Email      string
	Name       string // Display name (optional)
	Avatar     string // Avatar URL (optional)
	AuthMethod string
	IsVerified bool
	CreatedAt  time.Time
}
