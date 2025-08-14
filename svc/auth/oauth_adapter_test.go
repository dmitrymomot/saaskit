package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// Test GitHub OAuth Adapter
func TestGitHubAdapter_ProviderID(t *testing.T) {
	t.Parallel()

	cfg := GitHubOAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "https://example.com/callback",
	}

	adapter := NewGitHubAdapter(cfg)
	assert.Equal(t, OAuthProviderGithub, adapter.ProviderID())
}

func TestGitHubAdapter_AuthURL(t *testing.T) {
	t.Parallel()

	cfg := GitHubOAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "https://example.com/callback",
		Scopes:       []string{"user:email"},
	}

	adapter := NewGitHubAdapter(cfg)
	state := "test-state-token"

	authURL, err := adapter.AuthURL(state)

	require.NoError(t, err)
	require.NotEmpty(t, authURL)

	// Verify URL contains expected parameters (URL encoding can vary)
	assert.Contains(t, authURL, "client_id=test-client-id")
	assert.Contains(t, authURL, "example.com")
	assert.Contains(t, authURL, "state=test-state-token")
	assert.Contains(t, authURL, "user")  // part of scope
	assert.Contains(t, authURL, "email") // part of scope
	assert.Contains(t, authURL, "access_type=offline")
	assert.Contains(t, authURL, "github.com/login/oauth/authorize")
}

