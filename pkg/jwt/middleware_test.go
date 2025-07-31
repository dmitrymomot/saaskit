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
	service, err := jwt.New([]byte("test-secret"))
	require.NoError(t, err)
	require.NotNil(t, service)

	testClaims := jwt.StandardClaims{
		Subject:   "test-user",
		Issuer:    "test-issuer",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := service.Generate(testClaims)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := jwt.GetClaims[map[string]any](r.Context())
		if !ok {
			http.Error(w, "Claims not found in context", http.StatusInternalServerError)
			return
		}

		if claims["sub"] != testClaims.Subject {
			http.Error(w, "Subject mismatch", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	t.Run("DefaultTokenExtractor", func(t *testing.T) {
		middleware := jwt.Middleware(service)

		handler := middleware(testHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("MissingToken", func(t *testing.T) {
		middleware := jwt.Middleware(service)

		handler := middleware(testHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("InvalidToken", func(t *testing.T) {
		middleware := jwt.Middleware(service)

		handler := middleware(testHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer invalid-token")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("CustomExtractor", func(t *testing.T) {
		middleware := jwt.MiddlewareWithConfig(jwt.MiddlewareConfig{
			Service:   service,
			Extractor: jwt.HeaderTokenExtractor("X-Auth-Token"),
		})

		handler := middleware(testHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		req.Header.Set("X-Auth-Token", token)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("CookieTokenExtractor", func(t *testing.T) {
		middleware := jwt.MiddlewareWithConfig(jwt.MiddlewareConfig{
			Service:   service,
			Extractor: jwt.CookieTokenExtractor("jwt"),
		})

		handler := middleware(testHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{
			Name:  "jwt",
			Value: token,
		})

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("QueryTokenExtractor", func(t *testing.T) {
		middleware := jwt.MiddlewareWithConfig(jwt.MiddlewareConfig{
			Service:   service,
			Extractor: jwt.QueryTokenExtractor("token"),
		})

		handler := middleware(testHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		req, err := http.NewRequest("GET", server.URL+"?token="+token, nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("SkipMiddleware", func(t *testing.T) {
		skipFunc := func(r *http.Request) bool {
			return r.URL.Path == "/skip"
		}

		middleware := jwt.MiddlewareWithConfig(jwt.MiddlewareConfig{
			Service: service,
			Skip:    skipFunc,
		})

		skipHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("skipped"))
		})

		handler := middleware(skipHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		req, err := http.NewRequest("GET", server.URL+"/skip", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		req, err = http.NewRequest("GET", server.URL+"/other", nil)
		require.NoError(t, err)

		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("GetClaimsAs", func(t *testing.T) {
		typedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var claims jwt.StandardClaims
			err := jwt.GetClaimsAs(r.Context(), &claims)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if claims.Subject != testClaims.Subject {
				http.Error(w, "Subject mismatch", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success"))
		})

		middleware := jwt.Middleware(service)

		handler := middleware(typedHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("HeaderTokenExtractor", func(t *testing.T) {
		middleware := jwt.MiddlewareWithConfig(jwt.MiddlewareConfig{
			Service:   service,
			Extractor: jwt.HeaderTokenExtractor("X-API-Token"),
		})

		handler := middleware(testHandler)
		server := httptest.NewServer(handler)
		defer server.Close()

		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		req.Header.Set("X-API-Token", token)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
