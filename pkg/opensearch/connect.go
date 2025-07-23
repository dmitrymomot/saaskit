package opensearch

import (
	"context"
	"errors"

	"github.com/opensearch-project/opensearch-go/v2"
)

// New creates a new OpenSearch client.
// It returns an error if the client cannot be created.
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

	// Healthcheck
	if err := Healthcheck(client)(ctx); err != nil {
		return nil, err
	}

	return client, nil
}
