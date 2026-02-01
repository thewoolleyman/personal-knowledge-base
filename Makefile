.PHONY: help build test test-accept test-int test-all lint vet tidy clean run

BINARY := pkb
BUILD_DIR := .

## help: Show this help message
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'

## build: Compile the pkb binary
build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/pkb

## test: Run unit tests with race detection and coverage
test:
	go test -race -cover ./...

## test-accept: Run acceptance tests (builds real binary, tests from user perspective)
test-accept:
	go test -tags=acceptance -v ./tests/acceptance/

## test-int: Run component integration tests (requires Google Drive credentials)
test-int:
	go test -tags=integration -race -v ./...

## test-all: Run unit, acceptance, and integration tests
test-all: test test-accept test-int

## lint: Run golangci-lint (install with: brew install golangci-lint)
lint:
	golangci-lint run ./...

## vet: Run go vet
vet:
	go vet ./...

## tidy: Tidy and verify go.mod
tidy:
	go mod tidy
	git diff --exit-code go.mod go.sum

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)

## run: Build and run pkb --help
run: build
	./$(BINARY) --help
