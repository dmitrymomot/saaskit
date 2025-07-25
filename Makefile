.PHONY: mocks
mocks:
	@echo "Generating mocks..."
	@mockery --config=.mockery.yaml

.PHONY: test
test: mocks
	@echo "Running tests..."
	@go test -race -cover ./...

.PHONY: lint
lint:
	@echo "Running linter..."
	@go vet ./...
	@golangci-lint run --tests=false

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w -local github.com/dmitrymomot/saaskit .
