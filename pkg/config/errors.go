package config

import "errors"

// Package-specific errors
var (
	// ErrParsingConfig is returned when environment variables cannot be parsed into the config struct
	ErrParsingConfig = errors.New("failed to parse environment variables into config")

	// ErrInvalidConfigType is returned when trying to access a config with an invalid type
	ErrInvalidConfigType = errors.New("invalid config type")

	// ErrConfigNotLoaded is returned when attempting to access a config that hasn't been loaded
	ErrConfigNotLoaded = errors.New("configuration has not been loaded")

	// ErrNilPointer is returned when a nil pointer is provided to Load
	ErrNilPointer = errors.New("nil pointer provided to config loader")
)
