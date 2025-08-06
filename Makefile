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

.PHONY: load
load:
	@echo "Running load tests ($(PKG))..."
	@go test -tags=load -timeout=10m $(PKG)

.PHONY: load-race
load-race:
	@echo "Running load tests with race detector ($(PKG))..."
	@go test -tags=load -race -timeout=15m $(PKG)

.PHONY: perf
perf:
	@echo "Running performance tests: benchmarks + load tests ($(PKG))..."
	@go test -bench=. -benchmem $(PKG)
	@go test -tags=load -timeout=10m $(PKG)

.PHONY: test-fast
test-fast:
	@echo "Running fast unit tests only ($(PKG))..."
	@go clean -cache && go test -short -race -cover $(PKG)

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  test      - Run tests with race detector and coverage"
	@echo "  test-fast - Run fast unit tests only (with -short flag)"
	@echo "  load      - Run load/stress tests"
	@echo "  load-race - Run load tests with race detector"
	@echo "  bench     - Run benchmarks"
	@echo "  perf      - Run all performance tests (bench + load)"
	@echo "  lint      - Run go vet and golangci-lint"
	@echo "  fmt       - Format code with go fmt and goimports"
	@echo ""
	@echo "Use PKG=./path/to/package to run commands for specific packages:"
	@echo "  make test PKG=./pkg/scopes"
	@echo "  make load PKG=./pkg/broadcast"
	@echo "  make lint PKG=./modules/auth"
