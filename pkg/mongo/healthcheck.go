package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Healthcheck is a function that checks the health of the database.
// It returns an error if the database is not healthy.
func Healthcheck(client *mongo.Client) func(context.Context) error {
	return func(ctx context.Context) error {
		if err := client.Ping(ctx, nil); err != nil {
			return errors.Join(ErrHealthcheckFailed, err)
		}
		return nil
	}
}
