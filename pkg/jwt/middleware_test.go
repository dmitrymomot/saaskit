package jwt_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/jwt"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()
	// Create a JWT service for testing
	service, err := jwt.New([]byte("test-secret"))
	require.NoError(t, err)
	require.NotNil(t, service)

	// Create test claims
	testClaims := jwt.StandardClaims{
		Subject:   "test-user",
		Issuer:    "test-issuer",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	// Generate a test token
	token, err := service.Generate(testClaims)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Create a test handler that checks for claims in the context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get claims from context using the GetClaims helper
		claims, ok := jwt.GetClaims[map[string]any](r.Context())
		if !ok {
			http.Error(w, "Claims not found in context", http.StatusInternalServerError)
			return
		}

		// Check if the claims contain expected values
		if claims["sub"] != testClaims.Subject {
			http.Error(w, "Subject mismatch", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	t.Run("DefaultTokenExtractor", func(t *testing.T) {
		// Create middleware with default extractor
		middleware := jwt.Middleware(service)

		// Create a test server with the middleware
		handler := middleware(testHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		// Create a request with the token in the Authorization header
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check the response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("MissingToken", func(t *testing.T) {
		// Create middleware with default extractor
		middleware := jwt.Middleware(service)

		// Create a test server with the middleware
		handler := middleware(testHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		// Create a request without a token
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check the response (should be unauthorized)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("InvalidToken", func(t *testing.T) {
		// Create middleware with default extractor
		middleware := jwt.Middleware(service)

		// Create a test server with the middleware
		handler := middleware(testHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		// Create a request with an invalid token
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer invalid-token")

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check the response (should be unauthorized)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("CustomExtractor", func(t *testing.T) {
		// Create middleware with a custom extractor
		middleware := jwt.MiddlewareWithConfig(jwt.MiddlewareConfig{
			Service:   service,
			Extractor: jwt.HeaderTokenExtractor("X-Auth-Token"),
		})

		// Create a test server with the middleware
		handler := middleware(testHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		// Create a request with the token in the custom header
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		req.Header.Set("X-Auth-Token", token)

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check the response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("CookieTokenExtractor", func(t *testing.T) {
		// Create middleware with cookie extractor
		middleware := jwt.MiddlewareWithConfig(jwt.MiddlewareConfig{
			Service:   service,
			Extractor: jwt.CookieTokenExtractor("jwt"),
		})

		// Create a test server with the middleware
		handler := middleware(testHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		// Create a request with the token in a cookie
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{
			Name:  "jwt",
			Value: token,
		})

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check the response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("QueryTokenExtractor", func(t *testing.T) {
		// Create middleware with query extractor
		middleware := jwt.MiddlewareWithConfig(jwt.MiddlewareConfig{
			Service:   service,
			Extractor: jwt.QueryTokenExtractor("token"),
		})

		// Create a test server with the middleware
		handler := middleware(testHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		// Create a request with the token in a query parameter
		req, err := http.NewRequest("GET", server.URL+"?token="+token, nil)
		require.NoError(t, err)

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check the response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("SkipMiddleware", func(t *testing.T) {
		// Create a skip function that skips requests to a specific path
		skipFunc := func(r *http.Request) bool {
			return r.URL.Path == "/skip"
		}

		// Create middleware with the skip function
		middleware := jwt.MiddlewareWithConfig(jwt.MiddlewareConfig{
			Service: service,
			Skip:    skipFunc,
		})

		// Create a test handler that always succeeds
		skipHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("skipped"))
		})

		// Create a test server with the middleware
		handler := middleware(skipHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		// Create a request to the skip path without a token
		req, err := http.NewRequest("GET", server.URL+"/skip", nil)
		require.NoError(t, err)

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check the response (should be OK even without a token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Create a request to a different path without a token
		req, err = http.NewRequest("GET", server.URL+"/other", nil)
		require.NoError(t, err)

		// Send the request
		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check the response (should be unauthorized)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("GetClaimsAs", func(t *testing.T) {
		// Create a handler that uses GetClaimsAs to parse claims into a struct
		typedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var claims jwt.StandardClaims
			err := jwt.GetClaimsAs(r.Context(), &claims)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Check if the claims contain expected values
			if claims.Subject != testClaims.Subject {
				http.Error(w, "Subject mismatch", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success"))
		})

		// Create middleware with default extractor
		middleware := jwt.Middleware(service)

		// Create a test server with the middleware
		handler := middleware(typedHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		// Create a request with the token in the Authorization header
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check the response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("HeaderTokenExtractor", func(t *testing.T) {
		// Create middleware using the HeaderTokenExtractor
		middleware := jwt.MiddlewareWithConfig(jwt.MiddlewareConfig{
			Service:   service,
			Extractor: jwt.HeaderTokenExtractor("X-API-Token"),
		})

		// Create a test server with the middleware
		handler := middleware(testHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		// Create a request with the token in the custom header
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		req.Header.Set("X-API-Token", token)

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check the response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
