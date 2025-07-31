// Package mongo provides MongoDB connection management optimized for SaaS applications
// deployed by solo developers.
//
// This package emphasizes operational reliability through environment-based configuration,
// aggressive retry logic, and proper connection pooling defaults that work well for
// small-to-medium SaaS workloads without manual tuning.
//
// Key features:
//   - Environment-driven configuration eliminates deployment complexity
//   - Built-in retry logic handles MongoDB Atlas transient failures gracefully
//   - Connection pool defaults optimized for typical SaaS traffic patterns
//   - Health check integration for Kubernetes/Docker orchestration
//   - Error types compatible with errors.Is() for clean error handling
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
// Configuration is entirely environment-driven to simplify deployment across
// development, staging, and production environments. This eliminates the need
// for config file management and enables secure credential handling through
// environment variables or secret management systems.
//
// # Error Handling
//
// Connection failures are wrapped in domain-specific errors to enable proper
// error handling in application code. Use errors.Is() to check for specific
// failure scenarios and implement appropriate retry or fallback logic.
//
// # See Also
//
// Documentation for the official driver: https://pkg.go.dev/go.mongodb.org/mongo-driver.
package mongo
