package handler

import "net/http"

// emptyResponse represents an empty HTTP response with only a status code
type emptyResponse struct {
	status int
}

// Render writes the status code without any body content
func (e emptyResponse) Render(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(e.status)
	return nil
}

// Empty creates an empty response with status 204 (No Content).
// This is useful for successful operations that don't return data,
// such as DELETE endpoints or successful updates where no data needs to be returned.
//
// Example:
//
//	handler := handler.HandlerFunc[handler.Context, DeleteRequest](
//		func(ctx handler.Context, req DeleteRequest) handler.Response {
//			deleteResource(req.ID)
//			return handler.Empty()
//		},
//	)
func Empty() Response {
	return emptyResponse{
		status: http.StatusNoContent,
	}
}

// EmptyWithStatus creates an empty response with a custom status code.
// This allows returning any status code without a response body.
//
// Example:
//
//	// Return 201 Created without body
//	return handler.EmptyWithStatus(http.StatusCreated)
//
//	// Return 202 Accepted for async operations
//	return handler.EmptyWithStatus(http.StatusAccepted)
func EmptyWithStatus(status int) Response {
	return emptyResponse{
		status: status,
	}
}
