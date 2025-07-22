package i18n

import (
	"io"
	"log/slog"
)

// Option is a function that configures a Translator instance.
type Option func(*Translator)

// WithDefaultLanguage sets the default language for the translator.
// This language is used when no language is explicitly specified,
// or when the requested language is not available.
func WithDefaultLanguage(lang string) Option {
	return func(t *Translator) {
		if lang != "" {
			t.defaultLang = lang
		}
	}
}

// WithFallbackToKey determines whether to fall back to the key
// when a translation is not found. Default is true.
func WithFallbackToKey(fallback bool) Option {
	return func(t *Translator) {
		t.fallbackToKey = fallback
	}
}

// WithLogger provides a customizable logger for the translator.
// If not specified, a discard logger is used.
func WithLogger(logger *slog.Logger) Option {
	return func(t *Translator) {
		if logger != nil {
			t.logger = logger
		}
	}
}

// WithMissingTranslationsLogging controls whether missing translations
// are logged. Default is false to avoid excessive logging.
func WithMissingTranslationsLogging(log bool) Option {
	return func(t *Translator) {
		t.missingLogMode = log
	}
}

// WithNoLogging is a convenience option that disables all logging.
func WithNoLogging() Option {
	return func(t *Translator) {
		t.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
		t.missingLogMode = false
	}
}
