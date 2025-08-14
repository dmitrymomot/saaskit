# auth

Complete authentication solution for SaaS applications with OAuth, magic links, password authentication, and user management.

## Features

- OAuth authentication (Google, GitHub) with profile data (name, avatar) and extensible adapter pattern
- Magic link passwordless authentication with replay protection
- Password authentication with bcrypt hashing and strength validation
- User management with email changes, password updates, and profile management
- Context functions for middleware integration and user access
- Extensible hook system for custom business logic
- Type-safe interfaces with comprehensive error handling

## Installation

```bash
go get github.com/dmitrymomot/saaskit/svc/auth
```

## Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/dmitrymomot/saaskit/svc/auth"
    "github.com/google/uuid"
)

func main() {
    ctx := context.Background()

    // Initialize your storage implementation
    storage := &MyAuthStorage{} // implements required interfaces

    // Password authentication
    passwordAuth := auth.NewPasswordService(storage, "your-jwt-secret")

    // Magic link authentication
    magicLinkAuth := auth.NewMagicLinkService(storage, "your-jwt-secret")

    // OAuth with Google (automatically populates name and avatar)
    googleAuth := auth.NewOAuthService(
        storage,
        auth.NewGoogleAdapter(auth.GoogleOAuthConfig{
            ClientID:     "your-google-client-id",
            ClientSecret: "your-google-client-secret",
            RedirectURL:  "https://yourapp.com/auth/google/callback",
        }),
    )

    // User management
    userManager := auth.NewUserService(storage, "your-jwt-secret")

    // Register new user with password
    user, err := passwordAuth.Register(ctx, "user@example.com", "securePassword123")
    if err != nil {
        log.Fatal(err)
    }

    // Store user in context for middleware chain
    ctx = auth.SetUserToContext(ctx, user)

    // Retrieve user from context later
    currentUser := auth.GetUserFromContext(ctx)
    if currentUser != nil {
        log.Printf("User: %s (%s)", currentUser.Email, currentUser.Name)
    }
}
```

## Common Operations

### Password Authentication

```go
// Register new user
user, err := passwordAuth.Register(ctx, "user@example.com", "password123")

// Authenticate user
user, err := passwordAuth.Authenticate(ctx, "user@example.com", "password123")

// Password reset flow
resetReq, err := passwordAuth.ForgotPassword(ctx, "user@example.com")
user, err := passwordAuth.ResetPassword(ctx, resetReq.Token, "newPassword123")
```

### Magic Link Authentication

```go
// Request magic link
linkReq, err := magicLinkAuth.RequestMagicLink(ctx, "user@example.com")

// Verify magic link token
user, err := magicLinkAuth.VerifyMagicLink(ctx, linkReq.Token)
```

### OAuth Authentication

```go
// Get authorization URL
authURL, err := oauthAuth.GetAuthURL(ctx)

// Handle OAuth callback
user, err := oauthAuth.Auth(ctx, code, state, nil)

// Link OAuth account to existing user
user, err := oauthAuth.Auth(ctx, code, state, &existingUserID)

// Unlink OAuth account
err := oauthAuth.Unlink(ctx, userID)
```

### User Management

```go
// Get user by ID
user, err := userManager.GetUser(ctx, userID)

// Change password
err := userManager.ChangePassword(ctx, userID, "oldPassword", "newPassword")

// Request email change
emailReq, err := userManager.RequestEmailChange(ctx, userID, "new@example.com", "currentPassword")

// Confirm email change
user, err := userManager.ConfirmEmailChange(ctx, emailReq.Token)
```

### Context Functions

```go
// Store authenticated user in request context (typically in middleware)
ctx = auth.SetUserToContext(ctx, user)

// Retrieve user from context in handlers
user := auth.GetUserFromContext(ctx)
if user != nil {
    // User is authenticated
    fmt.Printf("Hello %s!", user.Name)
    fmt.Printf("Avatar: %s", user.Avatar)
}
```

## Error Handling

```go
if errors.Is(err, auth.ErrUserNotFound) {
    // Handle user not found
}

if errors.Is(err, auth.ErrInvalidCredentials) {
    // Handle authentication failure
}

if errors.Is(err, auth.ErrTokenExpired) {
    // Handle expired token
}

if errors.Is(err, auth.ErrEmailAlreadyExists) {
    // Handle duplicate email
}
```

## Configuration

### Password Service Options

```go
passwordAuth := auth.NewPasswordService(
    storage,
    tokenSecret,
    auth.WithBcryptCost(12),
    auth.WithResetTokenTTL(2*time.Hour),
    auth.WithPasswordStrength(validator.PasswordStrengthConfig{
        MinLength:      10,
        MaxLength:      128,
        MinCharClasses: 3,
    }),
    auth.WithAfterRegister(func(ctx context.Context, user *auth.User) error {
        // Send welcome email
        return nil
    }),
)
```

### OAuth Service Options

```go
oauthAuth := auth.NewOAuthService(
    storage,
    adapter,
    auth.WithStateTTL(15*time.Minute),
    auth.WithVerifiedOnly(true),
    auth.WithAfterAuth(func(ctx context.Context, user *auth.User) error {
        // Track OAuth registration
        return nil
    }),
)
```

### Magic Link Service Options

```go
magicLinkAuth := auth.NewMagicLinkService(
    storage,
    tokenSecret,
    auth.WithMagicLinkTTL(10*time.Minute),
    auth.WithAfterGenerate(func(ctx context.Context, user *auth.User, token string) error {
        // Send magic link email
        return nil
    }),
)
```

## Storage Interface Implementation

You must implement the required storage interfaces. The User struct now includes profile fields:

```go
type AuthStorage struct {
    // Your database/storage implementation
}

// User struct with profile data
// type User struct {
//     ID         uuid.UUID
//     Email      string
//     Name       string // Display name from OAuth or manual update
//     Avatar     string // Avatar URL from OAuth or manual update
//     AuthMethod string
//     IsVerified bool
//     CreatedAt  time.Time
// }

// Implement PasswordStorage interface
func (s *AuthStorage) CreateUser(ctx context.Context, user *auth.User) error {
    // Store user in database with profile fields (Name, Avatar)
}

func (s *AuthStorage) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
    // Fetch user by email, include profile fields
}

// Implement UserStorage interface (extended with profile management)
func (s *AuthStorage) UpdateUserProfile(ctx context.Context, id uuid.UUID, user *auth.User) error {
    // Update user's Name and Avatar fields
}

// Implement other required methods...
```

## API Documentation

For detailed API documentation:

```bash
go doc -all ./svc/auth
```

Or visit [pkg.go.dev](https://pkg.go.dev/github.com/dmitrymomot/saaskit/svc/auth) for online documentation.

## Notes

- All services use bcrypt for password hashing with configurable cost
- JWT tokens are used for password reset, email change, and magic link flows
- OAuth adapters handle provider-specific implementations and automatically populate user profile data (name, avatar) from providers
- OAuth services create users with populated Name and Avatar fields from provider profiles
- Context functions enable clean middleware integration for authenticated user access
- Storage operations should be atomic where indicated to prevent race conditions
- Hook functions run synchronously but do not block authentication on errors
