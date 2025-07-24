package i18n

import (
	"context"
)

// localeContextKey is the key for storing locale in context
type localeContextKey struct{}

// SetLocale sets the locale in the context.
func SetLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, localeContextKey{}, locale)
}

// GetLocale returns the locale from the context.
// If no locale is set, will return default locale - "en".
func GetLocale(ctx context.Context) string {
	locale, _ := ctx.Value(localeContextKey{}).(string)
	if locale == "" {
		return DefaultLanguage
	}
	return locale
}
