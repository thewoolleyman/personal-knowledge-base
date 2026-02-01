.PHONY: help build test test-accept test-int test-all lint vet tidy clean run verify-hooks

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

## verify-hooks: Prove event-logging and memory-import hooks work end-to-end
verify-hooks:
	@echo "==> Checking prerequisites..."
	@command -v jq >/dev/null 2>&1 || { echo "FAIL: jq not found"; exit 1; }
	@test -x .claude/hooks/persist-events.sh || { echo "FAIL: persist-events.sh missing or not executable"; exit 1; }
	@test -x .claude/hooks/import-events.sh || { echo "FAIL: import-events.sh missing or not executable"; exit 1; }
	@echo "==> Cleaning test state..."
	@rm -f .claude-flow/learning/events.jsonl
	@echo "==> Testing persist-events.sh with Edit event..."
	@echo '{"tool_name":"Edit","tool_input":{"file_path":"/tmp/test-file.go"}}' | .claude/hooks/persist-events.sh
	@test -f .claude-flow/learning/events.jsonl || { echo "FAIL: events.jsonl not created after Edit event"; exit 1; }
	@jq -e 'select(.type=="edit" and .file=="/tmp/test-file.go")' .claude-flow/learning/events.jsonl >/dev/null || { echo "FAIL: Edit event not found in events.jsonl"; exit 1; }
	@echo "    OK: Edit event recorded"
	@echo "==> Testing persist-events.sh with Task event..."
	@echo '{"tool_name":"Task","tool_input":{"description":"Test task","prompt":"Do something","subagent_type":"coder"}}' | .claude/hooks/persist-events.sh
	@jq -e 'select(.type=="task" and .agent=="coder")' .claude-flow/learning/events.jsonl >/dev/null || { echo "FAIL: Task event not found"; exit 1; }
	@echo "    OK: Task event recorded"
	@echo "==> Testing persist-events.sh with Bash event..."
	@echo '{"tool_name":"Bash","tool_input":{"command":"go test ./..."}}' | .claude/hooks/persist-events.sh
	@jq -e 'select(.type=="command")' .claude-flow/learning/events.jsonl >/dev/null || { echo "FAIL: Bash event not found"; exit 1; }
	@echo "    OK: Bash event recorded"
	@echo "==> Testing that hook-infrastructure commands are skipped..."
	@echo '{"tool_name":"Bash","tool_input":{"command":"npx @claude-flow/cli@latest memory search --query x"}}' | .claude/hooks/persist-events.sh
	@LINES=$$(wc -l < .claude-flow/learning/events.jsonl | tr -d ' '); \
	 if [ "$$LINES" -ne 3 ]; then echo "FAIL: expected 3 lines, got $$LINES (hook command was not skipped)"; exit 1; fi
	@echo "    OK: Hook-infrastructure command correctly skipped"
	@echo "==> Testing import-events.sh (session-end import)..."
	@.claude/hooks/import-events.sh 2>&1
	@test ! -f .claude-flow/learning/events.jsonl || { echo "FAIL: events.jsonl should have been rotated"; exit 1; }
	@ls .claude-flow/learning/events-*.jsonl.bak >/dev/null 2>&1 || { echo "FAIL: backup file not created"; exit 1; }
	@echo "    OK: Events imported and log rotated"
	@echo ""
	@echo "All checks passed."
