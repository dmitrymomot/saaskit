package opensearch

import (
	"context"
	"errors"

	"github.com/opensearch-project/opensearch-go/v2"
)

// Healthcheck is a function that checks the health of the database.
// It returns an error if the database is not healthy.
func Healthcheck(client *opensearch.Client) func(context.Context) error {
	return func(context.Context) error {
		if _, err := client.Info(); err != nil {
			return errors.Join(ErrHealthcheckFailed, err)
		}
		return nil
	}
}
