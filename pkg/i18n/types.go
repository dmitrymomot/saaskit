package i18n

import "net/http"

// LangExtractor is a function type that extracts language information from an HTTP request.
// It takes an *http.Request as input and returns a string representing the language code.
// This is typically used to determine the user's preferred language for localization.
type LangExtractor func(r *http.Request) string
