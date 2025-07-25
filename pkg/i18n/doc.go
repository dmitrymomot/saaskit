// Package i18n provides a simple yet powerful internationalisation (i18n) solution for Go
// applications. It focuses on developer-friendliness, performance and robust error handling
// while remaining thread-safe for concurrent use in production environments.
//
// The package allows you to:
//
//   - Load translations from the local file-system, an embedded file-system, or any custom
//     storage by implementing the TranslationAdapter interface.
//   - Detect the preferred user language via HTTP request inspection with pluggable language
//     extractors as well as helper functions for parsing the Accept-Language header.
//   - Translate strings with variable substitution using named placeholders (`%{key}`) and
//     count-aware pluralisation helpers.
//   - Format durations and "time ago" messages in a locale aware manner.
//   - Expose JSON dumps of the translations for client-side consumption.
//   - Seamlessly integrate with the standard `net/http` stack through middleware that injects
//     a ready-to-use translator into the request context.
//
// # Architecture
//
// At its core the package revolves around the Translator type which delegates storage concerns
// to a TranslationAdapter implementation. Adapters are thin wrappers that return translation
// strings for a given language/key pair and expose the list of supported languages. Ready-made
// adapters for directories, single files and `embed.FS` are included, but you can supply your
// own by fulfilling the interface.
//
// Pluralisation rules and placeholder replacement are implemented in pure Go without any CGO
// dependencies keeping the package lightweight and portable.
//
// # Usage
//
// Basic set-up with a filesystem adapter:
//
//	adapter, err := i18n.NewFileAdapter(i18n.NewYAMLParser(), "./translations")
//	if err != nil {
//		log.Fatalf("failed to create adapter: %v", err)
//	}
//
//	translator, err := i18n.NewTranslator(context.Background(), adapter,
//		i18n.WithDefaultLanguage("en"),
//		i18n.WithFallbackToKey(true),
//	)
//	if err != nil {
//		log.Fatalf("failed to init translator: %v", err)
//	}
//
//	msg := translator.T("en", "welcome", "name", "John")
//	// msg == "Welcome, John!"
//
// # HTTP Middleware
//
// The middleware automatically determines the request language (Accept-Language header by
// default) and stores a Translator bound to that language in the request context:
//
//	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    greeting := translator.Tc(r.Context(), "greeting")
//	    fmt.Fprintln(w, greeting)
//	})
//
//	http.Handle("/", i18n.Middleware(translator, nil)(handler))
//
// # Error Handling
//
// Custom error values such as ErrLanguageNotSupported allow fine-grained error checks, e.g.:
//
//	if errors.Is(err, i18n.ErrLanguageNotSupported) {
//	    // fallback logic
//	}
//
// # See Also
//
// Additional real-world examples can be found in the package README and the example_* test files.
package i18n
