package useragent

import "errors"

var (
	ErrEmptyUserAgent     = errors.New("empty user agent string")
	ErrMalformedUserAgent = errors.New("malformed user agent string")
	ErrUnsupportedBrowser = errors.New("unsupported browser")
	ErrUnsupportedOS      = errors.New("unsupported operating system")
	ErrUnknownDevice      = errors.New("unknown device type")
	ErrParsingFailed      = errors.New("failed to parse user agent")
)
