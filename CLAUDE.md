# Claude Code Configuration

## MANDATORY: TDD + Pair Programming

ALL code in this project MUST be developed using strict TDD:
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

### Testing Pyramid (CRITICAL — read this before writing any test)

This project uses a testing pyramid. Each level has a different purpose:

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
- For this CLI app: build the real binary, run it with real arguments, check stdout/stderr/exit codes
- These tests do exactly what a human would do when following the README
- Use build tag `//go:build acceptance`
- Run with `go test -tags=acceptance ./tests/acceptance/`

### THE RULE FOR ACCEPTANCE TESTS:
**If a human follows the README and gets an error, an acceptance test MUST catch it.**

Acceptance tests must:
1. Build the actual binary (`go build`)
2. Execute it as a subprocess (`exec.Command("./pkb", "search", "query")`)
3. Check stdout, stderr, and exit code
4. NEVER import internal packages or call Go functions directly
5. Treat the application as a black box — the same way a user does

If you write a test that imports `internal/*` and calls Go functions, it is NOT
an acceptance test. It may be a useful unit or integration test, but it does not
verify that the software is usable from a human's perspective.

### When to write which:
- New internal function → unit test
- New connector/API integration → component integration test
- New CLI command or user-facing behavior → acceptance test
- Updating the README with new instructions → acceptance test that mirrors those instructions

### EPIC COMPLETION REQUIREMENTS

**MANDATORY: Before closing ANY epic that adds or modifies user-facing functionality:**

1. **Acceptance Test Checklist:**
   - [ ] Acceptance tests exist for ALL new CLI commands
   - [ ] Acceptance tests exist for ALL new HTTP endpoints
   - [ ] Acceptance tests exist for ALL new UI features
   - [ ] Tests build real binary and execute as subprocess
   - [ ] Tests check stdout/stderr/exit codes (black box)
   - [ ] Tests NEVER import internal packages
   - [ ] Tests mirror what README instructs users to do

2. **Test Coverage Verification:**
   - [ ] Run: `go test -tags=acceptance -v ./tests/acceptance/`
   - [ ] All new functionality tests pass
   - [ ] Tests would catch regression if functionality broke

3. **Documentation Alignment:**
   - [ ] Every README example has corresponding acceptance test
   - [ ] Test names reference README sections
   - [ ] Failure messages guide users to documentation

**THE RULE:**
> If a human follows the README and gets an error, an acceptance test MUST exist that would have caught it before merge.

**NO EXCEPTIONS:** Epics cannot be closed without these checks. If acceptance tests are blocked by external dependencies, file a blocker bead.

### Makefile (MANDATORY)

All developer-facing commands live in the `Makefile`. Run `make help` to discover them.

Rules:
- When adding a new tool, task, or workflow that a developer would run, add a Makefile target for it.
- When adding a new test category, add a `make test-*` target.
- The README must reference `make` targets, not raw `go` commands.
- CI workflows should call `make` targets where possible.
- Keep targets simple — each one should be one or two commands, not a script.

## Secrets and Credentials (MANDATORY)

**Agents MUST NEVER set, write, or modify CI/CD secrets.** This is non-negotiable:
1. NEVER use `gh secret set` or any equivalent to write secrets to CI
2. NEVER attempt to copy, read, or exfiltrate secrets from local `.env` files to CI
3. NEVER embed secrets in workflow files, commit messages, or any tracked files
4. If CI tests fail due to missing secrets, **ask the human** to set them manually
5. Provide the human with the URL to the settings page and the secret name(s) needed
6. The human is the ONLY person who enters secret values into CI — agents provide instructions only
7. If a secret appears corrupted or empty, tell the human and ask them to re-enter it
8. NEVER assume a secret is "not needed" — if CI requires it, it must be present and valid

**Why:** Agents previously corrupted CI secrets by attempting to set them programmatically. Secret values were redacted/truncated, causing all CI live tests to silently skip for multiple sessions. This wasted significant debugging time.

## CI Test Skipping (MANDATORY — NEVER ALLOWED)

**CI tests MUST NEVER be skipped due to missing secrets or credentials.** This is non-negotiable:
1. CI workflows MUST fail hard (exit 1) when required secrets are missing — never skip gracefully
2. NEVER write CI workflow steps with `if: has_secrets == 'true'` patterns that silently skip tests
3. NEVER treat missing credentials as a "skip" condition — it is always a hard failure
4. If you see a CI workflow that skips tests when secrets are missing, fix it to fail instead
5. The test code itself must also use `t.Fatal()` (not `t.Skip()`) for missing env vars
6. Every CI run must either PASS the live tests or FAIL visibly — silent skips are bugs

**Why:** Silent skipping caused live tests to never actually run on CI for multiple sessions, hiding real bugs. The obsidian subfolder search bug was merged as "working" because the live test that would have caught it was silently skipped.

## CI Pipeline Rules (MANDATORY)

**CI must be green.** This is non-negotiable:
1. NEVER push while CI is red on main (check: `gh run list --limit=1`)
2. **AFTER EVERY PUSH: Wait for CI and verify it passes.** Run `gh run watch` to monitor the triggered pipeline. Do NOT mark work as complete until CI is green. If `gh run watch` is unavailable, poll with `gh run list --branch main --limit=1` until the run completes.
3. If CI fails after your push, fix it before starting new work
4. If failure is infra/external (not your code), ask the human to verify
5. The human may allow temporary skip -- but YOU must ask, never assume
6. NEVER use `git commit --no-verify` to skip pre-commit hooks
7. All linting tools (golangci-lint, gitleaks, go vet, actionlint) belong in the SINGLE ci-cd.yml pipeline -- never create standalone workflow files for linting
8. The pipeline auto-creates a P0 bug bead on any unexpected failure on main -- agents will see it on next `bd sync` and must fix it before starting new work

## File Organization Rules

**NEVER save to root folder. Use these directories:**
- `cmd/pkb` - CLI entry point
- `internal/` - Internal packages (config, connectors, search, server, tui)
- `tests/acceptance/` - Acceptance tests
- `docs/` - Documentation

# Important Instruction Reminders
Do what has been asked; nothing more, nothing less.
NEVER create files unless they're absolutely necessary for achieving your goal.
ALWAYS prefer editing an existing file to creating a new one.
NEVER proactively create documentation files (*.md) or README files. Only create documentation files if explicitly requested by the User.
Never save working files, text/mds and tests to the root folder.

<!-- bv-agent-instructions-v1 -->

## Beads Viewer

This project uses [beads_viewer](https://github.com/Dicklesworthstone/beads_viewer) (`bv`) for issue tracking. Issues are stored in `.beads/` and tracked in git. The `bv` command launches a TUI viewer (avoid in automated sessions); use `bd` subcommands for CLI access. Full `bd` command reference and workflow are provided by the startup hook.

<!-- end-bv-agent-instructions -->
