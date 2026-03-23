# PKB Implementation

You are implementing the PKB (Personal Knowledge Base) application per the specifications at `openspec/specs/`.

## Critical: Check for Repair Iteration

Before doing anything else, check for these files:
- `.ai/postmortem_latest.md` — If present, this is a **repair iteration**. Read the postmortem first and address its findings before making other changes. The postmortem contains the root cause analysis and specific repair guidance from the previous failed iteration.
- `.ai/unimplemented_specifications.md` — If present, lists specifications that were not yet implemented. Prioritize these.

If neither file exists, this is a fresh implementation. Proceed with the full specification.

## Read the Specification

Read ALL files under `openspec/specs/` to understand what to build:
- `openspec/specs/*/intent.md` — What each capability does and why
- `openspec/specs/*/contracts.md` — API boundaries and interface definitions
- `openspec/specs/*/constraints.md` — Non-negotiable invariants

The factory constraints at `openspec/specs/factory/constraints.md` define the development process you MUST follow.

## Project Architecture

PKB is a Go CLI application:

```
cmd/pkb/main.go          — CLI entry point (thin wrapper: main() calls run())
internal/
  auth/                   — OAuth flow (Google OAuth, token storage in XDG config)
  config/                 — Configuration loading (XDG paths, env vars)
  connectors/             — Source adapters:
    gdrive/               — Google Drive search
    gmail/                — Gmail search
    notion/               — Notion search
    obsidian/             — Obsidian vault search (via Google Drive)
    claude/               — Claude connector
  search/                 — Unified search engine across connectors
  server/                 — HTTP server (REST API, health endpoint)
  tui/                    — Terminal UI (bubbletea interactive search)
  web/                    — Web UI (embedded HTML/JS/CSS)
  apiclient/              — HTTP client for API calls
  tailscale/              — Tailscale integration for remote access
tests/
  acceptance/             — Black-box tests (build real binary, test as user)
  live/                   — Live API tests (requires credentials)
  e2e/                    — Playwright E2E tests (web UI)
```

## Development Process (Non-Negotiable)

1. **TDD**: Write a failing test FIRST (RED), minimum code to pass (GREEN), refactor (REFACTOR)
2. **Testing pyramid**: Unit tests (many, fast) → Acceptance tests (fewer, black-box)
3. **100% line coverage** on new code
4. **main() is thin**: It calls `run() error` — test `run()`, not `main()`
5. **No log.Fatal() or os.Exit()** outside main() — return errors
6. **All external interaction behind interfaces** — mock in tests
7. **Table-driven tests** for multiple cases
8. **Tests in same package** (_test.go alongside source)
9. **Race detection**: `go test -race ./...`

## Testing Stack

- `testing` — standard library (primary)
- `github.com/stretchr/testify/assert` — assertion helpers
- `github.com/stretchr/testify/mock` — mock generation
- Acceptance tests: `//go:build acceptance` tag, build real binary, test stdout/stderr/exit codes

## Key Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/charmbracelet/bubbles` — TUI components
- `github.com/charmbracelet/lipgloss` — TUI styling
- `golang.org/x/oauth2` — OAuth2 handling
- `google.golang.org/api` — Google APIs (Drive, Gmail)
- `github.com/stretchr/testify` — Testing

## Makefile Targets

All commands go through the Makefile:
- `make build` — Compile the binary
- `make test` — Unit tests with race detection
- `make test-accept` — Acceptance tests
- `make lint` — golangci-lint + actionlint
- `make vet` — go vet
- `make tidy` — go mod tidy with verification

## File Organization

NEVER save to root folder. Use:
- `cmd/pkb/` — CLI entry point
- `internal/` — Internal packages
- `tests/acceptance/` — Acceptance tests
- `docs/` — Documentation

## Acceptance Checks

Before reporting success, verify:
1. `make vet` passes
2. `make lint` passes
3. `make build` produces a working binary
4. `make test` passes with race detection
5. `make test-accept` passes
6. All new code has corresponding tests
7. No `log.Fatal()` or `os.Exit()` outside main()
8. All external dependencies are behind interfaces

## Status Reporting

When complete, write your status. If the environment variable `$KILROY_STAGE_STATUS_PATH` is set, write JSON to that path. If that write fails and `$KILROY_STAGE_STATUS_FALLBACK_PATH` is set, write to the fallback path instead.

```json
{"status": "success", "summary": "Implemented <description>"}
```

Or on failure:
```json
{"status": "fail", "summary": "<what failed>", "failure_class": "code_error"}
```

If neither path is set, print the status JSON to stdout as a last resort.
