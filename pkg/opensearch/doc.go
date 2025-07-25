// Package opensearch provides a lightweight wrapper around the official OpenSearch
// Go client adding type-safe configuration, automatic cluster health checking,
// and standardized error values.
//
// The package is designed for concurrent applications. It builds on top of
// github.com/opensearch-project/opensearch-go/v2 which is thread-safe by
// design. Beyond the underlying client, the package focuses on three public
// touch points:
//
//   - Config – declarative representation of connection settings that can be
//     populated from environment variables via github.com/dmitrymomot/saaskit/pkg/config.
//
//   - New – constructs a ready-to-use *opensearch.Client instance and performs
//     an initial Healthcheck ensuring the cluster is reachable.
//
//   - Healthcheck – returns a function suitable for liveness / readiness probes
//     (for example in HTTP /health endpoints).
//
// Errors specific to connectivity are exposed as ErrConnectionFailed and
// ErrHealthcheckFailed so that callers can distinguish infrastructure problems
// from business logic errors.
//
// # Usage
//
// Basic connection:
//
//	import (
//	    "context"
//	    "github.com/dmitrymomot/saaskit/pkg/opensearch"
//	)
//
//	client, err := opensearch.New(context.Background(), opensearch.Config{
//	    Addresses: []string{"https://localhost:9200"},
//	    Username:  "admin",
//	    Password:  "admin",
//	})
//	if err != nil {
//	    // use errors.Is(err, opensearch.ErrConnectionFailed)
//	}
//
//	info, _ := client.Info()
//
// Environment-based configuration:
//
//	import (
//	    "context"
//	    "github.com/dmitrymomot/saaskit/pkg/config"
//	    "github.com/dmitrymomot/saaskit/pkg/opensearch"
//	)
//
//	cfg, _ := config.Load[opensearch.Config]()
//	client, _ := opensearch.New(context.Background(), cfg)
//
// # Error Handling
//
// Use the standard errors.Is / errors.As helpers to check for sentinel errors:
//
//	if err := opensearch.Healthcheck(client)(ctx); err != nil {
//	    if errors.Is(err, opensearch.ErrHealthcheckFailed) {
//	        // handle health-check failure
//	    }
//	}
//
// # Performance Considerations
//
// The MaxRetries and DisableRetry fields in Config map directly to the
// opensearch-go/v2 client and should be tuned according to the latency and
// reliability of your network.
package opensearch
