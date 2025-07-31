package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// New establishes a MongoDB connection with retry logic optimized for production SaaS deployments.
//
// The function implements application-level retry logic because MongoDB Atlas instances
// can experience cold starts (5-8 seconds) and brief network hiccups that would otherwise
// cause application startup failures. The Ping() verification ensures the connection is
// actually usable before returning, preventing silent connection issues.
func New(ctx context.Context, cfg Config) (*mongo.Client, error) {
	for range cfg.RetryAttempts {
		client, err := mongo.Connect(
			options.Client().
				ApplyURI(cfg.ConnectionURL).
				SetConnectTimeout(cfg.ConnectTimeout).
				SetMaxPoolSize(cfg.MaxPoolSize).
				SetMinPoolSize(cfg.MinPoolSize).
				SetMaxConnIdleTime(cfg.MaxConnIdleTime).
				SetRetryWrites(cfg.RetryWrites).
				SetRetryReads(cfg.RetryReads),
		)
		if err == nil {
			// Ping verifies the connection is actually usable, not just established
			if err := client.Ping(ctx, nil); err == nil {
				return client, nil
			}
		}

		time.Sleep(cfg.RetryInterval)
	}

	return nil, ErrFailedToConnectToMongo
}

// NewWithDatabase provides a convenience wrapper that returns a database instance directly.
// This eliminates the common pattern of connecting and then selecting a database,
// reducing boilerplate code in typical SaaS application initialization.
func NewWithDatabase(ctx context.Context, cfg Config, database string) (*mongo.Database, error) {
	client, err := New(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return client.Database(database), nil
}
