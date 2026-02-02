.PHONY: help build test test-accept test-int test-all lint vet tidy clean run verify-hooks version scan-secrets scan-secrets-staged setup-hooks open-cicd-webpage

BINARY := pkb
BUILD_DIR := .
VERSION ?= $(shell cat VERSION 2>/dev/null || echo dev)

## help: Show this help message
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'

## build: Compile the pkb binary
build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY) ./cmd/pkb

## version: Print the current version
version:
	@echo $(VERSION)

## test: Run unit tests with race detection and coverage
test:
	go test -race -cover ./...

## test-accept: Run acceptance tests (builds real binary, tests from user perspective)
test-accept:
	go test -tags=acceptance -v ./tests/acceptance/

## test-int: Run component integration tests (requires Google Drive credentials)
test-int:
	go test -tags=integration -race -v -run TestIntegration ./...

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

## run: Build and run pkb with args (e.g. make run search "agentic")
ifeq (run,$(firstword $(MAKECMDGOALS)))
  RUN_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(RUN_ARGS):;@:)
endif
run: build
	./$(BINARY) $(RUN_ARGS)

## open-cicd-webpage: Open the GitHub Actions CI/CD page in the default browser (macOS)
open-cicd-webpage:
	open https://github.com/thewoolleyman/personal-knowledge-base/actions

## scan-secrets: Run gitleaks to detect hardcoded secrets (managed via mise)
scan-secrets:
	mise x -- gitleaks detect --source . --no-banner -c .gitleaks.toml --verbose

## scan-secrets-staged: Run gitleaks on staged files only (same check as pre-commit hook)
scan-secrets-staged:
	mise x -- gitleaks protect --staged --no-banner -c .gitleaks.toml --verbose

## setup-hooks: Install pre-commit hook with gitleaks + bd chaining
setup-hooks:
	@echo "Installing pre-commit hook (gitleaks + bd)..."
	@printf '%s\n' \
	  '#!/usr/bin/env sh' \
	  '# bd-shim v1 + gitleaks' \
	  '# bd-hooks-version: 0.49.0' \
	  '#' \
	  '# Pre-commit hook: gitleaks secrets scan, then bd (beads) export.' \
	  '#' \
	  '# Gitleaks runs first on staged content. If it detects secrets, the' \
	  '# commit is blocked before bd ever runs. If gitleaks is not installed,' \
	  '# a warning is printed but the commit proceeds (CI is the backstop).' \
	  '#' \
	  '# To reinstall this hook after bd overwrites it:' \
	  '#   make setup-hooks' \
	  '' \
	  '# --- Gitleaks: scan staged content for secrets ---' \
	  'GITLEAKS_CMD=""' \
	  'if command -v mise >/dev/null 2>&1; then' \
	  '    GITLEAKS_CMD="mise x -- gitleaks"' \
	  'elif command -v gitleaks >/dev/null 2>&1; then' \
	  '    GITLEAKS_CMD="gitleaks"' \
	  'fi' \
	  '' \
	  'if [ -n "$$GITLEAKS_CMD" ]; then' \
	  '    $$GITLEAKS_CMD protect --staged --no-banner -c .gitleaks.toml' \
	  '    if [ $$? -ne 0 ]; then' \
	  '        echo "" >&2' \
	  '        echo "ERROR: gitleaks detected secrets in staged files." >&2' \
	  '        echo "  Fix the issue and try again, or run:" >&2' \
	  '        echo "    make scan-secrets" >&2' \
	  '        echo "  to see full details." >&2' \
	  '        exit 1' \
	  '    fi' \
	  'else' \
	  '    echo "Warning: gitleaks not available, skipping pre-commit secret scan" >&2' \
	  '    echo "  Install via mise: mise install" >&2' \
	  'fi' \
	  '' \
	  '# --- bd (beads): export database to JSONL and stage ---' \
	  'if ! command -v bd >/dev/null 2>&1; then' \
	  '    echo "Warning: bd command not found in PATH, skipping beads pre-commit" >&2' \
	  '    echo "  Install bd: brew install steveyegge/tap/bd" >&2' \
	  '    echo "  Or add bd to your PATH" >&2' \
	  '    exit 0' \
	  'fi' \
	  '' \
	  'exec bd hook pre-commit "$$@"' \
	  > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Done. Pre-commit hook installed at .git/hooks/pre-commit"

