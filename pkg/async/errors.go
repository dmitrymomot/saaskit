package async

import "errors"

var (
	ErrTimeout   = errors.New("async: operation timed out waiting for future completion")
	ErrNoFutures = errors.New("async: WaitAny called with empty futures slice")
)
