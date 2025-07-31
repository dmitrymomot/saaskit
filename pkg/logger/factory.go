package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/dmitrymomot/saaskit/pkg/environment"
)

// Format represents logger output format.
type Format string

const (
	// FormatJSON outputs structured logs for production log aggregation systems.
	FormatJSON Format = "json"
	// FormatText outputs human-readable logs for development debugging.
	FormatText Format = "text"
)

// Option configures logger creation.
type Option func(*config)

func WithLevel(l slog.Level) Option {
	return func(c *config) { c.level = l }
}

// WithFormat sets output format.
// Panics for invalid formats to enforce fail-fast initialization - framework
// misconfiguration should prevent startup rather than cause runtime errors.
func WithFormat(f Format) Option {
	return func(c *config) {
		switch f {
		case FormatJSON, FormatText:
			c.format = f
		default:
			panic(fmt.Errorf("invalid log format %q: must be %q or %q", f, FormatJSON, FormatText))
		}
	}
}

func WithTextFormatter() Option {
	return func(c *config) {
		c.format = FormatText
	}
}

func WithJSONFormatter() Option {
	return func(c *config) {
		c.format = FormatJSON
	}
}

// WithOutput sets custom output destination, ignoring nil writers for safety.
func WithOutput(w io.Writer) Option {
	return func(c *config) {
		if w != nil {
			c.output = w
		}
	}
}

// WithHandlerOptions allows fine-grained control over slog behavior.
// Nil options are ignored to prevent accidental misconfiguration.
func WithHandlerOptions(opts *slog.HandlerOptions) Option {
	return func(c *config) {
		if opts != nil {
			c.handlerOptions = opts
		}
	}
}

// WithAttr adds static attributes to every log record.
// Empty attribute lists are ignored to avoid allocation overhead.
func WithAttr(attrs ...slog.Attr) Option {
	return func(c *config) {
		if len(attrs) > 0 {
			c.attrs = append(c.attrs, attrs...)
		}
	}
}

// WithContextExtractors registers functions that inject dynamic attributes from context.
// Nil extractors are filtered out defensively to prevent runtime panics.
func WithContextExtractors(extractors ...ContextExtractor) Option {
	return func(c *config) {
		for _, ex := range extractors {
			if ex != nil {
				c.extractors = append(c.extractors, ex)
			}
		}
	}
}

// WithContextValue is a convenience wrapper adding a context value extractor.
// Creates a closure that extracts values from context during logging, enabling
// automatic injection of request-scoped data like request IDs.
func WithContextValue(name string, key any) Option {
	return func(c *config) {
		if name == "" || key == nil {
			return
		}
		c.extractors = append(c.extractors, func(ctx context.Context) (slog.Attr, bool) {
			if v := ctx.Value(key); v != nil {
				return slog.Any(name, v), true
			}
			return slog.Attr{}, false
		})
	}
}

// WithDevelopment configures development defaults.
// Uses text format for readability and debug level for detailed diagnostics.
func WithDevelopment(service string) Option {
	return func(c *config) {
		if service == "" {
			return
		}
		c.level = slog.LevelDebug
		c.format = FormatText
		if c.output == nil {
			c.output = os.Stdout
		}
		c.attrs = append(c.attrs,
			slog.String("service", service),
			slog.String("env", string(environment.Development)),
		)
	}
}

// WithProduction configures production defaults.
// Uses JSON format for structured logging and info level to reduce noise.
func WithProduction(service string) Option {
	return func(c *config) {
		if service == "" {
			return
		}
		c.level = slog.LevelInfo
		c.format = FormatJSON
		if c.output == nil {
			c.output = os.Stdout
		}
		c.attrs = append(c.attrs,
			slog.String("service", service),
			slog.String("env", string(environment.Production)),
		)
	}
}

func WithStaging(service string) Option {
	return func(c *config) {
		if service == "" {
			return
		}
		c.level = slog.LevelInfo
		c.format = FormatJSON
		if c.output == nil {
			c.output = os.Stdout
		}
		c.attrs = append(c.attrs,
			slog.String("service", service),
			slog.String("env", string(environment.Staging)),
		)
	}
}

func WithEnvironment(env string, service string) Option {
	return func(c *config) {
		switch env {
		case string(environment.Production), "prod":
			WithProduction(service)(c)
		case string(environment.Staging), "stage":
			WithStaging(service)(c)
		default:
			WithDevelopment(service)(c)
		}
	}
}

func SetAsDefault(l *slog.Logger) {
	slog.SetDefault(l)
}

type config struct {
	level          slog.Level
	format         Format
	output         io.Writer
	attrs          []slog.Attr
	handlerOptions *slog.HandlerOptions
	extractors     []ContextExtractor
}

// defaultConfig provides production-safe defaults: JSON format with INFO level.
// JSON ensures compatibility with log aggregation systems, INFO reduces noise.
func defaultConfig() *config {
	return &config{
		level:  slog.LevelInfo,
		format: FormatJSON,
		output: os.Stdout,
	}
}

// New creates a configured slog.Logger with context injection capabilities.
// Applies options, creates appropriate handler, and wraps with decorator for
// automatic context attribute extraction in the logging hot path.
func New(opts ...Option) *slog.Logger {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	handlerOpts := cfg.handlerOptions
	if handlerOpts == nil {
		handlerOpts = &slog.HandlerOptions{Level: cfg.level}
	}

	var handler slog.Handler
	if cfg.format == FormatText {
		handler = slog.NewTextHandler(cfg.output, handlerOpts)
	} else {
		handler = slog.NewJSONHandler(cfg.output, handlerOpts)
	}

	if len(cfg.attrs) > 0 {
		handler = handler.WithAttrs(cfg.attrs)
	}

	decorated := NewLogHandlerDecorator(handler, cfg.extractors...)
	return slog.New(decorated)
}
