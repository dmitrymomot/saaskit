package handler_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/dmitrymomot/saaskit/handler"
)

// Mock templ components for testing
func mockErrorPage(params handler.ErrorPageParams) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := w.Write([]byte("Error: " + params.Error))
		return err
	})
}

func mockErrorToast(params handler.ErrorToastParams) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := w.Write([]byte("Toast: " + params.Message))
		return err
	})
}

func TestNewErrorHandler_HTTPRequest_GenericError(t *testing.T) {
	log := slog.Default()

	cfg := handler.ErrorHandlerConfig{
		ErrorPage: mockErrorPage,
	}

	errorHandler := handler.NewErrorHandler(log, cfg)

	// Create test request and response
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	ctx := handler.NewContext(w, req)

	// Test with generic error
	err := errors.New("something went wrong")

	errorHandler(ctx, err)

	// Check response - should be 500 for generic errors
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	if !strings.Contains(w.Body.String(), "An error occurred processing your request") {
		t.Errorf("Expected body to contain generic error message, got %s", w.Body.String())
	}
}

func TestNewErrorHandler_HTTPRequest_HTTPError(t *testing.T) {
	log := slog.Default()

	cfg := handler.ErrorHandlerConfig{
		ErrorPage: mockErrorPage,
	}

	errorHandler := handler.NewErrorHandler(log, cfg)

	// Create test request and response
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	ctx := handler.NewContext(w, req)

	// Test with HTTP error
	httpErr := handler.HTTPError{
		Code: http.StatusNotFound,
		Key:  "page.not_found",
	}

	errorHandler(ctx, httpErr)

	// Check response
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	if !strings.Contains(w.Body.String(), "page.not_found") {
		t.Errorf("Expected body to contain 'page.not_found', got %s", w.Body.String())
	}
}

func TestNewErrorHandler_HTTPRequest_ValidationError(t *testing.T) {
	log := slog.Default()

	cfg := handler.ErrorHandlerConfig{
		ErrorPage: mockErrorPage,
	}

	errorHandler := handler.NewErrorHandler(log, cfg)

	// Create test request and response
	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()

	ctx := handler.NewContext(w, req)

	// Test with validation error (single field)
	valErr := handler.ValidationError{
		"email": {"is required"},
	}

	errorHandler(ctx, valErr)

	// Check response - validation errors should be 400
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "email: is required") {
		t.Errorf("Expected body to contain validation error, got %s", w.Body.String())
	}
}

func TestNewErrorHandler_HTTPRequest_MultipleValidationErrors(t *testing.T) {
	log := slog.Default()

	cfg := handler.ErrorHandlerConfig{
		ErrorPage: mockErrorPage,
	}

	errorHandler := handler.NewErrorHandler(log, cfg)

	// Create test request and response
	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()

	ctx := handler.NewContext(w, req)

	// Test with validation error (multiple fields and messages)
	valErr := handler.ValidationError{
		"email":    {"is required", "must be valid email"},
		"password": {"too short", "must contain special characters"},
	}

	errorHandler(ctx, valErr)

	// Check response - validation errors should be 400
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	body := w.Body.String()
	// Should contain all validation errors
	expectedErrors := []string{
		"email: is required",
		"email: must be valid email",
		"password: too short",
		"password: must contain special characters",
	}

	for _, expected := range expectedErrors {
		if !strings.Contains(body, expected) {
			t.Errorf("Expected body to contain '%s', got %s", expected, body)
		}
	}
}

func TestNewErrorHandler_DataStarRequest_ValidationError(t *testing.T) {
	log := slog.Default()

	cfg := handler.ErrorHandlerConfig{
		ErrorToast: mockErrorToast,
	}

	errorHandler := handler.NewErrorHandler(log, cfg)

	// Create DataStar request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "text/event-stream") // DataStar header
	w := httptest.NewRecorder()

	ctx := handler.NewContext(w, req)

	// Test with validation error
	valErr := handler.ValidationError{
		"email": {"is required"},
	}

	errorHandler(ctx, valErr)

	// DataStar responses don't set status codes
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for DataStar, got %d", http.StatusOK, w.Code)
	}

	if !strings.Contains(w.Body.String(), "email: is required") {
		t.Errorf("Expected body to contain validation error, got %s", w.Body.String())
	}
}

