package auth

import "context"

// OAuth provider identifiers used across the auth system.
const (
	OAuthProviderGoogle = "google"
	OAuthProviderGithub = "github"
)

// ProviderAdapter abstracts provider-specific OAuth behavior behind a minimal,
// provider-agnostic interface. Implementations should encapsulate all protocol
// details (e.g., oauth2.Config, token exchange, API calls) and expose only
// primitives required by the core OAuth service.
type ProviderAdapter interface {
	// ProviderID returns a stable provider identifier used for storage and logging,
	// e.g., "google", "github".
	ProviderID() string

	// AuthURL builds the provider authorization URL for the given state token.
	// Implementations may include provider-specific options (e.g., offline access).
	AuthURL(state string) (string, error)

	// ResolveProfile performs the end-to-end flow for an authorization code:
	// - exchanges the code for an access token
	// - calls the provider's user/profile endpoint(s)
	// - returns a normalized ProviderProfile
	//
	// Notes:
	// - On invalid code or token exchange failures, return ErrInvalidCode.
	// - If the provider cannot produce an email, return ErrNoPrimaryEmail.
	// - Email normalization (lowercasing, trimming, etc.) is done in the core service.
	ResolveProfile(ctx context.Context, code string) (ProviderProfile, error)
}

// ProviderProfile represents the normalized user profile returned by a provider.
// The core OAuth service uses this information to enforce security policies
// (verified-only), prevent account takeover, and create/link local users.
type ProviderProfile struct {
	// ProviderUserID is the provider's stable user identifier, represented as a string.
	// Implementations should convert numeric IDs (e.g., GitHub) to string.
	ProviderUserID string

	// Email is the raw email returned by the provider. The core service will
	// normalize it (e.g., lowercase) before using it.
	Email string

	// EmailVerified indicates whether the provider asserts the email is verified.
	EmailVerified bool

	// Name is the display name from the provider (optional).
	Name string

	// AvatarURL is the URL to the user's avatar image (optional).
	AvatarURL string
}
