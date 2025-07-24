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
	// FormatJSON outputs logs as JSON.
	FormatJSON Format = "json"
	// FormatText outputs human readable text.
	FormatText Format = "text"
)

// Option configures logger creation.
type Option func(*config)

// WithLevel sets logger level.
func WithLevel(l slog.Level) Option {
	return func(c *config) { c.level = l }
}

// WithFormat sets output format.
func WithFormat(f Format) Option {
	return func(c *config) {
		switch f {
		case FormatJSON, FormatText:
			c.format = f
		default:
			panic(fmt.Errorf("invalid log format: %s", f))
		}
	}
}

// WithTextFormatter sets the output format to text.
func WithTextFormatter() Option {
	return func(c *config) {
		c.format = FormatText
	}
}

// WithJSONFormatter sets the output format to JSON.
func WithJSONFormatter() Option {
	return func(c *config) {
		c.format = FormatJSON
	}
}

// WithOutput sets the writer for log output.
func WithOutput(w io.Writer) Option {
	return func(c *config) {
		if w != nil {
			c.output = w
		}
	}
}

// WithHandlerOptions sets slog.HandlerOptions.
func WithHandlerOptions(opts *slog.HandlerOptions) Option {
	return func(c *config) {
		if opts != nil {
			c.handlerOptions = opts
		}
	}
}

// WithAttr adds default attributes to every log entry.
func WithAttr(attrs ...slog.Attr) Option {
	return func(c *config) {
		if len(attrs) > 0 {
			c.attrs = append(c.attrs, attrs...)
		}
	}
}

// WithContextExtractors registers extractors injecting attributes from context.
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

// WithStaging configures staging defaults.
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

// WithEnvironment configures logger based on environment.
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

// SetAsDefault sets logger as the default slog logger.
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

func defaultConfig() *config {
	return &config{
		level:  slog.LevelInfo,
		format: FormatJSON,
		output: os.Stdout,
	}
}

// New creates a slog.Logger configured by the provided options.
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
