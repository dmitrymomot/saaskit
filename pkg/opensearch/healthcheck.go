package opensearch

import (
	"context"
	"errors"

	"github.com/opensearch-project/opensearch-go/v2"
)

// Healthcheck returns a function suitable for liveness/readiness probes.
// The returned function calls client.Info() to verify cluster connectivity
// and is safe for concurrent use in HTTP health endpoints.
func Healthcheck(client *opensearch.Client) func(context.Context) error {
	return func(ctx context.Context) error {
		if _, err := client.Info(
			client.Info.WithContext(ctx),
			client.Info.WithErrorTrace(),
		); err != nil {
			return errors.Join(ErrHealthcheckFailed, err)
		}
		return nil
	}
}
