package handler

import (
	"encoding/json"
	"maps"
	"net/http"
)

// JSONResponse is the standard JSON response structure
type JSONResponse struct {
	Data  any            `json:"data,omitempty"`
	Meta  map[string]any `json:"meta,omitempty"`
	Error *ErrorDetail   `json:"error,omitempty"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
	Code    string              `json:"code,omitempty"`
	Message string              `json:"message,omitempty"`
	Details map[string][]string `json:"details,omitempty"`
}

// jsonResponse implements Response for JSON rendering
type jsonResponse struct {
	status int
	body   JSONResponse
}

func (j jsonResponse) Render(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(j.status)
	return json.NewEncoder(w).Encode(j.body)
}

// JSONOption configures JSON response
type JSONOption func(*jsonResponse)

// WithJSONStatus sets custom HTTP status code
func WithJSONStatus(status int) JSONOption {
	return func(r *jsonResponse) {
		r.status = status
	}
}

// WithJSONMeta adds metadata to response
func WithJSONMeta(meta map[string]any) JSONOption {
	return func(r *jsonResponse) {
		r.body.Meta = meta
	}
}

// JSON creates a JSON response with options
func JSON(v any, opts ...JSONOption) Response {
	r := &jsonResponse{
		status: http.StatusOK,
		body:   JSONResponse{},
	}

	// Handle different input types for flexible JSON response creation
	switch val := v.(type) {
	case JSONResponse:
		r.body = val
	case *ErrorDetail:
		r.body.Error = val
		r.status = http.StatusInternalServerError
	case error:
		r.body.Error = errorToDetail(val, &r.status)
	default:
		r.body.Data = v
	}

	// Apply options (can override defaults)
	for _, opt := range opts {
		opt(r)
	}

	return r
}

// JSONError creates a JSON error response from an error with options
func JSONError(err any, opts ...JSONOption) Response {
	r := &jsonResponse{
		status: http.StatusInternalServerError,
		body:   JSONResponse{},
	}

	// Handle different error input types
	switch e := err.(type) {
	case *ErrorDetail:
		r.body.Error = e
	case error:
		r.body.Error = errorToDetail(e, &r.status)
	}

	// Apply options (can override status or add meta)
	for _, opt := range opts {
		opt(r)
	}

	return r
}

// errorToDetail converts error to ErrorDetail and sets appropriate status
func errorToDetail(err error, status *int) *ErrorDetail {
	code := "internal_error"
	message := err.Error()

	// Set default error status if still at OK (200)
	if *status == http.StatusOK {
		*status = http.StatusInternalServerError
	}

	// Check for ValidationError
	if valErr, ok := err.(ValidationError); ok {
		*status = http.StatusUnprocessableEntity
		code = "validation_error"

		detail := &ErrorDetail{
			Code:    code,
			Message: message,
		}

		if len(valErr) > 0 {
			detail.Details = make(map[string][]string)
			maps.Copy(detail.Details, valErr)
		}

		return detail
	}

	// Check for HTTPError
	if httpErr, ok := err.(HTTPError); ok {
		*status = httpErr.Code
		code = httpErr.Key
		message = http.StatusText(httpErr.Code)
	}

	return &ErrorDetail{
		Code:    code,
		Message: message,
	}
}
