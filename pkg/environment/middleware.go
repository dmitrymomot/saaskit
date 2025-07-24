package environment

import "net/http"

// Middleware adds environment to request context
func Middleware(env string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := WithContext(r.Context(), env)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
