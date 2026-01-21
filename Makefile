.PHONY: all lint fmt test test-unit test-integration build clean help

all: lint test ## Default target

lint: ## Run golangci-lint
	@echo "Running linter..."
	golangci-lint run ./...

fmt: ## Run go fmt
	@echo "Formatting code..."
	go fmt ./...
	cd integration && go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

test: test-unit test-integration ## Run all tests (unit + integration)

test-unit: ## Run unit tests only (main module)
	@echo "Running unit tests..."
	go test -v -race ./...

test-integration: ## Run integration tests (requires Docker)
	@echo "Running integration tests (requires Docker)..."
	cd integration && go test -v -count=1 ./...

test-integration-short: ## Run integration tests with short timeout
	@echo "Running integration tests with 5m timeout..."
	cd integration && go test -v -count=1 -timeout 5m ./...

build: ## Build the library
	@echo "Building..."
	go build ./...

tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	go mod tidy
	cd integration && go mod tidy

clean: ## Clean build artifacts
	@echo "Cleaning..."
	go clean ./...
	cd integration && go clean ./...

coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

check-tools: ## Check if all tools are installed
	@echo "Checking required tools..."
	@which go > /dev/null || (echo "go is not installed" && exit 1)
	@which golangci-lint > /dev/null || (echo "golangci-lint is not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	@which docker > /dev/null || (echo "docker is not installed (required for integration tests)" && exit 1)
	@echo "All tools installed!"

help:	## Shows this help.
	@grep -hE '^[A-Za-z0-9_ \-]*?:.*##.*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
