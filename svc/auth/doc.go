// Package auth provides a comprehensive, extensible authentication system for SaaS applications
// with support for multiple authentication methods, secure token handling, and flexible provider integration.
//
// The package is designed around the saaskit framework principles of type safety, explicit configuration,
// and modular design. All services implement well-defined interfaces for easy testing and extension,
// with built-in hooks for customization at key points in the authentication flow.
//
// # Supported Authentication Methods
//
// The package supports four primary authentication methods with rich user profile data:
//
//   - Password-based authentication with bcrypt hashing and configurable strength requirements
//   - Magic link authentication via email for passwordless login
//   - OAuth integration with GitHub and Google providers (extensible to other providers)
//   - User management operations including password changes, email updates, and profile management
//
// # Architecture Overview
//
// The auth package follows a service-oriented architecture with clear separation of concerns:
//
//   - Service interfaces define contracts for each authentication method
//   - Storage interfaces abstract persistence layer implementation
//   - Provider adapters encapsulate OAuth provider-specific logic
//   - Configuration options enable customization without breaking changes
//   - Hooks allow extension points for business logic integration
//   - Context functions provide middleware access to authenticated users
//
// # Password Authentication
//
// Password authentication provides secure user registration and login with industry-standard practices:
//
//	storage := &MyPasswordStorage{} // implement PasswordStorage interface
//	tokenSecret := "your-jwt-secret"
//
//	passwordAuth := auth.NewPasswordService(storage, tokenSecret,
//		auth.WithBcryptCost(12), // Higher cost for better security
//		auth.WithPasswordStrength(validator.PasswordStrengthConfig{
//			MinLength: 10,
//			MinCharClasses: 3,
//		}),
//		auth.WithAfterRegister(func(ctx context.Context, user *auth.User) error {
//			// Send welcome email, create user profile, etc.
//			return nil
//		}),
//	)
//
//	// Register new user
//	user, err := passwordAuth.Register(ctx, "user@example.com", "securepassword123")
//	if err != nil {
//		// Handle registration errors (duplicate email, weak password, etc.)
//	}
//	// User contains: ID, Email, Name (empty for password auth), Avatar (empty), AuthMethod, etc.
//
//	// Authenticate existing user
//	user, err = passwordAuth.Authenticate(ctx, "user@example.com", "securepassword123")
//	if err != nil {
//		// Handle authentication errors (invalid credentials, etc.)
//	}
//
// # Magic Link Authentication
//
// Magic link authentication enables passwordless login through secure email tokens:
//
//	storage := &MyMagicLinkStorage{} // implement MagicLinkStorage interface
//	tokenSecret := "your-jwt-secret"
//
//	magicAuth := auth.NewMagicLinkService(storage, tokenSecret,
//		auth.WithMagicLinkTTL(15*time.Minute), // Short TTL for security
//		auth.WithAfterGenerate(func(ctx context.Context, user *auth.User, token string) error {
//			// Send magic link email to user
//			return emailService.SendMagicLink(user.Email, token)
//		}),
//	)
//
//	// Generate magic link
//	request, err := magicAuth.RequestMagicLink(ctx, "user@example.com")
//	if err != nil {
//		// Handle request errors
//	}
//
//	// Verify magic link token
//	user, err := magicAuth.VerifyMagicLink(ctx, request.Token)
//	if err != nil {
//		// Handle verification errors (expired, invalid, already used)
//	}
//
// # OAuth Authentication
//
// OAuth authentication supports GitHub and Google providers with an extensible adapter pattern:
//
//	// Configure GitHub OAuth
//	githubConfig := auth.GitHubOAuthConfig{
//		ClientID:     "your-github-client-id",
//		ClientSecret: "your-github-client-secret",
//		RedirectURL:  "https://yourapp.com/auth/github/callback",
//		Scopes:       []string{"user:email"},
//		VerifiedOnly: true, // Only accept verified emails
//	}
//
//	storage := &MyOAuthStorage{} // implement OAuthStorage interface
//	githubAdapter := auth.NewGitHubAdapter(githubConfig)
//
//	oauthService := auth.NewOAuthService(storage, githubAdapter,
//		auth.WithStateTTL(10*time.Minute),
//		auth.WithVerifiedOnly(true),
//		auth.WithAfterAuth(func(ctx context.Context, user *auth.User) error {
//			// Handle new OAuth user registration
//			return nil
//		}),
//	)
//
//	// Generate OAuth authorization URL
//	authURL, err := oauthService.GetAuthURL(ctx)
//	if err != nil {
//		// Handle URL generation error
//	}
//	// Redirect user to authURL
//
//	// Handle OAuth callback
//	user, err := oauthService.Auth(ctx, code, state, nil) // nil = new user flow
//	if err != nil {
//		// Handle OAuth errors (invalid state, unverified email, etc.)
//	}
//	// User contains profile data: ID, Email, Name (from provider), Avatar (from provider), etc.
//
//	// Link OAuth account to existing user
//	user, err = oauthService.Auth(ctx, code, state, &existingUserID)
//	if err != nil {
//		// Handle linking errors (already linked, etc.)
//	}
//
// # User Management
//
// User management provides secure operations for account maintenance:
//
//	storage := &MyUserStorage{} // implement UserStorage interface
//	tokenSecret := "your-jwt-secret"
//
//	userManager := auth.NewUserService(storage, tokenSecret,
//		auth.WithBeforeUpdate(func(ctx context.Context, userID uuid.UUID) error {
//			// Validate user permissions, rate limiting, etc.
//			return nil
//		}),
//		auth.WithAfterUpdate(func(ctx context.Context, user *auth.User) error {
//			// Invalidate sessions, send notification, etc.
//			return nil
//		}),
//	)
//
//	// Change password
//	err := userManager.ChangePassword(ctx, userID, "oldpassword", "newpassword")
//	if err != nil {
//		// Handle password change errors
//	}
//
//	// Request email change
//	request, err := userManager.RequestEmailChange(ctx, userID, "newemail@example.com", "currentpassword")
//	if err != nil {
//		// Handle email change request errors
//	}
//
//	// Confirm email change
//	user, err := userManager.ConfirmEmailChange(ctx, request.Token)
//	if err != nil {
//		// Handle confirmation errors
//	}
//
// # Context Functions
//
// The package provides context functions for storing and retrieving authenticated users
// in middleware chains, enabling clean separation between authentication and authorization:
//
//	// In authentication middleware
//	user, err := authService.Authenticate(ctx, token)
//	if err != nil {
//		return err
//	}
//	// Store authenticated user in context
//	ctx = auth.SetUserToContext(ctx, user)
//
//	// In downstream handlers
//	user := auth.GetUserFromContext(ctx)
//	if user == nil {
//		// User not authenticated
//		return errors.New("authentication required")
//	}
//	// Access user data: user.ID, user.Email, user.Name, user.Avatar, etc.
//
// # Profile Management
//
// OAuth providers automatically populate Name and Avatar fields from user profiles.
// The UserStorage interface includes UpdateUserProfile for custom profile updates:
//
//	storage := &MyUserStorage{} // implements UserStorage with UpdateUserProfile
//
//	// Update user profile data
//	updatedUser := &auth.User{
//		Name:   "Updated Name",
//		Avatar: "https://example.com/new-avatar.jpg",
//	}
//	err := storage.UpdateUserProfile(ctx, userID, updatedUser)
//	if err != nil {
//		// Handle profile update errors
//	}
//
// # Error Handling
//
// The package defines specific error types for different failure scenarios, enabling precise
// error handling and appropriate user feedback:
//
//	user, err := passwordAuth.Authenticate(ctx, email, password)
//	if err != nil {
//		switch {
//		case errors.Is(err, auth.ErrInvalidCredentials):
//			// Show generic "invalid email or password" message
//		case errors.Is(err, auth.ErrUserNotFound):
//			// User doesn't exist, might suggest registration
//		case errors.Is(err, auth.ErrTokenExpired):
//			// Token-based operation failed, request new token
//		default:
//			// Internal server error, log and show generic error
//		}
//	}
//
// # Security Features
//
// The package implements several security best practices:
//
//   - Password hashing with bcrypt and configurable cost factors
//   - CSRF protection for OAuth flows using cryptographically secure state tokens
//   - JWT tokens with expiration and replay protection for magic links
//   - Email normalization to prevent duplicate accounts
//   - Timing attack prevention in authentication flows
//   - Secure token generation using crypto/rand
//   - Input validation and sanitization
//
// # Storage Interface Implementation
//
// Applications must implement the storage interfaces for their chosen persistence layer.
// Each authentication method requires its specific storage interface. Note that the User
// struct now includes Name and Avatar fields that should be persisted:
//
//	// Example implementation for password authentication
//	type PostgresPasswordStorage struct {
//		db *sql.DB
//	}
//
//	func (s *PostgresPasswordStorage) CreateUser(ctx context.Context, user *auth.User) error {
//		// Insert user record with profile data into database
//		// INSERT INTO users (id, email, name, avatar, auth_method, is_verified, created_at)
//		// VALUES ($1, $2, $3, $4, $5, $6, $7)
//		return nil
//	}
//
//	func (s *PostgresPasswordStorage) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
//		// Query user by email, include all fields
//		// SELECT id, email, name, avatar, auth_method, is_verified, created_at FROM users WHERE email = $1
//		return nil, nil
//	}
//
//	func (s *PostgresPasswordStorage) UpdateUserProfile(ctx context.Context, id uuid.UUID, user *auth.User) error {
//		// Update user profile fields
//		// UPDATE users SET name = $1, avatar = $2 WHERE id = $3
//		return nil
//	}
//
//	// Implement remaining PasswordStorage methods...
//
// # Provider Extension
//
// New OAuth providers can be added by implementing the ProviderAdapter interface:
//
//	type CustomOAuthAdapter struct {
//		config oauth2.Config
//	}
//
//	func (a *CustomOAuthAdapter) ProviderID() string {
//		return "custom-provider"
//	}
//
//	func (a *CustomOAuthAdapter) AuthURL(state string) (string, error) {
//		return a.config.AuthCodeURL(state), nil
//	}
//
//	func (a *CustomOAuthAdapter) ResolveProfile(ctx context.Context, code string) (auth.ProviderProfile, error) {
//		// Implement token exchange and profile fetching
//		// Return profile with Name and AvatarURL populated
//		return auth.ProviderProfile{
//			ProviderUserID: "provider-user-id",
//			Email:          "user@example.com",
//			EmailVerified:  true,
//			Name:           "User Name", // Populated from provider
//			AvatarURL:      "https://provider.com/avatar.jpg", // Populated from provider
//		}, nil
//	}
//
// # Constants
//
// The package exports constants for authentication methods and token subjects:
//
//	// Authentication method identifiers
//	auth.MethodPassword    // "password"
//	auth.MethodMagicLink   // "magic_link"
//	auth.MethodOAuthGoogle // "oauth_google"
//	auth.MethodOAuthGithub // "oauth_github"
//
//	// JWT token subjects
//	auth.SubjectPasswordReset // "password_reset"
//	auth.SubjectEmailVerify   // "email_verify"
//	auth.SubjectEmailChange   // "email_change"
//	auth.SubjectMagicLink     // "magic_link"
//
//	// OAuth provider identifiers
//	auth.OAuthProviderGoogle // "google"
//	auth.OAuthProviderGithub // "github"
//
// # Thread Safety
//
// All service implementations are safe for concurrent use. Storage interface implementations
// must ensure thread-safety for their specific persistence layer.
//
// # Dependencies
//
// The package requires the following external dependencies:
//
//   - github.com/google/uuid for secure UUID generation
//   - golang.org/x/crypto/bcrypt for password hashing
//   - golang.org/x/oauth2 for OAuth 2.0 flows
//   - github.com/dmitrymomot/saaskit/pkg/validator for input validation
//   - github.com/dmitrymomot/saaskit/pkg/sanitizer for data normalization
//   - github.com/dmitrymomot/saaskit/pkg/token for JWT token handling
//   - github.com/dmitrymomot/saaskit/pkg/logger for structured logging
package auth
