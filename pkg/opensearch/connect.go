package opensearch

import (
	"context"
	"errors"

	"github.com/opensearch-project/opensearch-go/v2"
)

// New creates an OpenSearch client and verifies cluster connectivity.
// Performs an immediate health check to fail fast if the cluster is unreachable,
// preventing broken clients from being returned to callers.
func New(ctx context.Context, cfg Config) (*opensearch.Client, error) {
	ocfg := opensearch.Config{
		Addresses:    cfg.Addresses,
		Username:     cfg.Username,
		Password:     cfg.Password,
		MaxRetries:   cfg.MaxRetries,
		DisableRetry: cfg.DisableRetry,
	}
	client, err := opensearch.NewClient(ocfg)
	if err != nil {
		return nil, errors.Join(ErrConnectionFailed, err)
	}

	// Verify cluster is reachable before returning client to prevent runtime failures
	if err := Healthcheck(client)(ctx); err != nil {
		return nil, err
	}

	return client, nil
}
