.PHONY: build test testacc clean fmt lint coverage help

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the provider
	go build -o terraform-provider-quant

test: ## Run unit tests
	go test ./...

testacc: ## Run acceptance tests
	TF_ACC=1 go test ./internal/provider/ -v

testacc-short: ## Run acceptance tests with short timeout
	TF_ACC=1 go test ./internal/provider/ -v -timeout=10m

clean: ## Clean build artifacts
	rm -f terraform-provider-quant
	rm -f coverage.out

fmt: ## Format Go code
	go fmt ./...

lint: ## Run linters
	golangci-lint run

coverage: ## Run tests with coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

install: ## Install the provider locally
	go install

deps: ## Download dependencies
	go mod download
	go mod tidy

docs: ## Generate documentation
	@echo "Documentation is maintained manually in docs/ directory"

release: ## Build for release
	GOOS=linux GOARCH=amd64 go build -o terraform-provider-quant-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -o terraform-provider-quant-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -o terraform-provider-quant-darwin-arm64
	GOOS=windows GOARCH=amd64 go build -o terraform-provider-quant-windows-amd64.exe

# Development helpers
dev-setup: deps build ## Set up development environment
	@echo "Development environment ready!"

check: fmt lint test ## Run all checks (format, lint, test)
	@echo "All checks passed!" 