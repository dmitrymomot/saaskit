package requestid

import (
	"net/http"
	"regexp"

	"github.com/google/uuid"
)

const (
	Header      = "X-Request-ID"
	maxIDLength = 128
	idPattern   = "^[a-zA-Z0-9_-]+$"
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
	if len(id) == 0 || len(id) > maxIDLength {
		return false
	}
	return validIDRegex.MatchString(id)
}
