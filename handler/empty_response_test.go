package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/handler"
)

func TestEmpty(t *testing.T) {
	t.Run("returns 204 No Content", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/test", nil)

		resp := handler.Empty()
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())
	})

	t.Run("no content-type header", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/test", nil)

		resp := handler.Empty()
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Empty(t, w.Header().Get("Content-Type"))
	})
}

func TestEmptyWithStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{
			name:   "201 Created",
			status: http.StatusCreated,
		},
		{
			name:   "202 Accepted",
			status: http.StatusAccepted,
		},
		{
			name:   "200 OK",
			status: http.StatusOK,
		},
		{
			name:   "205 Reset Content",
			status: http.StatusResetContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/test", nil)

			resp := handler.EmptyWithStatus(tt.status)
			err := resp.Render(w, r)

			require.NoError(t, err)
			assert.Equal(t, tt.status, w.Code)
			assert.Empty(t, w.Body.String())
		})
	}
}

func TestEmpty_WithDataStarRequest(t *testing.T) {
	t.Run("Empty response with DataStar request", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/test", nil)
		r.Header.Set("datastar-request", "true")

		resp := handler.Empty()
		err := resp.Render(w, r)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())
	})
}

func TestEmpty_Integration(t *testing.T) {
	type deleteRequest struct {
		ID string `path:"id"`
	}

	t.Run("DELETE endpoint with Empty response", func(t *testing.T) {
		h := handler.HandlerFunc[handler.Context, deleteRequest](
			func(ctx handler.Context, req deleteRequest) handler.Response {
				// Simulate successful deletion
				return handler.Empty()
			},
		)

		httpHandler := handler.Wrap(h)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/items/123", nil)

		httpHandler(w, r)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())
	})

	t.Run("POST endpoint with EmptyWithStatus Created", func(t *testing.T) {
		type createRequest struct {
			Name string `json:"name"`
		}

		h := handler.HandlerFunc[handler.Context, createRequest](
			func(ctx handler.Context, req createRequest) handler.Response {
				// Simulate resource creation without returning data
				return handler.EmptyWithStatus(http.StatusCreated)
			},
		)

		httpHandler := handler.Wrap(h)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/items", nil)

		httpHandler(w, r)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Empty(t, w.Body.String())
	})
}