func TestGitHubAdapter_ResolveProfile(t *testing.T) {
	t.Parallel()

	t.Run("resolves profile with primary verified email", func(t *testing.T) {
		t.Parallel()

		// Mock GitHub API responses
		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Mock token exchange endpoint
			assert.Equal(t, http.MethodPost, r.Method)
			tokenResponse := map[string]any{
				"access_token": "test-access-token",
				"token_type":   "Bearer",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse)
		}))
		defer tokenServer.Close()

		userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer test-access-token", r.Header.Get("Authorization"))

			if strings.Contains(r.URL.Path, "/user/emails") {
				// Mock emails endpoint
				emailsResponse := []ghEmail{
					{Email: "secondary@example.com", Primary: false, Verified: true},
					{Email: "primary@example.com", Primary: true, Verified: true},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(emailsResponse)
			} else if strings.Contains(r.URL.Path, "/user") {
				// Mock user endpoint
				userResponse := ghUser{ID: 12345}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(userResponse)
			} else {
				t.Errorf("Unexpected request to %s", r.URL.Path)
			}
		}))
		defer userServer.Close()

		// Create adapter with custom HTTP client that redirects API calls to our mock servers
		cfg := GitHubOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}
		adapter := NewGitHubAdapter(cfg).(*githubAdapter)

		// Override the oauth config to use our token server
		adapter.conf = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServer.URL,
			},
		}

		// Create custom HTTP client that redirects GitHub API calls to our mock server
		transport := &mockTransport{
			userServer: userServer.URL,
		}
		adapter.httpClient = &http.Client{Transport: transport}

		ctx := context.Background()
		profile, err := adapter.ResolveProfile(ctx, "valid-code")

		require.NoError(t, err)
		assert.Equal(t, "12345", profile.ProviderUserID)
		assert.Equal(t, "primary@example.com", profile.Email) // Should prefer primary
		assert.True(t, profile.EmailVerified)
	})

	t.Run("resolves profile with any verified email when no primary", func(t *testing.T) {
		t.Parallel()

		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenResponse := map[string]any{
				"access_token": "test-access-token",
				"token_type":   "Bearer",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse)
		}))
		defer tokenServer.Close()

		userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/user/emails") {
				// Only verified, not primary emails
				emailsResponse := []ghEmail{
					{Email: "unverified@example.com", Primary: true, Verified: false},
					{Email: "verified@example.com", Primary: false, Verified: true},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(emailsResponse)
			} else if strings.Contains(r.URL.Path, "/user") {
				userResponse := ghUser{ID: 12345}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(userResponse)
			}
		}))
		defer userServer.Close()

		cfg := GitHubOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}
		adapter := NewGitHubAdapter(cfg).(*githubAdapter)

		adapter.conf = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServer.URL,
			},
		}

		transport := &mockTransport{userServer: userServer.URL}
		adapter.httpClient = &http.Client{Transport: transport}

		ctx := context.Background()
		profile, err := adapter.ResolveProfile(ctx, "valid-code")

		require.NoError(t, err)
		assert.Equal(t, "12345", profile.ProviderUserID)
		assert.Equal(t, "verified@example.com", profile.Email) // Should use verified email
		assert.True(t, profile.EmailVerified)
	})

	t.Run("returns error when no primary email available", func(t *testing.T) {
		t.Parallel()

		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenResponse := map[string]any{
				"access_token": "test-access-token",
				"token_type":   "Bearer",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse)
		}))
		defer tokenServer.Close()

		userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/user/emails") {
				// No verified emails
				emailsResponse := []ghEmail{
					{Email: "unverified1@example.com", Primary: true, Verified: false},
					{Email: "unverified2@example.com", Primary: false, Verified: false},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(emailsResponse)
			} else if strings.Contains(r.URL.Path, "/user") {
				userResponse := ghUser{ID: 12345}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(userResponse)
			}
		}))
		defer userServer.Close()

		cfg := GitHubOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}
		adapter := NewGitHubAdapter(cfg).(*githubAdapter)

		adapter.conf = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServer.URL,
			},
		}

		transport := &mockTransport{userServer: userServer.URL}
		adapter.httpClient = &http.Client{Transport: transport}

		ctx := context.Background()
		profile, err := adapter.ResolveProfile(ctx, "valid-code")

		assert.Equal(t, ErrNoPrimaryEmail, err)
		assert.Empty(t, profile.ProviderUserID)
	})

	t.Run("returns invalid code error for token exchange failures", func(t *testing.T) {
		t.Parallel()

		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Return error for invalid code
			http.Error(w, "invalid_grant", http.StatusBadRequest)
		}))
		defer tokenServer.Close()

		cfg := GitHubOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}
		adapter := NewGitHubAdapter(cfg).(*githubAdapter)

		adapter.conf = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServer.URL,
			},
		}

		ctx := context.Background()
		profile, err := adapter.ResolveProfile(ctx, "invalid-code")

		assert.Equal(t, ErrInvalidCode, err)
		assert.Empty(t, profile.ProviderUserID)
	})

	t.Run("handles API errors gracefully", func(t *testing.T) {
		t.Parallel()

		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenResponse := map[string]any{
				"access_token": "test-access-token",
				"token_type":   "Bearer",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse)
		}))
		defer tokenServer.Close()

		userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/user/emails") {
				// Return API error for emails endpoint
				http.Error(w, "API Error", http.StatusInternalServerError)
			} else if strings.Contains(r.URL.Path, "/user") {
				userResponse := ghUser{ID: 12345}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(userResponse)
			}
		}))
		defer userServer.Close()

		cfg := GitHubOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}
		adapter := NewGitHubAdapter(cfg).(*githubAdapter)

		adapter.conf = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServer.URL,
			},
		}

		transport := &mockTransport{userServer: userServer.URL}
		adapter.httpClient = &http.Client{Transport: transport}

		ctx := context.Background()
		profile, err := adapter.ResolveProfile(ctx, "valid-code")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fetch github emails")
		assert.Empty(t, profile.ProviderUserID)
	})
}

// Test Google OAuth Adapter
func TestGoogleAdapter_ProviderID(t *testing.T) {
	t.Parallel()

	cfg := GoogleOAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "https://example.com/callback",
	}

	adapter := NewGoogleAdapter(cfg)
	assert.Equal(t, OAuthProviderGoogle, adapter.ProviderID())
}

func TestGoogleAdapter_AuthURL(t *testing.T) {
	t.Parallel()

	cfg := GoogleOAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "https://example.com/callback",
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
	}

	adapter := NewGoogleAdapter(cfg)
	state := "test-state-token"

	authURL, err := adapter.AuthURL(state)

	require.NoError(t, err)
	require.NotEmpty(t, authURL)

	// Verify URL contains expected parameters (URL encoding can vary)
	assert.Contains(t, authURL, "client_id=test-client-id")
	assert.Contains(t, authURL, "example.com")
	assert.Contains(t, authURL, "state=test-state-token")
	assert.Contains(t, authURL, "googleapis.com") // part of scope
	assert.Contains(t, authURL, "userinfo")       // part of scope
	assert.Contains(t, authURL, "email")          // part of scope
	assert.Contains(t, authURL, "access_type=offline")
	assert.Contains(t, authURL, "accounts.google.com/o/oauth2/auth")
}

