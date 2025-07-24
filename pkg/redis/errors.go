package redis

import "errors"

var (
	ErrFailedToParseRedisConnString = errors.New("failed to parse redis connection string")
	ErrRedisNotReady                = errors.New("redis did not become ready within the given time period")
	ErrEmptyConnectionURL           = errors.New("empty redis connection URL")
	ErrHealthcheckFailed            = errors.New("redis healthcheck failed")
)
