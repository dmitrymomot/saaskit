package i18n

import "net/http"

// LangExtractor extracts preferred language from HTTP requests.
// Returns empty string to delegate default language selection to middleware.
type LangExtractor func(r *http.Request) string
