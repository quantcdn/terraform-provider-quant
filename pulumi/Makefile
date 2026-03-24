.PHONY: tfgen provider build clean test test-e2e coverage generate

SCHEMA_DIR := schema

build: tfgen provider

tfgen:
	go build -o bin/pulumi-tfgen-quant ./cmd/pulumi-tfgen-quant

provider:
	go build -o bin/pulumi-resource-quant ./cmd/pulumi-resource-quant

generate: tfgen
	mkdir -p $(SCHEMA_DIR)
	./bin/pulumi-tfgen-quant schema --out $(SCHEMA_DIR)
	cp $(SCHEMA_DIR)/schema.json cmd/pulumi-resource-quant/schema.json
	cp $(SCHEMA_DIR)/bridge-metadata.json provider/bridge-metadata.json

test: ## Run mock acceptance tests (preview-based, no API needed)
	go test -v -cover ./provider/

test-e2e: ## Run E2E tests against staging API (requires QUANTCDN_API_TOKEN)
	go test -v -tags=e2e -timeout=30m ./e2e/

coverage: ## Generate coverage report
	go test -coverprofile=coverage.out ./provider/
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean:
	rm -rf bin/ $(SCHEMA_DIR) coverage.out coverage.html
