package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/dmitrymomot/saaskit/pkg/logger"
	"github.com/dmitrymomot/saaskit/pkg/requestid"
)

// ErrorPageParams contains data for rendering error pages
type ErrorPageParams struct {
	Error      string
	StatusCode int
	RequestID  string
	RetryURL   string
}

// ErrorToastParams contains data for rendering error toasts
type ErrorToastParams struct {
	Message   string
	Type      string // "error", "warning", "info"
	RequestID string
}

// ErrorHandlerConfig configures the default error handler
type ErrorHandlerConfig struct {
	// ErrorPage renders full error page for regular HTTP requests
	ErrorPage func(ErrorPageParams) templ.Component

	// ErrorToast renders toast notification for DataStar requests
	ErrorToast func(ErrorToastParams) templ.Component

	// ToastTarget specifies where to render toast notifications (default: "#toast-container")
	ToastTarget string

	// ToastMode specifies how to render toasts (default: PatchPrepend)
	ToastMode datastar.ElementPatchMode
}

// NewErrorHandler creates the default error handler that adapts to request type.
// For regular HTTP requests, it renders a full error page.
// For DataStar requests, it sends a toast notification.
// Configure this once in main.go and pass to all services.
func NewErrorHandler(log *slog.Logger, cfg ErrorHandlerConfig) ErrorHandler[Context] {
	// Set defaults
	if cfg.ToastTarget == "" {
		cfg.ToastTarget = "#toast-container"
	}
	if cfg.ToastMode == "" {
		cfg.ToastMode = PatchPrepend
	}

	// Default logger if not provided
	if log == nil {
		log = slog.Default()
	}

	return func(ctx Context, err error) {
		requestID := requestid.FromContext(ctx.Request().Context())

		// Determine status code and message
		statusCode := http.StatusInternalServerError
		message := "An error occurred processing your request"
		errorType := "error"

		var httpErr HTTPError
		if errors.As(err, &httpErr) {
			statusCode = httpErr.Code
			message = httpErr.Key

			// Determine error type based on status code
			switch {
			case statusCode >= 400 && statusCode < 500:
				errorType = "warning" // Client errors
			case statusCode >= 500:
				errorType = "error" // Server errors
			}
		}

		// Check for validation errors
		var validationErr ValidationError
		if errors.As(err, &validationErr) {
			statusCode = http.StatusBadRequest
			errorType = "warning"
			// Get first error message from validation errors
			for field, messages := range validationErr {
				if len(messages) > 0 {
					message = fmt.Sprintf("%s: %s", field, messages[0])
					break
				}
			}
		}

		// Log the error with context using logger helpers
		logLevel := slog.LevelError
		if statusCode >= 400 && statusCode < 500 {
			logLevel = slog.LevelWarn
		}

		log.LogAttrs(ctx.Request().Context(), logLevel, "request error",
			logger.RequestID(requestID),
			logger.Error(err),
			slog.Int("status_code", statusCode),
			slog.String("method", ctx.Request().Method),
			slog.String("path", ctx.Request().URL.Path),
			slog.Bool("is_datastar", IsDataStar(ctx.Request())),
			logger.Component("error_handler"),
		)

		// Adapt based on request type
		if IsDataStar(ctx.Request()) {
			// DataStar: Send toast notification
			if cfg.ErrorToast != nil {
				params := ErrorToastParams{
					Message:   message,
					Type:      errorType,
					RequestID: requestID,
				}

				component := cfg.ErrorToast(params)
				response := Templ(
					component,
					WithTarget(cfg.ToastTarget),
					WithPatchMode(cfg.ToastMode),
				)

				// Don't set status code for SSE responses
				if renderErr := response.Render(ctx.ResponseWriter(), ctx.Request()); renderErr != nil {
					log.Error("failed to render error toast",
						logger.RequestID(requestID),
						logger.Error(renderErr),
						logger.Event("render_error_toast"),
					)
				}
			} else {
				log.Warn("no error toast component configured for DataStar request",
					logger.RequestID(requestID),
					logger.Component("error_handler"),
				)
			}
		} else {
			// Regular HTTP: Render full error page
			if cfg.ErrorPage != nil {
				params := ErrorPageParams{
					Error:      message,
					StatusCode: statusCode,
					RequestID:  requestID,
					RetryURL:   ctx.Request().URL.Path,
				}

				component := cfg.ErrorPage(params)

				ctx.ResponseWriter().WriteHeader(statusCode)
				response := Templ(component)

				if renderErr := response.Render(ctx.ResponseWriter(), ctx.Request()); renderErr != nil {
					log.Error("failed to render error page",
						logger.RequestID(requestID),
						logger.Error(renderErr),
						logger.Event("render_error_page"),
					)
					// Fallback to basic error
					http.Error(ctx.ResponseWriter(), "Internal Server Error", http.StatusInternalServerError)
				}
			} else {
				log.Warn("no error page component configured",
					logger.RequestID(requestID),
					logger.Component("error_handler"),
				)
				// Fallback if no error page configured
				http.Error(ctx.ResponseWriter(), message, statusCode)
			}
		}
	}
}
