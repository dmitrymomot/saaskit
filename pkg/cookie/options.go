package cookie

import "net/http"

type Options struct {
	Path     string
	Domain   string
	MaxAge   int
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

type Option func(*Options)

func WithPath(path string) Option {
	return func(o *Options) {
		o.Path = path
	}
}

func WithDomain(domain string) Option {
	return func(o *Options) {
		o.Domain = domain
	}
}

func WithMaxAge(seconds int) Option {
	return func(o *Options) {
		o.MaxAge = seconds
	}
}

func WithSecure(secure bool) Option {
	return func(o *Options) {
		o.Secure = secure
	}
}

func WithHTTPOnly(httpOnly bool) Option {
	return func(o *Options) {
		o.HttpOnly = httpOnly
	}
}

func WithSameSite(sameSite http.SameSite) Option {
	return func(o *Options) {
		o.SameSite = sameSite
	}
}

// applyOptions creates a new Options struct by copying the base options
// and applying the provided option functions. The base options are not modified.
func applyOptions(base Options, opts []Option) Options {
	// Explicit struct copy ensures base options are immutable
	result := Options{
		Path:     base.Path,
		Domain:   base.Domain,
		MaxAge:   base.MaxAge,
		Secure:   base.Secure,
		HttpOnly: base.HttpOnly,
		SameSite: base.SameSite,
	}

	for _, opt := range opts {
		opt(&result)
	}

	return result
}
