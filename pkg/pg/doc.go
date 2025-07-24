// Package pg provides utilities for interacting with PostgreSQL using the
// pgx/v5 driver. It offers a thin abstraction around connection pooling,
// migrations, health checks, and common error helpers so that applications can
// bootstrap a resilient database layer with only a few lines of code.
//
// The package purposefully keeps a very small API surface while relying on
// battle-tested upstream libraries (`pgx/v5` for connectivity and `goose/v3`
// for schema migrations) so that callers are never locked-in and can freely
// extend the behaviour where needed.
//
// # Architecture
//
// At its core the package exposes three cooperating building blocks:
//
//   • Config – a declarative struct whose fields are populated from
//     environment variables via github.com/caarlos0/env. It controls
//     connection pool limits, health-check cadence and migration paths.
//
//   • Connect – opens a *pgxpool.Pool based on Config, retrying with
//     exponential back-off until the database becomes available.
//
//   • Migrate – runs goose database migrations against the same connection
//     pool, guaranteeing the schema is up-to-date before the service starts
//     serving traffic.
//
// The helpers are intentionally decoupled so that you can plug them into your
// preferred dependency-injection or lifecycle framework (Fx, Wire, Uber/fx
// etc.).
//
// # Usage
//
// Basic set-up using the default configuration:
//
//	package main
//
//	import (
// 	    "context"
// 	    "log/slog"
//	    "github.com/dmitrymomot/saaskit/pkg/pg"
// 	)
//
// 	func main() {
// 	    var cfg pg.Config
// 	    if err := env.Parse(&cfg); err != nil {
// 	        panic(err)
// 	    }
//
// 	    ctx := context.Background()
// 	    pool, err := pg.Connect(ctx, cfg)
// 	    if err != nil {
// 	        panic(err)
// 	    }
// 	    defer pool.Close()
//
// 	    if err := pg.Migrate(ctx, pool, cfg, slog.Default()); err != nil {
// 	        panic(err)
// 	    }
//
// 	    // expose health endpoint
// 	    health := pg.Healthcheck(pool)
// 	    if err := health(ctx); err != nil {
// 	        panic(err)
// 	    }
// 	}
//
// # Configuration
//
// All configuration values are provided through environment variables so that
// they can be tuned per-environment without code changes. Refer to the field
// tags in Config for exact variable names and defaults.
//
// # Error Handling
//
// Convenience helpers such as [pg.IsDuplicateKeyError] or
// [pg.IsForeignKeyViolationError] unwrap errors returned by pgx/
// `*pgconn.PgError` and make error classification trivial inside business
// logic.
//
// Deprecated: none at the moment.
package pg
