package mongo

import "errors"

var (
	ErrFailedToConnectToMongo = errors.New("failed to connect to mongo")
	ErrHealthcheckFailed      = errors.New("mongo healthcheck failed")
)
