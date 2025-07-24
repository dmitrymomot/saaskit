// Package mongo provides utility functions and types that simplify working with
// MongoDB in Go applications.
//
// It wraps the official MongoDB Go driver and offers:
//
//   • Config type for environment-driven connection parameters
//   • New and NewWithDatabase helpers that establish a client with configurable
//     retry logic and time-outs
//   • Healthcheck helper that can be plugged into readiness / liveness probes
//   • Pre-declared error variables describing common failure scenarios
//
// # Usage
//
//	import (
//		"context"
//		"github.com/dmitrymomot/saaskit/pkg/mongo"
//	)
//
//	func main() {
//		cfg := mongo.Config{
//			ConnectionURL: "mongodb://localhost:27017",
//		}
//
//		client, err := mongo.New(context.Background(), cfg)
//		if err != nil {
//			log.Fatal(err)
//		}
//		defer client.Disconnect(context.Background())
//
//		db, _ := mongo.NewWithDatabase(context.Background(), cfg, "mydb")
//
//		// Wire health check
//		health := mongo.Healthcheck(client)
//		if err := health(context.Background()); err != nil {
//			log.Println("mongo is unavailable:", err)
//		}
//	}
//
// # Configuration
//
// The Config struct is annotated with github.com/caarlos0/env tags so it can be
// populated from environment variables. Refer to the field-level comments in
// Config for available variables and their defaults.
//
// # Error Handling
//
// Errors returned by this package wrap the underlying driver errors and may be
// compared against ErrFailedToConnectToMongo and ErrHealthcheckFailed.
//
// # See Also
//
// Documentation for the official driver: https://pkg.go.dev/go.mongodb.org/mongo-driver.
package mongo