## verify-hooks: Prove two-tier logging, context bundles, and recall work end-to-end
verify-hooks:
	@echo "==> Checking prerequisites..."
	@command -v jq >/dev/null 2>&1 || { echo "FAIL: jq not found"; exit 1; }
	@test -x .claude/hooks/log-hook-event.sh || { echo "FAIL: log-hook-event.sh missing or not executable"; exit 1; }
	@test -x .claude/hooks/build-context-bundle.sh || { echo "FAIL: build-context-bundle.sh missing or not executable"; exit 1; }
	@test -x .claude/hooks/recall-memory.sh || { echo "FAIL: recall-memory.sh missing or not executable"; exit 1; }
	@echo "==> Cleaning test state..."
	@rm -rf .claude-flow/learning/hook_logs/test-session
	@rm -f .claude-flow/learning/context_bundles/*_test-session.jsonl
	@echo "==> Testing log-hook-event.sh (raw event logging)..."
	@echo '{"session_id":"test-session","hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/test.go"}}' \
	  | .claude/hooks/log-hook-event.sh
	@test -f .claude-flow/learning/hook_logs/test-session/PostToolUse.jsonl \
	  || { echo "FAIL: raw log file not created"; exit 1; }
	@jq -e '.payload.tool_name=="Edit"' .claude-flow/learning/hook_logs/test-session/PostToolUse.jsonl >/dev/null \
	  || { echo "FAIL: raw log entry missing tool_name"; exit 1; }
	@echo "    OK: Raw event logged to hook_logs/test-session/PostToolUse.jsonl"
	@echo "==> Testing build-context-bundle.sh with Edit event..."
	@echo '{"session_id":"test-session","tool_name":"Edit","tool_input":{"file_path":"'$$(pwd)'/internal/server/server.go"}}' \
	  | .claude/hooks/build-context-bundle.sh --type tool
	@BUNDLE=$$(ls -t .claude-flow/learning/context_bundles/*_test-session.jsonl 2>/dev/null | head -1); \
	 test -n "$$BUNDLE" || { echo "FAIL: context bundle not created"; exit 1; }; \
	 echo "$$BUNDLE" > /tmp/pkb-verify-bundle-path; \
	 jq -e 'select(.op=="edit" and .file=="internal/server/server.go")' "$$BUNDLE" >/dev/null \
	   || { echo "FAIL: edit entry not found or path not relative"; exit 1; }
	@echo "    OK: Edit event in context bundle with relative path"
	@echo "==> Testing build-context-bundle.sh with Bash event..."
	@echo '{"session_id":"test-session","tool_name":"Bash","tool_input":{"command":"go test ./..."}}' \
	  | .claude/hooks/build-context-bundle.sh --type tool
	@cat .claude-flow/learning/context_bundles/*_test-session.jsonl 2>/dev/null \
	 | jq -e 'select(.op=="command")' >/dev/null \
	   || { echo "FAIL: command entry not found in bundle"; exit 1; }
	@echo "    OK: Bash command in context bundle"
	@echo "==> Testing that hook-infrastructure commands are skipped..."
	@echo '{"session_id":"test-session","tool_name":"Bash","tool_input":{"command":"npx @claude-flow/cli@latest memory search"}}' \
	  | .claude/hooks/build-context-bundle.sh --type tool
	@BUNDLE=$$(cat /tmp/pkb-verify-bundle-path); \
	 LINES=$$(cat .claude-flow/learning/context_bundles/*_test-session.jsonl 2>/dev/null | wc -l | tr -d ' '); \
	 if [ "$$LINES" -ne 2 ]; then echo "FAIL: expected 2 total lines, got $$LINES (hook command was not skipped)"; exit 1; fi
	@echo "    OK: Hook-infrastructure command correctly skipped"
	@echo "==> Testing build-context-bundle.sh with prompt..."
	@echo '{"session_id":"test-session","prompt":"How do I implement the OAuth connector?"}' \
	  | .claude/hooks/build-context-bundle.sh --type prompt
	@cat .claude-flow/learning/context_bundles/*_test-session.jsonl 2>/dev/null \
	 | jq -e 'select(.op=="prompt")' >/dev/null \
	   || { echo "FAIL: prompt entry not found in bundle"; exit 1; }
	@echo "    OK: User prompt in context bundle"
	@echo "==> Testing recall-memory.sh (memory + bundle recall)..."
	@RECALL_OUT=$$(echo '{"prompt":"How do hooks persist data to the memory database?"}' | .claude/hooks/recall-memory.sh 2>/dev/null); \
	 if echo "$$RECALL_OUT" | grep -qi "memory\|context"; then \
	   echo "    OK: recall-memory.sh returned results"; \
	 else \
	   echo "    WARN: recall-memory.sh returned no results (expected on fresh DB)"; \
	 fi
	@SKIP_OUT=$$(echo '{"prompt":"hi"}' | .claude/hooks/recall-memory.sh 2>/dev/null); \
	 if [ -z "$$SKIP_OUT" ]; then \
	   echo "    OK: Short prompts correctly skipped"; \
	 else \
	   echo "FAIL: Short prompt should produce no output"; exit 1; \
	 fi
	@echo "==> Cleaning up test artifacts..."
	@rm -rf .claude-flow/learning/hook_logs/test-session
	@rm -f .claude-flow/learning/context_bundles/*_test-session.jsonl
	@rm -f /tmp/pkb-verify-bundle-path
	@echo ""
	@echo "All checks passed."
