package i18n

import (
	"net/http"
)

// Middleware returns an HTTP middleware that determines the client's preferred language
// and stores it in the request context.
//
// Parameters:
//   - extr: An optional langExtractor function that can extract language from the request
//
// The middleware attempts to determine the language using the provided extractor function.
// If no extractor is provided, it uses a default extractor (DefaultLangExtractor).
// If the extractor returns an empty string, the middleware falls back to "en".
//
// The determined language is stored in the request context using SetLocale and
// can be retrieved later with GetLocale.
func Middleware(extr LangExtractor) func(http.Handler) http.Handler {
	// Use default extractor if none provided
	if extr == nil {
		extr = DefaultLangExtractor()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lang := extr(r)

			// Fallback to "en" if extractor returns empty string
			if lang == "" {
				lang = DefaultLanguage
			}

			next.ServeHTTP(w, r.WithContext(SetLocale(r.Context(), lang)))
		})
	}
}
