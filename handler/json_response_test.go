package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/handler"
)

func TestJSON(t *testing.T) {
	t.Parallel()

	t.Run("simple data", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSON(map[string]string{"id": "123", "name": "test"})
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Data: map[string]any{"id": "123", "name": "test"},
		}, got)
	})

	t.Run("with meta", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSON(
			map[string]string{"id": "123"},
			handler.WithJSONMeta(map[string]any{"version": "1.0"}),
		)
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Data: map[string]any{"id": "123"},
			Meta: map[string]any{"version": "1.0"},
		}, got)
	})

	t.Run("minimal response", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSON(nil)
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{}, got)
	})

	t.Run("with custom status", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSON(
			map[string]string{"id": "456"},
			handler.WithJSONStatus(http.StatusCreated),
		)
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Data: map[string]any{"id": "456"},
		}, got)
	})

	t.Run("multiple options", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSON(
			map[string]string{"id": "789"},
			handler.WithJSONStatus(http.StatusAccepted),
			handler.WithJSONMeta(map[string]any{"page": 1}),
		)
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusAccepted, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Data: map[string]any{"id": "789"},
			Meta: map[string]any{"page": float64(1)},
		}, got)
	})

	t.Run("direct JSONResponse", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSON(handler.JSONResponse{
			Data: map[string]any{"key": "value"},
		})
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Data: map[string]any{"key": "value"},
		}, got)
	})

	t.Run("direct ErrorDetail", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSON(&handler.ErrorDetail{
			Code:    "test_error",
			Message: "Test error message",
		})
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Error: &handler.ErrorDetail{
				Code:    "test_error",
				Message: "Test error message",
			},
		}, got)
	})

	t.Run("error as value", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSON(errors.New("test error"))
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Error: &handler.ErrorDetail{
				Code:    "internal_error",
				Message: "test error",
			},
		}, got)
	})
}

func TestJSONError(t *testing.T) {
	t.Parallel()

	t.Run("standard error", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSONError(errors.New("something went wrong"))
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Error: &handler.ErrorDetail{
				Code:    "internal_error",
				Message: "something went wrong",
			},
		}, got)
	})

	t.Run("http error", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSONError(handler.NewHTTPError(http.StatusBadRequest, "invalid request"))
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Error: &handler.ErrorDetail{
				Code:    "invalid request",
				Message: "Bad Request",
			},
		}, got)
	})

	t.Run("not found error", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSONError(handler.NewHTTPError(http.StatusNotFound, "resource not found"))
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Error: &handler.ErrorDetail{
				Code:    "resource not found",
				Message: "Not Found",
			},
		}, got)
	})

	t.Run("validation error", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		valErr := handler.NewValidationError()
		valErr.Add("email", "invalid format")
		valErr.Add("email", "already exists")
		valErr.Add("age", "must be positive")

		resp := handler.JSONError(valErr)
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)

		// Check message separately due to map iteration order
		assert.Equal(t, "validation_error", got.Error.Code)
		assert.Contains(t, got.Error.Message, "validation error:")
		assert.Contains(t, got.Error.Message, "email: invalid format")
		assert.Contains(t, got.Error.Message, "age: must be positive")
		assert.Equal(t, map[string][]string{
			"email": {"invalid format", "already exists"},
			"age":   {"must be positive"},
		}, got.Error.Details)
	})

	t.Run("empty validation error", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSONError(handler.NewValidationError())
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Error: &handler.ErrorDetail{
				Code:    "validation_error",
				Message: "Validation failed",
			},
		}, got)
	})

	t.Run("predefined not found error", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSONError(handler.ErrNotFound)
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Error: &handler.ErrorDetail{
				Code:    "not_found",
				Message: "Not Found",
			},
		}, got)
	})

	t.Run("predefined unauthorized error", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSONError(handler.ErrUnauthorized)
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Error: &handler.ErrorDetail{
				Code:    "unauthorized",
				Message: "Unauthorized",
			},
		}, got)
	})
}

func TestJSONResponse_OmitEmpty(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	// Create response with no data
	resp := handler.JSON(nil)
	err := resp.Render(w, r)
	require.NoError(t, err)

	// Check that empty fields are omitted
	var result map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, 0, len(result), "should have no fields when all are empty")
	assert.Nil(t, result["data"])
	assert.Nil(t, result["meta"])
	assert.Nil(t, result["error"])
}

func TestJSONErrorWithOptions(t *testing.T) {
	t.Parallel()

	t.Run("with custom status", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSONError(
			errors.New("conflict"),
			handler.WithJSONStatus(http.StatusConflict),
		)
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Error: &handler.ErrorDetail{
				Code:    "internal_error",
				Message: "conflict",
			},
		}, got)
	})

	t.Run("with metadata", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSONError(
			errors.New("error with context"),
			handler.WithJSONMeta(map[string]any{
				"request_id": "123-456",
				"timestamp":  "2024-01-01T00:00:00Z",
			}),
		)
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Error: &handler.ErrorDetail{
				Code:    "internal_error",
				Message: "error with context",
			},
			Meta: map[string]any{
				"request_id": "123-456",
				"timestamp":  "2024-01-01T00:00:00Z",
			},
		}, got)
	})

	t.Run("with custom status and metadata", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		resp := handler.JSONError(
			handler.NewHTTPError(http.StatusForbidden, "access_denied"),
			handler.WithJSONStatus(http.StatusForbidden),
			handler.WithJSONMeta(map[string]any{
				"resource": "/admin",
				"action":   "write",
			}),
		)
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Error: &handler.ErrorDetail{
				Code:    "access_denied",
				Message: "Forbidden",
			},
			Meta: map[string]any{
				"resource": "/admin",
				"action":   "write",
			},
		}, got)
	})

	t.Run("direct ErrorDetail", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		detail := &handler.ErrorDetail{
			Code:    "rate_limit",
			Message: "Too many requests",
			Details: map[string][]string{
				"limit": {"100 per minute"},
			},
		}
		resp := handler.JSONError(detail, handler.WithJSONStatus(http.StatusTooManyRequests))
		err := resp.Render(w, r)
		require.NoError(t, err)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got handler.JSONResponse
		err = json.Unmarshal(w.Body.Bytes(), &got)
		require.NoError(t, err)
		assert.Equal(t, handler.JSONResponse{
			Error: detail,
		}, got)
	})
}