func TestGoogleAdapter_ResolveProfile(t *testing.T) {
	t.Parallel()

	t.Run("resolves profile with verified email", func(t *testing.T) {
		t.Parallel()

		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenResponse := map[string]any{
				"access_token": "test-access-token",
				"token_type":   "Bearer",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse)
		}))
		defer tokenServer.Close()

		userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer test-access-token", r.Header.Get("Authorization"))

			userResponse := gUser{
				ID:            "google-user-123",
				Email:         "user@example.com",
				VerifiedEmail: true,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userResponse)
		}))
		defer userServer.Close()

		cfg := GoogleOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}
		adapter := NewGoogleAdapter(cfg).(*googleAdapter)

		adapter.conf = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServer.URL,
			},
		}

		transport := &mockTransport{userServer: userServer.URL}
		adapter.httpClient = &http.Client{Transport: transport}

		ctx := context.Background()
		profile, err := adapter.ResolveProfile(ctx, "valid-code")

		require.NoError(t, err)
		assert.Equal(t, "google-user-123", profile.ProviderUserID)
		assert.Equal(t, "user@example.com", profile.Email)
		assert.True(t, profile.EmailVerified)
	})

	t.Run("resolves profile with unverified email", func(t *testing.T) {
		t.Parallel()

		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenResponse := map[string]any{
				"access_token": "test-access-token",
				"token_type":   "Bearer",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse)
		}))
		defer tokenServer.Close()

		userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userResponse := gUser{
				ID:            "google-user-123",
				Email:         "unverified@example.com",
				VerifiedEmail: false,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userResponse)
		}))
		defer userServer.Close()

		cfg := GoogleOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}
		adapter := NewGoogleAdapter(cfg).(*googleAdapter)

		adapter.conf = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServer.URL,
			},
		}

		transport := &mockTransport{userServer: userServer.URL}
		adapter.httpClient = &http.Client{Transport: transport}

		ctx := context.Background()
		profile, err := adapter.ResolveProfile(ctx, "valid-code")

		require.NoError(t, err)
		assert.Equal(t, "google-user-123", profile.ProviderUserID)
		assert.Equal(t, "unverified@example.com", profile.Email)
		assert.False(t, profile.EmailVerified)
	})

	t.Run("returns error when no email available", func(t *testing.T) {
		t.Parallel()

		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenResponse := map[string]any{
				"access_token": "test-access-token",
				"token_type":   "Bearer",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse)
		}))
		defer tokenServer.Close()

		userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userResponse := gUser{
				ID:            "google-user-123",
				Email:         "", // No email
				VerifiedEmail: false,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userResponse)
		}))
		defer userServer.Close()

		cfg := GoogleOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}
		adapter := NewGoogleAdapter(cfg).(*googleAdapter)

		adapter.conf = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServer.URL,
			},
		}

		transport := &mockTransport{userServer: userServer.URL}
		adapter.httpClient = &http.Client{Transport: transport}

		ctx := context.Background()
		profile, err := adapter.ResolveProfile(ctx, "valid-code")

		assert.Equal(t, ErrNoPrimaryEmail, err)
		assert.Empty(t, profile.ProviderUserID)
	})

	t.Run("returns invalid code error for token exchange failures", func(t *testing.T) {
		t.Parallel()

		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "invalid_grant", http.StatusBadRequest)
		}))
		defer tokenServer.Close()

		cfg := GoogleOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}
		adapter := NewGoogleAdapter(cfg).(*googleAdapter)

		adapter.conf = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServer.URL,
			},
		}

		ctx := context.Background()
		profile, err := adapter.ResolveProfile(ctx, "invalid-code")

		assert.Equal(t, ErrInvalidCode, err)
		assert.Empty(t, profile.ProviderUserID)
	})

	t.Run("handles API errors gracefully", func(t *testing.T) {
		t.Parallel()

		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenResponse := map[string]any{
				"access_token": "test-access-token",
				"token_type":   "Bearer",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse)
		}))
		defer tokenServer.Close()

		userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "API Error", http.StatusInternalServerError)
		}))
		defer userServer.Close()

		cfg := GoogleOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}
		adapter := NewGoogleAdapter(cfg).(*googleAdapter)

		adapter.conf = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServer.URL,
			},
		}

		transport := &mockTransport{userServer: userServer.URL}
		adapter.httpClient = &http.Client{Transport: transport}

		ctx := context.Background()
		profile, err := adapter.ResolveProfile(ctx, "valid-code")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fetch google user")
		assert.Empty(t, profile.ProviderUserID)
	})
}

