package core

import "net/http"

// HTTPError represents an HTTP error with status code and translation key.
// The Key field is intended for i18n/l10n - response types can use it
// to look up translated error messages.
type HTTPError struct {
	Code int    // HTTP status code
	Key  string // Translation key (e.g., "not_found", "unauthorized")
}

// Error implements the error interface.
func (e HTTPError) Error() string {
	return e.Key
}

// 4xx Client Errors
var (
	ErrBadRequest                   = HTTPError{Code: http.StatusBadRequest, Key: "bad_request"}
	ErrUnauthorized                 = HTTPError{Code: http.StatusUnauthorized, Key: "unauthorized"}
	ErrPaymentRequired              = HTTPError{Code: http.StatusPaymentRequired, Key: "payment_required"}
	ErrForbidden                    = HTTPError{Code: http.StatusForbidden, Key: "forbidden"}
	ErrNotFound                     = HTTPError{Code: http.StatusNotFound, Key: "not_found"}
	ErrMethodNotAllowed             = HTTPError{Code: http.StatusMethodNotAllowed, Key: "method_not_allowed"}
	ErrNotAcceptable                = HTTPError{Code: http.StatusNotAcceptable, Key: "not_acceptable"}
	ErrProxyAuthRequired            = HTTPError{Code: http.StatusProxyAuthRequired, Key: "proxy_auth_required"}
	ErrRequestTimeout               = HTTPError{Code: http.StatusRequestTimeout, Key: "request_timeout"}
	ErrConflict                     = HTTPError{Code: http.StatusConflict, Key: "conflict"}
	ErrGone                         = HTTPError{Code: http.StatusGone, Key: "gone"}
	ErrLengthRequired               = HTTPError{Code: http.StatusLengthRequired, Key: "length_required"}
	ErrPreconditionFailed           = HTTPError{Code: http.StatusPreconditionFailed, Key: "precondition_failed"}
	ErrRequestEntityTooLarge        = HTTPError{Code: http.StatusRequestEntityTooLarge, Key: "request_entity_too_large"}
	ErrRequestURITooLong            = HTTPError{Code: http.StatusRequestURITooLong, Key: "request_uri_too_long"}
	ErrUnsupportedMediaType         = HTTPError{Code: http.StatusUnsupportedMediaType, Key: "unsupported_media_type"}
	ErrRequestedRangeNotSatisfiable = HTTPError{Code: http.StatusRequestedRangeNotSatisfiable, Key: "requested_range_not_satisfiable"}
	ErrExpectationFailed            = HTTPError{Code: http.StatusExpectationFailed, Key: "expectation_failed"}
	ErrTeapot                       = HTTPError{Code: http.StatusTeapot, Key: "teapot"}
	ErrMisdirectedRequest           = HTTPError{Code: http.StatusMisdirectedRequest, Key: "misdirected_request"}
	ErrUnprocessableEntity          = HTTPError{Code: http.StatusUnprocessableEntity, Key: "unprocessable_entity"}
	ErrLocked                       = HTTPError{Code: http.StatusLocked, Key: "locked"}
	ErrFailedDependency             = HTTPError{Code: http.StatusFailedDependency, Key: "failed_dependency"}
	ErrTooEarly                     = HTTPError{Code: http.StatusTooEarly, Key: "too_early"}
	ErrUpgradeRequired              = HTTPError{Code: http.StatusUpgradeRequired, Key: "upgrade_required"}
	ErrPreconditionRequired         = HTTPError{Code: http.StatusPreconditionRequired, Key: "precondition_required"}
	ErrTooManyRequests              = HTTPError{Code: http.StatusTooManyRequests, Key: "too_many_requests"}
	ErrRequestHeaderFieldsTooLarge  = HTTPError{Code: http.StatusRequestHeaderFieldsTooLarge, Key: "request_header_fields_too_large"}
	ErrUnavailableForLegalReasons   = HTTPError{Code: http.StatusUnavailableForLegalReasons, Key: "unavailable_for_legal_reasons"}
)

// 5xx Server Errors
var (
	ErrInternalServerError           = HTTPError{Code: http.StatusInternalServerError, Key: "internal_server_error"}
	ErrNotImplemented                = HTTPError{Code: http.StatusNotImplemented, Key: "not_implemented"}
	ErrBadGateway                    = HTTPError{Code: http.StatusBadGateway, Key: "bad_gateway"}
	ErrServiceUnavailable            = HTTPError{Code: http.StatusServiceUnavailable, Key: "service_unavailable"}
	ErrGatewayTimeout                = HTTPError{Code: http.StatusGatewayTimeout, Key: "gateway_timeout"}
	ErrHTTPVersionNotSupported       = HTTPError{Code: http.StatusHTTPVersionNotSupported, Key: "http_version_not_supported"}
	ErrVariantAlsoNegotiates         = HTTPError{Code: http.StatusVariantAlsoNegotiates, Key: "variant_also_negotiates"}
	ErrInsufficientStorage           = HTTPError{Code: http.StatusInsufficientStorage, Key: "insufficient_storage"}
	ErrLoopDetected                  = HTTPError{Code: http.StatusLoopDetected, Key: "loop_detected"}
	ErrNotExtended                   = HTTPError{Code: http.StatusNotExtended, Key: "not_extended"}
	ErrNetworkAuthenticationRequired = HTTPError{Code: http.StatusNetworkAuthenticationRequired, Key: "network_authentication_required"}
)

// NewHTTPError creates a custom HTTP error with the given status code and translation key.
//
// Example:
//
//	err := saaskit.NewHTTPError(http.StatusForbidden, "insufficient_permissions")
func NewHTTPError(code int, key string) HTTPError {
	return HTTPError{Code: code, Key: key}
}
