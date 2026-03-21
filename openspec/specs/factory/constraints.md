# Factory — Constraints

These are non-negotiable invariants. They are not suggestions — they are hard boundaries that all implementation MUST satisfy.

## TDD

ALL code MUST be developed using strict Test-Driven Development:
1. Write a failing test FIRST (RED)
2. Write minimum code to pass (GREEN)
3. Refactor with tests passing (REFACTOR)
4. No code without a test. No exceptions.

Go testing conventions:
- Tests in same package (_test.go files alongside source)
- Interfaces for all external dependencies (mockable)
- Table-driven tests for multiple cases
- `go test -race ./...` before every commit
- 100% line coverage on new code — every line exists because a test demanded it
- `main()` is a thin wrapper calling `run() error` — test `run()`, not `main()`
- Never use `log.Fatal()` or `os.Exit()` outside of `main()` — return errors instead
- All OS/system/network interaction behind interfaces — mock in tests

Testing stack:
- `testing` — standard library (primary)
- `github.com/stretchr/testify/assert` — assertion helpers
- `github.com/stretchr/testify/mock` — mock generation for interfaces
- `go test -cover ./...` — coverage tracking
- `go test -race ./...` — race condition detection

## Testing Pyramid

**Unit tests (bottom — many, fast, isolated):**
- Test individual functions and methods with mocks
- Run with `go test ./...`
- Cover internal logic, edge cases, error handling

**Component integration tests (middle — fewer, test real interactions):**
- Test that real components work together (e.g., Connector + real API)
- Use build tag `//go:build integration`
- May require credentials, network access

**Acceptance tests (top — fewest, test from the USER'S perspective):**
- Test the ACTUAL USER EXPERIENCE of the application
- Build the real binary, run it with real arguments, check stdout/stderr/exit codes
- Use build tag `//go:build acceptance`
- Run with `go test -tags=acceptance ./tests/acceptance/`
- NEVER import internal packages or call Go functions directly

**The rule:** If a human follows the README and gets an error, an acceptance test MUST catch it.

### When to write which:
- New internal function → unit test
- New connector/API integration → component integration test
- New CLI command or user-facing behavior → acceptance test
- Updating the README with new instructions → acceptance test that mirrors those instructions

## Epic Completion Requirements

Before closing ANY epic that adds or modifies user-facing functionality:

1. Acceptance tests exist for ALL new CLI commands, HTTP endpoints, UI features
2. Tests build real binary and execute as subprocess (black box)
3. Tests NEVER import internal packages
4. Tests mirror what README instructs users to do
5. CI green, all child tasks complete, documentation updated

## Secrets and Credentials

Agents MUST NEVER set, write, or modify CI/CD secrets. This is non-negotiable:
1. NEVER use `gh secret set` or any equivalent to write secrets to CI
2. NEVER attempt to copy, read, or exfiltrate secrets from local `.env` files to CI
3. NEVER embed secrets in workflow files, commit messages, or any tracked files
4. If CI tests fail due to missing secrets, ask the human to set them manually
5. The human is the ONLY person who enters secret values into CI

**Why:** Agents previously corrupted CI secrets by attempting to set them programmatically, causing all CI live tests to silently skip for multiple sessions.

## CI Test Skipping — NEVER ALLOWED

CI tests MUST NEVER be skipped due to missing secrets or credentials:
1. CI workflows MUST fail hard (exit 1) when required secrets are missing
2. NEVER write CI workflow steps with `if: has_secrets == 'true'` patterns that silently skip tests
3. NEVER treat missing credentials as a "skip" condition — it is always a hard failure
4. Test code must use `t.Fatal()` (not `t.Skip()`) for missing env vars

**Why:** Silent skipping caused live tests to never actually run on CI for multiple sessions, hiding real bugs.

## CI Pipeline

CI must be green. This is non-negotiable:
1. NEVER push while CI is red on main
2. After every push: wait for CI and verify it passes
3. If CI fails after your push, fix it before starting new work
4. NEVER use `git commit --no-verify` to skip pre-commit hooks
5. All linting tools belong in the SINGLE ci-cd.yml pipeline

## Holdout Scenario Visibility

During implementation (/opsx:apply), agents MUST NOT read or reference holdout scenarios. They are invisible to implementation agents. During validation, a separate process evaluates the implementation against those scenarios.

## Makefile

All developer-facing commands live in the `Makefile`. Run `make help` to discover them.
- When adding a new tool, task, or workflow, add a Makefile target
- When adding a new test category, add a `make test-*` target
- The README must reference `make` targets, not raw `go` commands
- CI workflows should call `make` targets where possible

## File Organization

NEVER save to root folder. Use these directories:
- `cmd/pkb` — CLI entry point
- `internal/` — Internal packages
- `tests/acceptance/` — Acceptance tests
- `docs/` — Documentation
- `openspec/specs/` — Specifications
- `holdout-scenarios/` — Validation scenarios
