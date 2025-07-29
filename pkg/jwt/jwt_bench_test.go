package jwt_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/jwt"
)

// BenchmarkGenerate benchmarks JWT token generation
func BenchmarkGenerate(b *testing.B) {
	service, err := jwt.New([]byte("benchmark-secret-key"))
	require.NoError(b, err)
	require.NotNil(b, service)

	b.Run("StandardClaims", func(b *testing.B) {
		claims := jwt.StandardClaims{
			Subject:   "user123",
			Issuer:    "saaskit-benchmark",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
			IssuedAt:  time.Now().Unix(),
			ID:        "token-id-123",
			Audience:  "api-users",
		}

		b.ResetTimer()
		for b.Loop() {
			token, err := service.Generate(claims)
			if err != nil {
				b.Fatal(err)
			}
			if token == "" {
				b.Fatal("empty token")
			}
		}
	})

	b.Run("CustomClaims", func(b *testing.B) {
		type BenchClaims struct {
			jwt.StandardClaims
			UserID    string         `json:"user_id"`
			Email     string         `json:"email"`
			FirstName string         `json:"first_name"`
			LastName  string         `json:"last_name"`
			Roles     []string       `json:"roles"`
			Metadata  map[string]any `json:"metadata"`
		}

		claims := BenchClaims{
			StandardClaims: jwt.StandardClaims{
				Subject:   "user456",
				Issuer:    "saaskit-benchmark",
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
				IssuedAt:  time.Now().Unix(),
				ID:        "token-id-456",
				Audience:  "api-users",
			},
			UserID:    "usr_123456789",
			Email:     "user@example.com",
			FirstName: "John",
			LastName:  "Doe",
			Roles:     []string{"admin", "user", "manager"},
			Metadata: map[string]any{
				"login_count": 42,
				"last_login":  time.Now().Format(time.RFC3339),
				"preferences": map[string]string{
					"theme":    "dark",
					"timezone": "UTC",
					"language": "en",
				},
			},
		}

		b.ResetTimer()
		for b.Loop() {
			token, err := service.Generate(claims)
			if err != nil {
				b.Fatal(err)
			}
			if token == "" {
				b.Fatal("empty token")
			}
		}
	})
}

// BenchmarkParse benchmarks JWT token parsing
func BenchmarkParse(b *testing.B) {
	service, err := jwt.New([]byte("benchmark-secret-key"))
	require.NoError(b, err)
	require.NotNil(b, service)

	b.Run("StandardClaims", func(b *testing.B) {
		// Generate a token once for parsing benchmark
		standardClaims := jwt.StandardClaims{
			Subject:   "user123",
			Issuer:    "saaskit-benchmark",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
			IssuedAt:  time.Now().Unix(),
			ID:        "token-id-123",
			Audience:  "api-users",
		}

		token, err := service.Generate(standardClaims)
		require.NoError(b, err)
		require.NotEmpty(b, token)

		b.ResetTimer()
		for b.Loop() {
			var parsedClaims jwt.StandardClaims
			err = service.Parse(token, &parsedClaims)
			if err != nil {
				b.Fatal(err)
			}
			// Quick sanity check
			if parsedClaims.Subject != standardClaims.Subject {
				b.Fatal("subject mismatch")
			}
		}
	})

	b.Run("CustomClaims", func(b *testing.B) {
		// Define a complex custom claims type
		type BenchClaims struct {
			jwt.StandardClaims
			UserID    string         `json:"user_id"`
			Email     string         `json:"email"`
			FirstName string         `json:"first_name"`
			LastName  string         `json:"last_name"`
			Roles     []string       `json:"roles"`
			Metadata  map[string]any `json:"metadata"`
		}

		// Generate a token once for parsing benchmark
		originalClaims := BenchClaims{
			StandardClaims: jwt.StandardClaims{
				Subject:   "user456",
				Issuer:    "saaskit-benchmark",
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
				IssuedAt:  time.Now().Unix(),
				ID:        "token-id-456",
				Audience:  "api-users",
			},
			UserID:    "usr_123456789",
			Email:     "user@example.com",
			FirstName: "John",
			LastName:  "Doe",
			Roles:     []string{"admin", "user", "manager"},
			Metadata: map[string]any{
				"login_count": 42,
				"last_login":  time.Now().Format(time.RFC3339),
				"preferences": map[string]string{
					"theme":    "dark",
					"timezone": "UTC",
					"language": "en",
				},
			},
		}

		token, err := service.Generate(originalClaims)
		require.NoError(b, err)
		require.NotEmpty(b, token)

		b.ResetTimer()
		for b.Loop() {
			var parsedClaims BenchClaims
			err = service.Parse(token, &parsedClaims)
			if err != nil {
				b.Fatal(err)
			}
			// Quick sanity check
			if parsedClaims.UserID != originalClaims.UserID {
				b.Fatal("user ID mismatch")
			}
		}
	})
}

// BenchmarkEnd2End benchmarks the entire JWT lifecycle (generation + parsing)
func BenchmarkEnd2End(b *testing.B) {
	service, err := jwt.New([]byte("benchmark-secret-key"))
	require.NoError(b, err)
	require.NotNil(b, service)

	b.Run("StandardClaims", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			// Generate claims with unique ID to prevent caching
			claims := jwt.StandardClaims{
				Subject:   "user123",
				Issuer:    "saaskit-benchmark",
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
				IssuedAt:  time.Now().Unix(),
				ID:        fmt.Sprintf("token-id-%d", time.Now().UnixNano()),
			}

			// Generate token
			token, err := service.Generate(claims)
			if err != nil {
				b.Fatal(err)
			}

			// Parse token
			var parsedClaims jwt.StandardClaims
			err = service.Parse(token, &parsedClaims)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("CustomClaims", func(b *testing.B) {
		type BenchClaims struct {
			jwt.StandardClaims
			UserID string   `json:"user_id"`
			Email  string   `json:"email"`
			Roles  []string `json:"roles"`
		}

		b.ResetTimer()
		for b.Loop() {
			// Generate claims with unique ID to prevent caching
			claims := BenchClaims{
				StandardClaims: jwt.StandardClaims{
					Subject:   "user456",
					Issuer:    "saaskit-benchmark",
					ExpiresAt: time.Now().Add(time.Hour).Unix(),
					IssuedAt:  time.Now().Unix(),
					ID:        fmt.Sprintf("token-id-%d", time.Now().UnixNano()),
				},
				UserID: "usr_123456789",
				Email:  "user@example.com",
				Roles:  []string{"admin", "user"},
			}

			// Generate token
			token, err := service.Generate(claims)
			if err != nil {
				b.Fatal(err)
			}

			// Parse token
			var parsedClaims BenchClaims
			err = service.Parse(token, &parsedClaims)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
