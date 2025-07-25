# Variables
APP_NAME := mcp-gateway
MODULE_NAME := github.com/matthisholleville/mcp-gateway
CONFIG_FILE := ./config/config.yaml
BUILD_DIR := ./bin
GO_VERSION := 1.23.2

# Colors for messages
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
RESET := \033[0m

.PHONY: help run build clean test lint fmt vet deps install serve dev check mocks swagger envrc-sample check-envrc

## help: Show this help
help:
	@echo "Available commands:"
	@echo ""
	@grep -E '^## [a-zA-Z_-]+:' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = "## "}; {printf "  $(GREEN)%-15s$(RESET) %s\n", $$2, $$3}' | \
		sed 's/:/ /'

## serve: Run the application in server mode with debug configuration
serve:
	@echo "$(YELLOW)Starting server in debug mode...$(RESET)"
	go run main.go serve --log-format=raw --log-level=debug --config=$(CONFIG_FILE)

## dev: Alias for serve (for development)
dev: serve

## run: Run the application (equivalent to serve)
run: serve

## build: Compile the application
build:
	@echo "$(YELLOW)Building application...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) main.go
	@echo "$(GREEN)✓ Application built in $(BUILD_DIR)/$(APP_NAME)$(RESET)"

## install: Install the application in $GOPATH/bin
install:
	@echo "$(YELLOW)Installing application...$(RESET)"
	go install .
	@echo "$(GREEN)✓ Application installed$(RESET)"

## test: Run tests
test:
	@echo "$(YELLOW)Running tests...$(RESET)"
	go test -v ./...

## test-cover: Run tests with coverage
test-cover:
	@echo "$(YELLOW)Running tests with coverage...$(RESET)"
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report generated in coverage.html$(RESET)"

## bench: Run benchmarks
bench:
	@echo "$(YELLOW)Running benchmarks...$(RESET)"
	go test -bench=. -benchmem ./...

## lint: Run linter (golangci-lint)
lint:
	@echo "$(YELLOW)Running linter...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "$(RED)golangci-lint is not installed. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(RESET)"; \
	fi

## fmt: Format the code
fmt:
	@echo "$(YELLOW)Formatting code...$(RESET)"
	go fmt ./...
	@echo "$(GREEN)✓ Code formatted$(RESET)"

## vet: Run go vet
vet:
	@echo "$(YELLOW)Running go vet...$(RESET)"
	go vet ./...
	@echo "$(GREEN)✓ go vet completed$(RESET)"

## deps: Download dependencies
deps:
	@echo "$(YELLOW)Downloading dependencies...$(RESET)"
	go mod download
	@echo "$(GREEN)✓ Dependencies downloaded$(RESET)"

## tidy: Clean up dependencies
tidy:
	@echo "$(YELLOW)Tidying dependencies...$(RESET)"
	go mod tidy
	@echo "$(GREEN)✓ Dependencies tidied$(RESET)"

## vendor: Create vendor directory
vendor:
	@echo "$(YELLOW)Creating vendor directory...$(RESET)"
	go mod vendor
	@echo "$(GREEN)✓ Vendor directory created$(RESET)"

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test check-envrc
	@echo "$(GREEN)✓ All checks passed$(RESET)"

## clean: Clean generated files
clean:
	@echo "$(YELLOW)Cleaning...$(RESET)"
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean
	@echo "$(GREEN)✓ Cleaning completed$(RESET)"

## docker-build: Build Docker image (if Dockerfile exists)
docker-build:
	@if [ -f Dockerfile ]; then \
		echo "$(YELLOW)Building Docker image...$(RESET)"; \
		docker build -t $(APP_NAME) .; \
		echo "$(GREEN)✓ Docker image built$(RESET)"; \
	else \
		echo "$(RED)No Dockerfile found$(RESET)"; \
	fi

## version: Display versions
version:
	@echo "Go version: $(shell go version)"
	@echo "Module: $(MODULE_NAME)"
	@echo "App: $(APP_NAME)"