func TestNewErrorHandler_DataStarRequest_HTTPError(t *testing.T) {
	log := slog.Default()

	cfg := handler.ErrorHandlerConfig{
		ErrorToast: mockErrorToast,
	}

	errorHandler := handler.NewErrorHandler(log, cfg)

	// Create DataStar request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "text/event-stream") // DataStar header
	w := httptest.NewRecorder()

	ctx := handler.NewContext(w, req)

	// Test with HTTP error (server error)
	httpErr := handler.HTTPError{
		Code: http.StatusInternalServerError,
		Key:  "server.error",
	}

	errorHandler(ctx, httpErr)

	// DataStar responses don't set status codes
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for DataStar, got %d", http.StatusOK, w.Code)
	}

	if !strings.Contains(w.Body.String(), "server.error") {
		t.Errorf("Expected body to contain 'server.error', got %s", w.Body.String())
	}
}

func TestNewErrorHandler_NoComponentsConfigured(t *testing.T) {
	log := slog.Default()
	cfg := handler.ErrorHandlerConfig{} // No components configured

	errorHandler := handler.NewErrorHandler(log, cfg)

	// Test HTTP request without error page component
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ctx := handler.NewContext(w, req)

	err := errors.New("test error")
	errorHandler(ctx, err)

	// Should fallback to http.Error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	// Test DataStar request without toast component
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("Accept", "text/event-stream") // DataStar header
	w2 := httptest.NewRecorder()
	ctx2 := handler.NewContext(w2, req2)

	errorHandler(ctx2, err)

	// DataStar with no component should not set status code
	if w2.Code != http.StatusOK {
		t.Errorf("Expected status %d for DataStar without component, got %d", http.StatusOK, w2.Code)
	}
}

func TestNewErrorHandler_ConfigDefaults(t *testing.T) {
	log := slog.Default()

	// Test that defaults are applied when not specified
	cfg := handler.ErrorHandlerConfig{
		ErrorToast: mockErrorToast,
		// Don't specify ToastTarget or ToastMode - should get defaults
	}

	errorHandler := handler.NewErrorHandler(log, cfg)

	// Create DataStar request
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Accept", "text/event-stream")
	w := httptest.NewRecorder()

	ctx := handler.NewContext(w, req)

	// Test with error
	err := errors.New("test error")
	errorHandler(ctx, err)

	// Should render without errors (proves defaults were set)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestNewErrorHandler_CustomToastConfig(t *testing.T) {
	log := slog.Default()

	// Test custom toast configuration
	cfg := handler.ErrorHandlerConfig{
		ErrorToast:  mockErrorToast,
		ToastTarget: "#custom-toast-container",
		ToastMode:   handler.PatchAppend,
	}

	errorHandler := handler.NewErrorHandler(log, cfg)

	// Create DataStar request
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Accept", "text/event-stream")
	w := httptest.NewRecorder()

	ctx := handler.NewContext(w, req)

	// Test with error
	err := errors.New("custom config test")
	errorHandler(ctx, err)

	// Should render without errors (proves custom config works)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestNewErrorHandler_StatusCodeClassification(t *testing.T) {
	log := slog.Default()

	cfg := handler.ErrorHandlerConfig{
		ErrorPage: mockErrorPage,
	}

	errorHandler := handler.NewErrorHandler(log, cfg)

	tests := []struct {
		name       string
		error      error
		expectCode int
	}{
		{
			name: "client error - 400",
			error: handler.HTTPError{
				Code: http.StatusBadRequest,
				Key:  "bad.request",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "client error - 401",
			error: handler.HTTPError{
				Code: http.StatusUnauthorized,
				Key:  "unauthorized",
			},
			expectCode: http.StatusUnauthorized,
		},
		{
			name: "client error - 404",
			error: handler.HTTPError{
				Code: http.StatusNotFound,
				Key:  "not.found",
			},
			expectCode: http.StatusNotFound,
		},
		{
			name: "server error - 500",
			error: handler.HTTPError{
				Code: http.StatusInternalServerError,
				Key:  "server.error",
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name: "server error - 502",
			error: handler.HTTPError{
				Code: http.StatusBadGateway,
				Key:  "bad.gateway",
			},
			expectCode: http.StatusBadGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			ctx := handler.NewContext(w, req)

			errorHandler(ctx, tt.error)

			if w.Code != tt.expectCode {
				t.Errorf("Expected status %d, got %d", tt.expectCode, w.Code)
			}
		})
	}
}