func TestAdapterTimeouts(t *testing.T) {
	t.Parallel()

	t.Run("GitHub adapter has reasonable timeout", func(t *testing.T) {
		t.Parallel()

		cfg := GitHubOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}

		adapter := NewGitHubAdapter(cfg).(*githubAdapter)
		assert.Equal(t, 10*time.Second, adapter.httpClient.Timeout)
	})

	t.Run("Google adapter has reasonable timeout", func(t *testing.T) {
		t.Parallel()

		cfg := GoogleOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}

		adapter := NewGoogleAdapter(cfg).(*googleAdapter)
		assert.Equal(t, 10*time.Second, adapter.httpClient.Timeout)
	})
}

func TestAdapterInterfaces(t *testing.T) {
	t.Parallel()

	t.Run("GitHub adapter implements ProviderAdapter", func(t *testing.T) {
		t.Parallel()

		cfg := GitHubOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}

		var adapter ProviderAdapter = NewGitHubAdapter(cfg)
		require.NotNil(t, adapter)
		// If this compiles, the interface is correctly implemented
	})

	t.Run("Google adapter implements ProviderAdapter", func(t *testing.T) {
		t.Parallel()

		cfg := GoogleOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}

		var adapter ProviderAdapter = NewGoogleAdapter(cfg)
		require.NotNil(t, adapter)
		// If this compiles, the interface is correctly implemented
	})
}

// Test adapter configurations
func TestAdapterConfigurations(t *testing.T) {
	t.Parallel()

	t.Run("GitHub adapter uses correct OAuth endpoints", func(t *testing.T) {
		t.Parallel()

		cfg := GitHubOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
			Scopes:       []string{"user:email", "read:user"},
		}

		adapter := NewGitHubAdapter(cfg).(*githubAdapter)

		assert.Equal(t, cfg.ClientID, adapter.conf.ClientID)
		assert.Equal(t, cfg.ClientSecret, adapter.conf.ClientSecret)
		assert.Equal(t, cfg.RedirectURL, adapter.conf.RedirectURL)
		assert.Equal(t, cfg.Scopes, adapter.conf.Scopes)
		assert.Contains(t, adapter.conf.Endpoint.AuthURL, "github.com")
		assert.Contains(t, adapter.conf.Endpoint.TokenURL, "github.com")
	})

	t.Run("Google adapter uses correct OAuth endpoints", func(t *testing.T) {
		t.Parallel()

		cfg := GoogleOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		}

		adapter := NewGoogleAdapter(cfg).(*googleAdapter)

		assert.Equal(t, cfg.ClientID, adapter.conf.ClientID)
		assert.Equal(t, cfg.ClientSecret, adapter.conf.ClientSecret)
		assert.Equal(t, cfg.RedirectURL, adapter.conf.RedirectURL)
		assert.Equal(t, cfg.Scopes, adapter.conf.Scopes)
		assert.Contains(t, adapter.conf.Endpoint.AuthURL, "accounts.google.com")
		assert.Contains(t, adapter.conf.Endpoint.TokenURL, "oauth2.googleapis.com")
	})
}

// mockTransport is a custom transport for redirecting API calls to mock servers
type mockTransport struct {
	userServer string
}

func (mt *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch {
	case strings.Contains(req.URL.Host, "api.github.com"):
		// Redirect GitHub API calls to mock server
		req.URL.Host = strings.TrimPrefix(mt.userServer, "http://")
		req.URL.Scheme = "http"
	case strings.Contains(req.URL.Host, "googleapis.com"):
		// Redirect Google API calls to mock server
		req.URL.Host = strings.TrimPrefix(mt.userServer, "http://")
		req.URL.Scheme = "http"
	}

	return http.DefaultTransport.RoundTrip(req)
}
