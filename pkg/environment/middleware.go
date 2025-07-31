package environment

import "net/http"

// Middleware returns a middleware that attaches the given environment to all
// request contexts. This enables environment-aware behavior throughout the
// request handling pipeline without requiring explicit parameter passing.
func Middleware(env Environment) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := WithContext(r.Context(), env)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
