package i18n

import (
	"net/http"
)

// Middleware determines user language preference and injects it into request context.
// Uses extractor to detect language with fallback to DefaultLanguage for graceful degradation.
// This pattern ensures every request has a valid language context for translation functions.
func Middleware(extr LangExtractor) func(http.Handler) http.Handler {
	if extr == nil {
		extr = DefaultLangExtractor()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lang := extr(r)

			if lang == "" {
				lang = DefaultLanguage
			}

			next.ServeHTTP(w, r.WithContext(SetLocale(r.Context(), lang)))
		})
	}
}
