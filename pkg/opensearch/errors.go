package opensearch

import "errors"

var (
	// ErrConnectionFailed indicates the OpenSearch client could not be created
	// due to configuration or network issues. Use errors.Is() to check.
	ErrConnectionFailed = errors.New("opensearch connection failed")

	// ErrHealthcheckFailed indicates the cluster is unreachable or unhealthy.
	// Returned by both New() during initialization and Healthcheck() during monitoring.
	ErrHealthcheckFailed = errors.New("opensearch healthcheck failed")
)
