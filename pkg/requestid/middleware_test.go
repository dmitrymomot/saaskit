package requestid_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/requestid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()
	t.Run("generates new request ID when not provided", func(t *testing.T) {
		t.Parallel()
		handler := requestid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := requestid.FromContext(r.Context())
			assert.NotEmpty(t, id)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.NotEmpty(t, rec.Header().Get(requestid.Header))
	})

	t.Run("uses existing request ID from header", func(t *testing.T) {
		t.Parallel()
		existingID := "test-request-id-123"
		handler := requestid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := requestid.FromContext(r.Context())
			assert.Equal(t, existingID, id)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(requestid.Header, existingID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, existingID, rec.Header().Get(requestid.Header))
	})

	t.Run("generates new ID for invalid request ID", func(t *testing.T) {
		t.Parallel()
		invalidIDs := []string{
			"",                              // empty
			"test@request#id",               // invalid characters
			"test request id",               // spaces
			"test/request/id",               // slashes
			"test\\request\\id",             // backslashes
			"test<script>alert(1)</script>", // XSS attempt
			"a-very-long-request-id-that-exceeds-the-maximum-allowed-length-of-128-characters-which-should-be-rejected-and-replaced-with-a-new-uuid",
		}

		for _, invalidID := range invalidIDs {
			t.Run(invalidID, func(t *testing.T) {
				t.Parallel()
				handler := requestid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					id := requestid.FromContext(r.Context())
					assert.NotEmpty(t, id)
					assert.NotEqual(t, invalidID, id)
					w.WriteHeader(http.StatusOK)
				}))

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				if invalidID != "" {
					req.Header.Set(requestid.Header, invalidID)
				}
				rec := httptest.NewRecorder()

				handler.ServeHTTP(rec, req)

				require.Equal(t, http.StatusOK, rec.Code)
				responseID := rec.Header().Get(requestid.Header)
				assert.NotEmpty(t, responseID)
				assert.NotEqual(t, invalidID, responseID)
			})
		}
	})

	t.Run("accepts valid request IDs", func(t *testing.T) {
		t.Parallel()
		validIDs := []string{
			"abc123",
			"test-request-id",
			"test_request_id",
			"ABC-123_xyz",
			"550e8400-e29b-41d4-a716-446655440000", // UUID
		}

		for _, validID := range validIDs {
			t.Run(validID, func(t *testing.T) {
				t.Parallel()
				handler := requestid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					id := requestid.FromContext(r.Context())
					assert.Equal(t, validID, id)
					w.WriteHeader(http.StatusOK)
				}))

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set(requestid.Header, validID)
				rec := httptest.NewRecorder()

				handler.ServeHTTP(rec, req)

				require.Equal(t, http.StatusOK, rec.Code)
				assert.Equal(t, validID, rec.Header().Get(requestid.Header))
			})
		}
	})
}

func TestContext(t *testing.T) {
	t.Parallel()
	t.Run("stores and retrieves request ID", func(t *testing.T) {
		t.Parallel()
		ctx := requestid.WithContext(context.Background(), "test-id")
		id := requestid.FromContext(ctx)
		assert.Equal(t, "test-id", id)
	})

	t.Run("returns empty string when no request ID in context", func(t *testing.T) {
		t.Parallel()
		id := requestid.FromContext(context.Background())
		assert.Empty(t, id)
	})
}
