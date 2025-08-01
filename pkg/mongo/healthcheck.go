package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Healthcheck returns a health check function suitable for Kubernetes readiness/liveness probes
// or HTTP health endpoints.
//
// The returned function performs a lightweight Ping operation to verify MongoDB connectivity
// without impacting database performance. This is essential for container orchestration
// where failed health checks trigger pod restarts or traffic redirection.
func Healthcheck(client *mongo.Client) func(context.Context) error {
	return func(ctx context.Context) error {
		if err := client.Ping(ctx, nil); err != nil {
			return errors.Join(ErrHealthcheckFailed, err)
		}
		return nil
	}
}
