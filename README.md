# saaskit

> ⚠️ **WARNING: This package is under active development and is NOT ready for production use. APIs may change without notice.**

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/dmitrymomot/saaskit)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/dmitrymomot/saaskit)](https://github.com/dmitrymomot/saaskit/tags)
[![Go Reference](https://pkg.go.dev/badge/github.com/dmitrymomot/saaskit.svg)](https://pkg.go.dev/github.com/dmitrymomot/saaskit)
[![License](https://img.shields.io/github/license/dmitrymomot/saaskit)](https://github.com/dmitrymomot/saaskit/blob/main/LICENSE)

[![Tests](https://github.com/dmitrymomot/saaskit/actions/workflows/tests.yml/badge.svg)](https://github.com/dmitrymomot/saaskit/actions/workflows/tests.yml)
[![CodeQL Analysis](https://github.com/dmitrymomot/saaskit/actions/workflows/codeql.yml/badge.svg)](https://github.com/dmitrymomot/saaskit/actions/workflows/codeql.yml)
[![GolangCI Lint](https://github.com/dmitrymomot/saaskit/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/dmitrymomot/saaskit/actions/workflows/golangci-lint.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/dmitrymomot/saaskit)](https://goreportcard.com/report/github.com/dmitrymomot/saaskit)

## About

SaasKit is a minimal, pragmatic Go framework for building SaaS applications. It's designed for solo developers who want to ship MVPs quickly without sacrificing code quality or type safety. The framework adheres to principles of explicitness, type safety, and convention, with escape hatches.

## Quick Start

```go
package main

import (
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/dmitrymomot/saaskit/modules/auth"
    "github.com/dmitrymomot/saaskit/modules/billing"
)

func main() {
    r := chi.NewRouter()

    // Mount authentication module
    r.Mount("/auth", auth.Router(auth.Config{
        UserStore:   db,
        Sessions:    sessionStore,
        EmailSender: emailClient,
    }))

    // Mount billing module
    r.Mount("/billing", billing.Router(billing.Config{
        StripeKey: stripeSecretKey,
        UserStore: db,
    }))

    http.ListenAndServe(":8080", r)
}
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
