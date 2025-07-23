package jwt

import (
	"net/http"
	"strings"
)

// TokenExtractorFunc defines a function that extracts a token from an HTTP request
type TokenExtractorFunc func(r *http.Request) (string, error)

// SkipFunc defines a function that determines whether to skip the middleware
type SkipFunc func(r *http.Request) bool

// MiddlewareConfig contains configuration for the JWT middleware
type MiddlewareConfig struct {
	// Service is the JWT service to use for parsing tokens
	Service *Service

	// Extractor is a function that extracts a token from an HTTP request
	// If not specified, BearerTokenExtractor is used
	Extractor TokenExtractorFunc

	// Skip is a function that determines whether to skip the middleware
	// If not specified, the middleware is never skipped
	Skip SkipFunc
}

// Middleware returns a new JWT middleware handler with default configuration.
// The default configuration uses the BearerTokenExtractor and does not skip the middleware.
// The claims are stored in the request context using the default ContextKey.
func Middleware(service *Service) func(next http.Handler) http.Handler {
	return MiddlewareWithConfig(MiddlewareConfig{
		Service:   service,
		Extractor: BearerTokenExtractor,
	})
}

// Middleware returns a new JWT middleware handler
func MiddlewareWithConfig(config MiddlewareConfig) func(next http.Handler) http.Handler {
	// Use default token extractor if none is provided
	if config.Extractor == nil {
		config.Extractor = BearerTokenExtractor
	}

	// Return the middleware handler
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if we should skip the middleware
			if config.Skip != nil && config.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract the token from the request
			tokenString, err := config.Extractor(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			// Parse the token
			claims := make(map[string]any)
			if err := config.Service.Parse(tokenString, &claims); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			ctx := r.Context()

			// Add the token to the request context
			ctx = SetToken(ctx, tokenString)

			// Add the claims to the request context
			ctx = SetClaims(ctx, claims)

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// BearerTokenExtractor extracts a JWT token from the Authorization header
// It expects the format "Bearer <token>"
func BearerTokenExtractor(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", ErrInvalidToken
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", ErrInvalidToken
	}

	return parts[1], nil
}

// CookieTokenExtractor creates a token extractor that extracts tokens from a cookie
func CookieTokenExtractor(cookieName string) TokenExtractorFunc {
	return func(r *http.Request) (string, error) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			return "", ErrInvalidToken
		}
		return cookie.Value, nil
	}
}

// QueryTokenExtractor creates a token extractor that extracts tokens from a query parameter
func QueryTokenExtractor(paramName string) TokenExtractorFunc {
	return func(r *http.Request) (string, error) {
		token := r.URL.Query().Get(paramName)
		if token == "" {
			return "", ErrInvalidToken
		}
		return token, nil
	}
}

// HeaderTokenExtractor creates a token extractor that extracts tokens from a header
func HeaderTokenExtractor(headerName string) TokenExtractorFunc {
	return func(r *http.Request) (string, error) {
		token := r.Header.Get(headerName)
		if token == "" {
			return "", ErrInvalidToken
		}
		return token, nil
	}
}
