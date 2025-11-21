# ABOUTME: Makefile for pagen - Personal Agent Toolkit
# ABOUTME: Standard build targets for testing, building, and development
.PHONY: help build test test-race clean install lint fmt

BINARY_NAME=pagen
GO=go
GOFLAGS=-v

help: ## Show this help message
	@echo "Pagen - Personal Agent Toolkit"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the pagen binary
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME)

test: ## Run all tests
	$(GO) test $(GOFLAGS) ./...

test-race: ## Run tests with race detection
	$(GO) test -race -count=1 $(GOFLAGS) ./...

test-coverage: ## Run tests with coverage
	$(GO) test -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

install: build ## Build and install to GOPATH/bin
	$(GO) install

lint: ## Run golangci-lint
	golangci-lint run --timeout=10m

fmt: ## Format code with gofmt and goimports
	$(GO) fmt ./...
	goimports -w .

vet: ## Run go vet
	$(GO) vet ./...

mod-tidy: ## Tidy go.mod
	$(GO) mod tidy

all: clean fmt vet lint test build ## Run all checks and build

.DEFAULT_GOAL := help
