package clientip

import "net/http"

// Middleware creates HTTP middleware that extracts and stores client IP in context
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := GetIP(r)
		ctx := SetIPToContext(r.Context(), ip)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
