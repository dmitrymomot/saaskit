package fingerprint

import "net/http"

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fp := Generate(r)
		ctx := SetFingerprintToContext(r.Context(), fp)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
