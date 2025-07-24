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

	tests := []struct {
		name     string
		code     string
		data     any
		meta     map[string]any
		expected handler.JSONResponse
	}{
		{
			name: "success with data",
			code: "OK",
			data: map[string]string{"id": "123", "name": "test"},
			meta: map[string]any{"version": "1.0"},
			expected: handler.JSONResponse{
				Code: "OK",
				Data: map[string]any{"id": "123", "name": "test"},
				Meta: map[string]any{"version": "1.0"},
			},
		},
		{
			name: "minimal response",
			code: "CREATED",
			data: nil,
			meta: nil,
			expected: handler.JSONResponse{
				Code: "CREATED",
			},
		},
		{
			name: "with meta only",
			code: "OK",
			data: nil,
			meta: map[string]any{"request_id": "abc123"},
			expected: handler.JSONResponse{
				Code: "OK",
				Meta: map[string]any{"request_id": "abc123"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			resp := handler.JSON(tt.code, tt.data, tt.meta)
			err := resp.Render(w, r)
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var got handler.JSONResponse
			err = json.Unmarshal(w.Body.Bytes(), &got)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestJSONError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		err          error
		expectedCode int
		expectedBody handler.JSONResponse
	}{
		{
			name:         "standard error",
			err:          errors.New("something went wrong"),
			expectedCode: http.StatusInternalServerError,
			expectedBody: handler.JSONResponse{
				Code: "internal_error",
				Error: &handler.ErrorDetail{
					Code:    "internal_error",
					Message: "something went wrong",
				},
			},
		},
		{
			name:         "http error",
			err:          handler.NewHTTPError(http.StatusBadRequest, "invalid request"),
			expectedCode: http.StatusBadRequest,
			expectedBody: handler.JSONResponse{
				Code: "invalid request",
				Error: &handler.ErrorDetail{
					Code:    "invalid request",
					Message: "Bad Request",
				},
			},
		},
		{
			name:         "not found error",
			err:          handler.NewHTTPError(http.StatusNotFound, "resource not found"),
			expectedCode: http.StatusNotFound,
			expectedBody: handler.JSONResponse{
				Code: "resource not found",
				Error: &handler.ErrorDetail{
					Code:    "resource not found",
					Message: "Not Found",
				},
			},
		},
		{
			name: "validation error",
			err: func() error {
				err := handler.NewValidationError()
				err.Add("email", "invalid format")
				err.Add("email", "already exists")
				err.Add("age", "must be positive")
				return err
			}(),
			expectedCode: http.StatusUnprocessableEntity,
			expectedBody: handler.JSONResponse{
				Code: "validation_error",
				Error: &handler.ErrorDetail{
					Code:    "validation_error",
					Message: "validation error: email: invalid format, age: must be positive",
					Details: map[string][]string{
						"email": {"invalid format", "already exists"},
						"age":   {"must be positive"},
					},
				},
			},
		},
		{
			name:         "empty validation error",
			err:          handler.NewValidationError(),
			expectedCode: http.StatusUnprocessableEntity,
			expectedBody: handler.JSONResponse{
				Code: "validation_error",
				Error: &handler.ErrorDetail{
					Code:    "validation_error",
					Message: "Validation failed",
				},
			},
		},
		{
			name:         "predefined not found error",
			err:          handler.ErrNotFound,
			expectedCode: http.StatusNotFound,
			expectedBody: handler.JSONResponse{
				Code: "not_found",
				Error: &handler.ErrorDetail{
					Code:    "not_found",
					Message: "Not Found",
				},
			},
		},
		{
			name:         "predefined unauthorized error",
			err:          handler.ErrUnauthorized,
			expectedCode: http.StatusUnauthorized,
			expectedBody: handler.JSONResponse{
				Code: "unauthorized",
				Error: &handler.ErrorDetail{
					Code:    "unauthorized",
					Message: "Unauthorized",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			resp := handler.JSONError(tt.err)
			err := resp.Render(w, r)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedCode, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var got handler.JSONResponse
			err = json.Unmarshal(w.Body.Bytes(), &got)
			require.NoError(t, err)
			// For validation errors, check message separately due to map iteration order
			if tt.name == "validation error" {
				assert.Equal(t, tt.expectedBody.Code, got.Code)
				assert.Equal(t, tt.expectedBody.Error.Code, got.Error.Code)
				assert.Contains(t, got.Error.Message, "validation error:")
				assert.Contains(t, got.Error.Message, "email: invalid format")
				assert.Contains(t, got.Error.Message, "age: must be positive")
				assert.Equal(t, tt.expectedBody.Error.Details, got.Error.Details)
			} else {
				assert.Equal(t, tt.expectedBody, got)
			}
		})
	}
}

func TestJSONResponse_OmitEmpty(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	// Create response with only code
	resp := handler.JSON("OK", nil, nil)
	err := resp.Render(w, r)
	require.NoError(t, err)

	// Check that empty fields are omitted
	var result map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, 1, len(result), "should only have 'code' field")
	assert.Equal(t, "OK", result["code"])
	assert.Nil(t, result["data"])
	assert.Nil(t, result["meta"])
	assert.Nil(t, result["error"])
}
