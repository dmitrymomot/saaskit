package opensearch

// Config holds OpenSearch client connection parameters with environment variable mapping.
// Uses struct tags compatible with github.com/dmitrymomot/saaskit/pkg/config for
// zero-config environment-based initialization.
type Config struct {
	Addresses    []string `env:"OPENSEARCH_ADDRESSES,required"`
	Username     string   `env:"OPENSEARCH_USERNAME,notEmpty"`
	Password     string   `env:"OPENSEARCH_PASSWORD,notEmpty"`
	MaxRetries   int      `env:"OPENSEARCH_MAX_RETRIES" envDefault:"3"`
	DisableRetry bool     `env:"OPENSEARCH_DISABLE_RETRY" envDefault:"false"`
}
