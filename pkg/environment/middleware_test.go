package environment_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/environment"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  environment.Environment
	}{
		{
			name: "development environment",
			env:  environment.Development,
		},
		{
			name: "production environment",
			env:  environment.Production,
		},
		{
			name: "staging environment",
			env:  environment.Staging,
		},
		{
			name: "custom environment",
			env:  environment.Environment("custom"),
		},
		{
			name: "empty environment",
			env:  environment.Environment(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a test handler that checks if environment is in context
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				envFromContext := environment.FromContext(r.Context())
				assert.Equal(t, tt.env, envFromContext)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Wrap with environment middleware
			handler := environment.Middleware(tt.env)(testHandler)

			// Create test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rr := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(rr, req)

			// Verify response
			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "OK", rr.Body.String())
		})
	}
}

func TestMiddleware_ChainOrder(t *testing.T) {
	t.Parallel()

	// Create multiple middleware instances
	devMiddleware := environment.Middleware(environment.Development)
	prodMiddleware := environment.Middleware(environment.Production)

	// Create a test handler that records the environment it sees
	var receivedEnv environment.Environment
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedEnv = environment.FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	t.Run("dev then prod - last one wins", func(t *testing.T) {
		// Chain: dev -> prod -> handler
		handler := devMiddleware(prodMiddleware(testHandler))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// Should receive production environment (last middleware wins)
		assert.Equal(t, environment.Production, receivedEnv)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("prod then dev - last one wins", func(t *testing.T) {
		// Chain: prod -> dev -> handler
		handler := prodMiddleware(devMiddleware(testHandler))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// Should receive development environment (last middleware wins)
		assert.Equal(t, environment.Development, receivedEnv)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
