package auth

import "errors"

// General authentication errors
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
)

// Token-related errors
var (
	ErrTokenExpired  = errors.New("token expired")
	ErrTokenInvalid  = errors.New("invalid token")
	ErrTokenNotFound = errors.New("token not found")
)

// Password-specific errors
var (
	ErrWeakPassword     = errors.New("password does not meet security requirements")
	ErrPasswordMismatch = errors.New("passwords do not match")
	ErrPasswordRequired = errors.New("password is required")
)

// OAuth-specific errors
var (
	ErrInvalidState       = errors.New("invalid OAuth state")
	ErrStateNotFound      = errors.New("OAuth state not found or expired")
	ErrInvalidCode        = errors.New("invalid OAuth code")
	ErrProviderLinked     = errors.New("provider already linked to another account")
	ErrNoProviderLink     = errors.New("no provider link found")
	ErrUnverifiedEmail    = errors.New("email not verified by provider")
	ErrNoPrimaryEmail     = errors.New("no primary email from provider")
	ErrProviderEmailInUse = errors.New("email from provider already registered")
)

// Magic link errors
var (
	ErrMagicLinkExpired = errors.New("magic link expired")
	ErrMagicLinkInvalid = errors.New("invalid magic link")
	ErrTokenAlreadyUsed = errors.New("token already used")
)

// User management errors
var (
	ErrEmailUnchanged   = errors.New("email unchanged")
	ErrCannotDeleteUser = errors.New("cannot delete user")
)
