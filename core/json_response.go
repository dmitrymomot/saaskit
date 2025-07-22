package core

import (
	"encoding/json"
	"maps"
	"net/http"
)

// JSONResponse is the standard JSON response structure
type JSONResponse struct {
	Code    string         `json:"code,omitempty"`
	Message string         `json:"message,omitempty"`
	Data    any            `json:"data,omitempty"`
	Meta    map[string]any `json:"meta,omitempty"`
	Error   *ErrorDetail   `json:"error,omitempty"`
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

// JSON creates a JSON response
func JSON(code string, data any, meta map[string]any) Response {
	return jsonResponse{
		status: http.StatusOK,
		body: JSONResponse{
			Code: code,
			Data: data,
			Meta: meta,
		},
	}
}

// JSONError creates a JSON error response from an error
func JSONError(err error) Response {
	// Default to internal server error
	status := http.StatusInternalServerError
	code := "internal_error"
	errorDetail := &ErrorDetail{
		Code:    code,
		Message: err.Error(),
	}

	// Check if error is ValidationError
	if valErr, ok := err.(ValidationError); ok {
		status = http.StatusUnprocessableEntity
		code = "validation_error"
		errorDetail.Code = code

		// Convert ValidationError to map[string][]string
		if len(valErr) > 0 {
			errorDetail.Details = make(map[string][]string)
			maps.Copy(errorDetail.Details, valErr)
		}
	} else if httpErr, ok := err.(HTTPError); ok {
		// Check if error is HTTPError
		status = httpErr.Code
		code = httpErr.Key
		errorDetail.Code = code
		errorDetail.Message = http.StatusText(httpErr.Code)
	}

	return jsonResponse{
		status: status,
		body: JSONResponse{
			Code:  code,
			Error: errorDetail,
		},
	}
}
