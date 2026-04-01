# ABOUTME: Build and development targets for the crm project.
# ABOUTME: Provides build, test, lint, and install targets.

.PHONY: build test test-race test-coverage install lint fmt clean check

build:
	go build -o crm ./cmd/crm

test:
	go test ./... -v

test-race:
	go test -race ./... -v

test-coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

install:
	go install ./cmd/crm

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w .

clean:
	rm -f crm coverage.out coverage.html
	go clean

check: fmt lint test

.DEFAULT_GOAL := build
