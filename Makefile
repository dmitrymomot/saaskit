# PKG is an optional parameter to run commands for specific packages
# If not specified, commands run for all packages (./...)
# Examples:
#   make test                    # Test all packages (default)
#   make test PKG=./pkg/scopes     # Test only pkg/scopes
#   make lint PKG=./modules/auth   # Lint only modules/auth
#   make fmt PKG=./handler         # Format only handler package

PKG ?= ./...

.PHONY: test
test:
	@echo "Running tests ($(PKG))..."
	@go clean -cache && go test -race -cover $(PKG)

.PHONY: lint
lint:
	@echo "Running linter ($(PKG))..."
	@go vet $(PKG)
	@golangci-lint run --tests=false $(PKG)

.PHONY: fmt
fmt:
	@echo "Formatting code ($(PKG))..."
	@go fmt $(PKG)
	@goimports -w -local github.com/dmitrymomot/saaskit $(shell go list -f '{{.Dir}}' $(PKG))

.PHONY: bench
bench:
	@echo "Running benchmarks ($(PKG))..."
	@go test -bench=. -benchmem $(PKG)

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  test   - Run tests with race detector and coverage"
	@echo "  lint   - Run go vet and golangci-lint"
	@echo "  fmt    - Format code with go fmt and goimports"
	@echo "  bench  - Run benchmarks"
	@echo ""
	@echo "Use PKG=./path/to/package to run commands for specific packages:"
	@echo "  make test PKG=./pkg/scopes"
	@echo "  make lint PKG=./modules/auth"
