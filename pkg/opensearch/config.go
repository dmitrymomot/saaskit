package opensearch

// Config represents the configuration settings for the OpenSearch client.
type Config struct {
	Addresses    []string `env:"OPENSEARCH_ADDRESSES,required"`
	Username     string   `env:"OPENSEARCH_USERNAME,notEmpty"`
	Password     string   `env:"OPENSEARCH_PASSWORD,notEmpty"`
	MaxRetries   int      `env:"OPENSEARCH_MAX_RETRIES" default:"3"`
	DisableRetry bool     `env:"OPENSEARCH_DISABLE_RETRY" default:"false"`
}
