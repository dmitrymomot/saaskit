package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

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

// ErrorInfo contains classified error information
type ErrorInfo struct {
	StatusCode int
	Message    string
	Type       string
	LogLevel   slog.Level
}

// Helper functions for HTTP status code classification
func isClientError(statusCode int) bool {
	return statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError
}

func isServerError(statusCode int) bool {
	return statusCode >= http.StatusInternalServerError
}

// determineErrorType maps HTTP status codes to error types for UI display
func determineErrorType(statusCode int) string {
	switch {
	case isClientError(statusCode):
		return "warning"
	case isServerError(statusCode):
		return "error"
	default:
		return "info"
	}
}

// determineLogLevel maps HTTP status codes to appropriate log levels
func determineLogLevel(statusCode int) slog.Level {
	if isClientError(statusCode) {
		return slog.LevelWarn
	}
	return slog.LevelError
}

// setConfigDefaults applies default values to ErrorHandlerConfig
func setConfigDefaults(cfg ErrorHandlerConfig) ErrorHandlerConfig {
	if cfg.ToastTarget == "" {
		cfg.ToastTarget = "#toast-container"
	}
	if cfg.ToastMode == "" {
		cfg.ToastMode = PatchPrepend
	}
	return cfg
}

// formatValidationErrors creates a comprehensive message from validation errors
func formatValidationErrors(validationErr ValidationError) string {
	var messages []string
	for field, fieldMessages := range validationErr {
		for _, msg := range fieldMessages {
			messages = append(messages, fmt.Sprintf("%s: %s", field, msg))
		}
	}
	if len(messages) == 0 {
		return "Validation failed"
	}
	return strings.Join(messages, "; ")
}

// classifyError analyzes the error and returns structured error information
func classifyError(err error) ErrorInfo {
	info := ErrorInfo{
		StatusCode: http.StatusInternalServerError,
		Message:    "An error occurred processing your request",
	}

	// Check for HTTP errors first
	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		info.StatusCode = httpErr.Code
		info.Message = httpErr.Key
	}

	// Check for validation errors (overrides HTTP error if both exist)
	var validationErr ValidationError
	if errors.As(err, &validationErr) {
		info.StatusCode = http.StatusBadRequest
		info.Message = formatValidationErrors(validationErr)
	}

	// Set error type and log level based on final status code
	info.Type = determineErrorType(info.StatusCode)
	info.LogLevel = determineLogLevel(info.StatusCode)

	return info
}

// logError logs the error with comprehensive context
func logError(log *slog.Logger, ctx Context, err error, info ErrorInfo) {
	requestID := requestid.FromContext(ctx.Request().Context())

	log.LogAttrs(ctx.Request().Context(), info.LogLevel, "request error",
		logger.RequestID(requestID),
		logger.Error(err),
		slog.Int("status_code", info.StatusCode),
		slog.String("method", ctx.Request().Method),
		slog.String("path", ctx.Request().URL.Path),
		slog.Bool("is_datastar", IsDataStar(ctx.Request())),
		logger.Component("error_handler"),
	)
}

// renderDataStarResponse renders error as DataStar toast notification
func renderDataStarResponse(ctx Context, cfg ErrorHandlerConfig, info ErrorInfo, requestID string, log *slog.Logger) {
	if cfg.ErrorToast == nil {
		log.Warn("no error toast component configured for DataStar request",
			logger.RequestID(requestID),
			logger.Component("error_handler"),
		)
		return
	}

	params := ErrorToastParams{
		Message:   info.Message,
		Type:      info.Type,
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
}

// renderHTTPResponse renders error as full HTTP error page
func renderHTTPResponse(ctx Context, cfg ErrorHandlerConfig, info ErrorInfo, requestID string, log *slog.Logger) {
	if cfg.ErrorPage == nil {
		log.Warn("no error page component configured",
			logger.RequestID(requestID),
			logger.Component("error_handler"),
		)
		// Fallback if no error page configured
		http.Error(ctx.ResponseWriter(), info.Message, info.StatusCode)
		return
	}

	params := ErrorPageParams{
		Error:      info.Message,
		StatusCode: info.StatusCode,
		RequestID:  requestID,
		RetryURL:   ctx.Request().URL.Path,
	}

	component := cfg.ErrorPage(params)

	ctx.ResponseWriter().WriteHeader(info.StatusCode)
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
}

// NewErrorHandler creates the default error handler that adapts to request type.
// For regular HTTP requests, it renders a full error page.
// For DataStar requests, it sends a toast notification.
// Configure this once in main.go and pass to all services.
func NewErrorHandler(log *slog.Logger, cfg ErrorHandlerConfig) ErrorHandler[Context] {
	cfg = setConfigDefaults(cfg)

	// Default logger if not provided
	if log == nil {
		log = slog.Default()
	}

	return func(ctx Context, err error) {
		requestID := requestid.FromContext(ctx.Request().Context())
		info := classifyError(err)
		logError(log, ctx, err, info)

		// Adapt response based on request type
		if IsDataStar(ctx.Request()) {
			renderDataStarResponse(ctx, cfg, info, requestID, log)
		} else {
			renderHTTPResponse(ctx, cfg, info, requestID, log)
		}
	}
}
