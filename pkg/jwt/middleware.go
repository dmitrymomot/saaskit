package jwt

import (
	"net/http"
	"strings"
)

// TokenExtractorFunc defines a function that extracts a token from an HTTP request.
type TokenExtractorFunc func(r *http.Request) (string, error)

// SkipFunc defines a function that determines whether to skip JWT validation for a request.
type SkipFunc func(r *http.Request) bool

// MiddlewareConfig configures JWT middleware behavior.
type MiddlewareConfig struct {
	Service   *Service           // JWT service for token validation
	Extractor TokenExtractorFunc // Token extraction strategy (defaults to Bearer)
	Skip      SkipFunc           // Optional request filter to bypass validation
}

// Middleware creates JWT middleware with default Bearer token extraction.
// Validates tokens and injects claims into request context for downstream handlers.
func Middleware(service *Service) func(next http.Handler) http.Handler {
	return MiddlewareWithConfig(MiddlewareConfig{
		Service:   service,
		Extractor: BearerTokenExtractor,
	})
}

// MiddlewareWithConfig creates JWT middleware with custom configuration.
func MiddlewareWithConfig(config MiddlewareConfig) func(next http.Handler) http.Handler {
	if config.Extractor == nil {
		config.Extractor = BearerTokenExtractor
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.Skip != nil && config.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			tokenString, err := config.Extractor(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			// Parse to map[string]any for maximum flexibility
			claims := make(map[string]any)
			if err := config.Service.Parse(tokenString, &claims); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			ctx := r.Context()
			ctx = SetToken(ctx, tokenString)
			ctx = SetClaims(ctx, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// BearerTokenExtractor extracts JWT tokens from "Authorization: Bearer <token>" headers.
// This is the most common JWT transport method per RFC 6750.
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

// CookieTokenExtractor creates a token extractor for cookie-based JWT transport.
// Useful for browser applications where Authorization headers aren't practical.
func CookieTokenExtractor(cookieName string) TokenExtractorFunc {
	return func(r *http.Request) (string, error) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			return "", ErrInvalidToken
		}
		return cookie.Value, nil
	}
}

// QueryTokenExtractor creates a token extractor for URL query parameters.
// Generally discouraged due to token exposure in logs and referrer headers.
func QueryTokenExtractor(paramName string) TokenExtractorFunc {
	return func(r *http.Request) (string, error) {
		token := r.URL.Query().Get(paramName)
		if token == "" {
			return "", ErrInvalidToken
		}
		return token, nil
	}
}

// HeaderTokenExtractor creates a token extractor for custom headers.
// Useful for APIs that use non-standard header names for token transport.
func HeaderTokenExtractor(headerName string) TokenExtractorFunc {
	return func(r *http.Request) (string, error) {
		token := r.Header.Get(headerName)
		if token == "" {
			return "", ErrInvalidToken
		}
		return token, nil
	}
}
