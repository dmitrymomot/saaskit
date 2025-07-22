# httpserver

A lightweight wrapper around `net/http` that provides graceful shutdown and configuration via functional options.

## Quick Start

```go
router := chi.NewRouter()
router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
})

srv := httpserver.New(
    httpserver.WithAddr(":8080"),
    httpserver.WithShutdownTimeout(10*time.Second),
)

if err := srv.Run(ctx, router); err != nil {
    log.Error("server stopped", "err", err)
}
```

## Options

- `WithAddr(string)` – listen address (default `":8080"`)
- `WithReadTimeout(time.Duration)`
- `WithWriteTimeout(time.Duration)`
- `WithIdleTimeout(time.Duration)`
- `WithShutdownTimeout(time.Duration)` – graceful shutdown deadline (default 5s)
- `WithServer(*http.Server)` – use preconfigured server (its Handler and timeouts may be modified)
- `WithLogger(*slog.Logger)` – external logger; falls back to noop
- `WithStartHook(func(*slog.Logger))` – run when server starts
- `WithStopHook(func(*slog.Logger))` – run after server stops

All option constructors panic if provided invalid values (e.g. negative durations).

## Errors

`Run` returns errors wrapped with `ErrStart` and `Shutdown` wraps underlying errors with `ErrShutdown`. Use `errors.Is` to check.

```go
if err := srv.Run(ctx, router); err != nil {
    if errors.Is(err, httpserver.ErrStart) {
        // handle start failure
    }
}
```

When shutting down programmatically:

```go
if err := srv.Shutdown(context.Background()); err != nil {
    if errors.Is(err, httpserver.ErrShutdown) {
        log.Error("graceful shutdown failed", "err", err)
    }
}
```
