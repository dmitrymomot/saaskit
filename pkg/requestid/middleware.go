package requestid

import (
	"net/http"
	"regexp"

	"github.com/google/uuid"
)

const (
	Header = "X-Request-ID"
	// Prevents DoS attacks via oversized IDs
	maxIDLength = 128
	// Alphanumeric chars prevent XSS/path traversal
	idPattern = "^[a-zA-Z0-9_-]+$"
)

var validIDRegex = regexp.MustCompile(idPattern)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(Header)
		if requestID == "" {
			requestID = uuid.New().String()
		} else if !isValidRequestID(requestID) {
			requestID = uuid.New().String()
		}
		w.Header().Set(Header, requestID)
		next.ServeHTTP(w, r.WithContext(WithContext(r.Context(), requestID)))
	})
}

func isValidRequestID(id string) bool {
	return len(id) <= maxIDLength && validIDRegex.MatchString(id)
}
