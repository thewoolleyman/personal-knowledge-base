# Agent Dispatch Guide

This project uses **OpenSpec** for spec-driven development and **Kilroy** for factory pipeline execution. Read `CLAUDE.md` and the factory specs at `openspec/specs/factory/` before starting work.

## Purpose

PKB (Personal Knowledge Base) is a Go CLI application that provides unified search across personal knowledge sources — Obsidian, Google Drive, Gmail, Notion — with OAuth authentication, a terminal UI, web UI, and HTTP API.

## Tech Stack

- **Language**: Go 1.25
- **CLI framework**: spf13/cobra
- **TUI**: charmbracelet/bubbletea + bubbles + lipgloss
- **Web UI**: Embedded HTML/JS/CSS served by internal HTTP server
- **OAuth**: golang.org/x/oauth2 + Google APIs
- **Testing**: stdlib `testing` + testify/assert + testify/mock
- **Build**: Makefile (run `make help` to discover all targets)
- **CI**: GitHub Actions (`.github/workflows/ci-cd.yml`)
- **Factory**: Kilroy pipeline (`factory/`)

## Dependencies

- **Kilroy** (factory pipeline runner): `../kilroy`
- **CXDB** (factory observability): `../cxdb`
- Both are peer directories. Use relative paths, never hardcode absolute paths.

## Quick Reference

```bash
# Discover available work
openspec list --json                          # See active changes
openspec status --change "<name>" --json      # Check change status

# Build and test
make help                                     # All available targets
make build                                    # Compile binary
make test                                     # Unit tests with race detection
make test-accept                              # Acceptance tests
make lint                                     # golangci-lint + actionlint
```

## Specification Structure

All specifications live under `openspec/specs/`, organized by capability:

| Capability | Description |
|------------|-------------|
| `factory` | The build system itself (meta-capability) |
| `knowledge-retrieval` | Search, query, context assembly |
| `knowledge-ingestion` | Connectors, sync, indexing |
| `protocol-interfaces` | MCP, ACP, REST API, CLI |
| `connectors` | Individual source adapters |
| `authentication` | OAuth, API keys, access control |
| `infrastructure` | Config, networking, deployment |

Each capability directory has up to three files:
- `intent.md` — Purpose, domain model, behavioral narratives
- `contracts.md` — API boundaries, input/output shapes
- `constraints.md` — Non-negotiable invariants

## Internal Packages

| Package | Purpose |
|---------|---------|
| `cmd/pkb` | CLI entry point (main.go — thin wrapper) |
| `internal/auth` | OAuth flow handling (Google, token storage) |
| `internal/config` | Configuration loading (XDG paths, env vars) |
| `internal/connectors/gdrive` | Google Drive search adapter |
| `internal/connectors/gmail` | Gmail search adapter |
| `internal/connectors/notion` | Notion search adapter |
| `internal/connectors/obsidian` | Obsidian vault search (via Drive) |
| `internal/connectors/claude` | Claude connector |
| `internal/search` | Unified search engine |
| `internal/server` | HTTP server (REST API, health) |
| `internal/tui` | Terminal UI (bubbletea) |
| `internal/web` | Web UI (embedded static assets) |
| `internal/apiclient` | HTTP client wrapper |
| `internal/tailscale` | Tailscale integration |

## Test Suites

| Suite | Location | Build tag | Command |
|-------|----------|-----------|---------|
| Unit | `internal/**/*_test.go` | (none) | `make test` |
| Acceptance | `tests/acceptance/` | `acceptance` | `make test-accept` |
| Integration | `internal/**/*_test.go` | `integration` | `make test-int` |
| Live | `tests/live/` | `live` | `make test-live` |
| E2E | `tests/e2e/` | (Playwright) | `make test-e2e` |

## Path Conventions

- Use `.` (self) or `../sibling-repo` for relative paths
- Never hardcode absolute paths
- All files go in their designated directories (see `openspec/specs/factory/constraints.md`)
- Never save to the root folder

## Factory Pipeline

The factory is configured in `factory/`:
- `pipeline-config.yaml` — Pipeline graph definition
- `run.yaml` — Execution configuration
- `prompts/` — Agent prompts for each pipeline node
- `pipeline.dot` — Generated artifact (never edit directly)

See `docs/software-factory.md` for the full factory operations guide.

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create OpenSpec changes for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   git push
   git status  # MUST show "up to date with origin"
   ```
4. **Clean up** - Clear stashes, prune remote branches
5. **Verify** - All changes committed AND pushed
6. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
