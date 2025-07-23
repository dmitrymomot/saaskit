package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// New creates a new mongo client.
// It returns an error if the client cannot be created.
func New(ctx context.Context, cfg Config) (*mongo.Client, error) {
	// Retry to connect to the mongo server
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
			if err := client.Ping(ctx, nil); err == nil {
				return client, nil
			}
		}

		// Wait for the next retry interval
		time.Sleep(cfg.RetryInterval)
	}

	return nil, ErrFailedToConnectToMongo
}

// NewWithDatabase creates a new mongo client and returns a database object.
// This function is useful when you want to connect to a specific database.
func NewWithDatabase(ctx context.Context, cfg Config, database string) (*mongo.Database, error) {
	client, err := New(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return client.Database(database), nil
}
