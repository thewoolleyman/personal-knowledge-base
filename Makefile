.PHONY: help build test test-accept test-int test-live test-e2e test-all lint lint-actions vet tidy clean run version scan-secrets scan-secrets-staged setup-hooks open-cicd-webpage serve serve-kill tailscale-health token-health deploy deploy-local deploy-status deploy-logs deploy-setup

BINARY := pkb
BUILD_DIR := .
VERSION ?= $(shell cat VERSION 2>/dev/null || echo dev)

# Cross-platform URL opener: macOS 'open', Linux 'xdg-open', fallback prints URL
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
  OPEN_CMD = open
else ifneq ($(shell command -v xdg-open 2>/dev/null),)
  OPEN_CMD = xdg-open
else
  OPEN_CMD = echo "Open this URL in your browser:"
endif

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

## test-live: Run live API tests (requires real Google credentials and token)
test-live:
	go test -tags=live -v -timeout=60s ./tests/live/

## test-e2e: Run Playwright E2E tests for the web UI (requires credentials + Playwright)
test-e2e: build
	cd tests/e2e && npx playwright test

## test-all: Run unit and acceptance tests (no live credentials required)
test-all: test test-accept

## test-full: Run unit, acceptance, and integration tests (requires OAuth token)
test-full: test test-accept test-int

## lint: Run golangci-lint and actionlint
lint: lint-actions
	golangci-lint run ./...

## lint-actions: Lint GitHub Actions workflow files (managed via mise)
lint-actions:
	mise x -- actionlint

## vet: Run go vet
vet:
	go vet ./...

## tidy: Tidy and verify go.mod
tidy:
	go mod tidy
	git diff --exit-code go.mod go.sum

## release-build: Cross-compile and package release artifacts for all platforms
release-build:
	scripts/build-release.sh

## release-validate: Validate release artifacts by installing and running the current-platform binary
release-validate:
	scripts/validate-release.sh

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist

## run: Build and run pkb with args (e.g. make run search "agentic")
ifeq (run,$(firstword $(MAKECMDGOALS)))
  RUN_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(RUN_ARGS):;@:)
endif
run: build
	./$(BINARY) $(RUN_ARGS)

## serve: Build, start the server, and open the web UI in the browser
serve: build
	./$(BINARY) serve & sleep 1 && $(OPEN_CMD) http://localhost:8080

## serve-kill: Kill any pkb process listening on port 8080
serve-kill:
	@KILLED=0; \
	for PID in $$(lsof -ti tcp:8080 2>/dev/null); do \
		CMD=$$(ps -p $$PID -o comm= 2>/dev/null); \
		case "$$CMD" in \
			*pkb*) kill $$PID && echo "Killed pkb (PID $$PID)" && KILLED=1;; \
		esac; \
	done; \
	if [ "$$KILLED" -eq 0 ]; then \
		echo "No pkb process found on port 8080"; \
	fi

## tailscale-health: Check the Tailscale health endpoint (run `make build && ./pkb serve` first)
tailscale-health:
	@IP=$$(tailscale ip -4 2>/dev/null) || { echo "Error: tailscale not running"; exit 1; }; \
	echo "Checking http://$$IP:9000/health ..."; \
	curl -sf "http://$$IP:9000/health" && echo "" || { echo "Error: server not responding. Run: make build && PKB_TAILSCALE=true ./pkb serve"; exit 1; }

## token-health: Check OAuth token health via the /health endpoint
token-health:
	@curl -sf http://localhost:9000/health | python3 -m json.tool 2>/dev/null || curl -sf http://localhost:8080/health | python3 -m json.tool 2>/dev/null || echo "Error: server not responding on port 9000 or 8080"

## open-cicd-webpage: Open the GitHub Actions CI/CD page in the default browser
open-cicd-webpage:
	$(OPEN_CMD) https://github.com/thewoolleyman/personal-knowledge-base/actions

## scan-secrets: Run gitleaks to detect hardcoded secrets (managed via mise)
scan-secrets:
	mise x -- gitleaks detect --source . --no-banner -c .gitleaks.toml --verbose

## scan-secrets-staged: Run gitleaks on staged files only (same check as pre-commit hook)
scan-secrets-staged:
	mise x -- gitleaks protect --staged --no-banner -c .gitleaks.toml --verbose

## setup-hooks: Install pre-commit hook with gitleaks secret scanning
setup-hooks:
	@echo "Installing pre-commit hook (gitleaks)..."
	@printf '%s\n' \
	  '#!/usr/bin/env sh' \
	  '#' \
	  '# Pre-commit hook: gitleaks secrets scan.' \
	  '#' \
	  '# Scans staged content for secrets. If detected, the commit is blocked.' \
	  '# If gitleaks is not installed, a warning is printed but the commit' \
	  '# proceeds (CI is the backstop).' \
	  '#' \
	  '# To reinstall: make setup-hooks' \
	  '' \
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
	  > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Done. Pre-commit hook installed at .git/hooks/pre-commit"

## deploy-local: Build, install to ~/.local/bin, install service file, and restart
deploy-local: build
	mkdir -p $(HOME)/.local/bin
	mkdir -p $(HOME)/.config/systemd/user
	systemctl --user stop pkb || true
	cp $(BINARY) $(HOME)/.local/bin/$(BINARY)
	cp deploy/pkb.service $(HOME)/.config/systemd/user/pkb.service
	systemctl --user daemon-reload
	systemctl --user enable pkb
	systemctl --user start pkb

## deploy: Alias for deploy-local
deploy: deploy-local

## deploy-status: Check systemd service status and health endpoint
deploy-status:
	@systemctl --user status pkb --no-pager || true
	@echo ""
	@echo "Health check:"
	@curl -sf http://localhost:9000/health && echo "" || echo "Server not responding"

## deploy-logs: Show recent service logs
deploy-logs:
	@journalctl --user -u pkb --no-pager -n 50

## deploy-setup: Install systemd service file and enable it (one-time setup)
deploy-setup:
	mkdir -p $(HOME)/.config/systemd/user
	cp deploy/pkb.service $(HOME)/.config/systemd/user/pkb.service
	systemctl --user daemon-reload
	systemctl --user enable pkb
	loginctl enable-linger $(USER)
	@echo "Service installed and enabled. Run 'make deploy' to build and start."