mocks:
	@echo "Generating mocks..."
	@mockgen -source=pkg/grafana/grafana.go -destination=pkg/grafana/mocks/grafana_mocks.go -package=mocks
	@mockgen -source=pkg/nexus/nexus.go -destination=pkg/nexus/mocks/nexus_mocks.go -package=mocks
	@mockgen -source=pkg/pritunl/pritunl.go -destination=pkg/pritunl/mocks/pritunl_mocks.go -package=mocks
	@echo "Mocks generated successfully!"

swagger:
	@echo "Generating Swagger documentation..."
	@rm -rf ./swagger
	@swag init \
		--generalInfo ./internal/server/server.go \
		--output ./swagger \
		--parseDependency \
		--parseInternal \
		--parseDepth 2
	@echo "Swagger documentation generated successfully!"

## envrc-sample: Generate .envrc-sample with obfuscated values for git safety
envrc-sample:
	@echo "$(YELLOW)Generating .envrc-sample...$(RESET)"
	@if [ ! -f .envrc ]; then \
		echo "$(RED)No .envrc file found$(RESET)"; \
		exit 1; \
	fi
	@sed -E \
		-e 's/export ([^=]+)="[^"]*"/export \1="***REDACTED***"/g' \
		-e 's/export ([^=]+)=\$$\(cat [^)]+\)/export \1="***REDACTED_FILE_CONTENT***"/g' \
		.envrc > .envrc-sample
	@echo "# This is a sample environment file" > .envrc-sample.tmp
	@echo "# Copy this to .envrc and fill in your actual values" >> .envrc-sample.tmp
	@echo "# DO NOT commit .envrc with real values!" >> .envrc-sample.tmp
	@echo "" >> .envrc-sample.tmp
	@cat .envrc-sample >> .envrc-sample.tmp
	@mv .envrc-sample.tmp .envrc-sample
	@echo "$(GREEN)✓ .envrc-sample generated successfully$(RESET)"
	@echo "$(YELLOW)📋 Review the generated file:$(RESET)"
	@cat .envrc-sample

## check-envrc: Verify .envrc-sample is up to date
check-envrc:
	@echo "$(YELLOW)Checking if .envrc-sample is up to date...$(RESET)"
	@if [ ! -f .envrc-sample ]; then \
		echo "$(RED)❌ .envrc-sample missing! Run 'make envrc-sample' to generate it$(RESET)"; \
		exit 1; \
	fi
	@if [ ! -f .envrc ]; then \
		echo "$(YELLOW)⚠️  No .envrc found (normal in CI). Skipping .envrc-sample check.$(RESET)"; \
	else \
		echo "$(YELLOW)Comparing .envrc-sample with current .envrc...$(RESET)"; \
		sed -E \
			-e 's/export ([^=]+)="[^"]*"/export \1="***REDACTED***"/g' \
			-e 's/export ([^=]+)=\$$\(cat [^)]+\)/export \1="***REDACTED_FILE_CONTENT***"/g' \
			.envrc > .envrc-temp; \
		echo "# This is a sample environment file" > .envrc-sample-temp; \
		echo "# Copy this to .envrc and fill in your actual values" >> .envrc-sample-temp; \
		echo "# DO NOT commit .envrc with real values!" >> .envrc-sample-temp; \
		echo "" >> .envrc-sample-temp; \
		cat .envrc-temp >> .envrc-sample-temp; \
		rm -f .envrc-temp; \
		if ! diff -q .envrc-sample .envrc-sample-temp >/dev/null 2>&1; then \
			echo "$(RED)❌ .envrc-sample is outdated! Run 'make envrc-sample' to update it$(RESET)"; \
			rm -f .envrc-sample-temp; \
			exit 1; \
		else \
			echo "$(GREEN)✅ .envrc-sample is up to date$(RESET)"; \
			rm -f .envrc-sample-temp; \
		fi; \
	fi

# Default rule
.DEFAULT_GOAL := help