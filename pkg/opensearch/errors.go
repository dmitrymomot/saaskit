package opensearch

import "errors"

var (
	ErrConnectionFailed  = errors.New("opensearch connection failed")
	ErrHealthcheckFailed = errors.New("opensearch healthcheck failed")
)
