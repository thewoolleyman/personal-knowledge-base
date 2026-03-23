## Context

PKB is a Go CLI application with 7 capability domains, a testing pyramid (unit/acceptance/integration/live/e2e), and CI via GitHub Actions. The project has factory specs defining the OpenSpec workflow and conventions, but no execution infrastructure to run autonomous factory builds. The reference implementation is cxdb-graph-ui, which uses Kilroy (a Go implementation of the Attractor pattern) with CXDB for observability.

Kilroy and CXDB both exist as peer directories (`../kilroy`, `../cxdb`). Both must be synced to latest upstream before the factory can run.

## Goals / Non-Goals

**Goals:**
- PKB has a complete `factory/` directory that Kilroy can execute
- Pipeline gates map to existing Makefile targets (no new test infrastructure)
- Factory agents receive PKB's specification and can implement against it
- Holdout scenarios are hidden from factory agents via sparse-checkout
- A human can run `kilroy run` and the factory executes the full implement → gate → review loop
- Factory commits push through existing GitHub Actions CI

**Non-Goals:**
- Backfilling capability specifications (that's Phase 2+ vertical slices)
- Writing holdout scenarios (that's Phase 2+)
- Modifying existing application code
- Changing the GitHub Actions CI pipeline
- Automating the factory trigger (manual `kilroy run` for now)
- E2E or live tests as pipeline gates (these require credentials the factory agent won't have)

## Decisions

### D1: Pipeline topology — linear with retry (no-fanout)

Follow cxdb-graph-ui's `no-fanout` topology. PKB's gate sequence is simpler than cxdb-graph-ui (Go toolchain, no Rust+frontend dual build), so a linear pipeline is appropriate.

**Pipeline node sequence:**
```
start → implement → check_implement
  → verify_vet → check_vet
  → verify_lint → check_lint
  → verify_build → check_build
  → verify_test → check_test
  → verify_test_accept → check_test_accept
  → review_final → exit
```

With `postmortem` and `human_gate` as error-handling nodes.

**Why linear over fanout:** All gates are sequential dependencies (can't lint what doesn't vet, can't test what doesn't build). No parallelism opportunity.

**Alternative considered:** Including `verify_fmt` as a separate gate. Rejected because `golangci-lint` already catches formatting issues, and Go's `gofmt` is typically run as a pre-save editor action rather than a pipeline gate. If needed later, it's a one-node addition.

### D2: Tool gate commands use `make` targets

Pipeline tool gates call existing Makefile targets rather than raw `go` commands. This follows the factory constraint that "all developer-facing commands live in the Makefile" and ensures the factory runs the same commands as CI and developers.

**Gate mapping:**
| Gate ID | Make target | Timeout |
|---------|-------------|---------|
| `verify_vet` | `make vet` | 60s |
| `verify_lint` | `make lint` | 120s |
| `verify_build` | `make build` | 120s |
| `verify_test` | `make test` | 120s |
| `verify_test_accept` | `make test-accept` | 120s |

**Not included as gates:**
- `make test-int` — requires Google Drive credentials
- `make test-live` — requires live API credentials
- `make test-e2e` — requires Playwright + credentials
- `make scan-secrets` — run by pre-commit hook, not a pipeline gate

These are validated by CI after the factory pushes, not during the factory run.

### D3: Model selection — Sonnet for implementation, Sonnet for review

Start with `claude-sonnet-4-6` for all nodes (matching cxdb-graph-ui). This keeps costs predictable. The model stylesheet can be upgraded per-node later if quality demands it (e.g., Opus for review_final).

**Alternative considered:** Opus for implement, Sonnet for gates. Rejected for Phase 1 — start simple, upgrade based on observed quality.

### D4: Sparse-checkout to hide holdout scenarios

Follow cxdb-graph-ui's exact approach: the `setup` command in `run.yaml` writes a sparse-checkout pattern that excludes `holdout-scenarios/` from the factory agent's working tree. This is the practitioners guide's recommended mechanism for the holdout split.

### D5: Implement prompt structure — spec-driven with repair awareness

The implement prompt will:
1. Direct the agent to read `openspec/specs/` for requirements
2. Check for `.ai/postmortem_latest.md` (repair iteration awareness)
3. List the Go project structure and conventions
4. Include acceptance criteria derived from factory constraints (TDD, testing pyramid, coverage)

This mirrors cxdb-graph-ui's implement prompt but adapted for Go toolchain.

### D6: Review prompt structure — acceptance criteria checklist

The review_final prompt will define acceptance criteria (AC-1, AC-2, etc.) that the reviewer agent checks. These will cover:
- TDD compliance (tests exist for all new code)
- Testing pyramid compliance (unit + acceptance for user-facing features)
- Factory constraints (no log.Fatal outside main, interfaces for dependencies, etc.)
- Build/test passing

The reviewer writes `.ai/review_final.md` with pass/fail per criterion.

### D7: Git configuration — commit per node, run branch prefix

Follow cxdb-graph-ui: `commit_per_node: true`, `run_branch_prefix: attractor/run`. Factory runs create a branch like `attractor/run/<timestamp>`, commit after each node, and the human decides when to land (merge to main) or discard.

### D8: AGENTS.md — agent dispatch guide

Create an `AGENTS.md` at the repo root that orients agents to:
- PKB's tech stack (Go, Cobra CLI, bubbletea TUI, web UI)
- Dependency locations (Kilroy, CXDB as peer directories)
- Specification structure (`openspec/specs/`)
- Path conventions (never hardcode absolute paths)
- Available Make targets

### D9: docs/software-factory.md — PKB-specific factory guide

Create a factory guide for humans, documenting:
- How to set up prerequisites (Kilroy, CXDB)
- How to generate the pipeline DOT file
- How to run the factory
- How to check status, resume, land changes
- The relationship between factory runs and CI

## Risks / Trade-offs

**[Risk] Implement prompt may be too generic for first run** → The implement prompt describes the full PKB architecture, but without backfilled specs, the factory agent has limited specification to implement against. Mitigation: Phase 1 establishes infrastructure; the factory becomes useful when Phase 2 adds capability specs.

**[Risk] Gate timeouts may need tuning** → Initial timeouts are estimates based on local build times. Mitigation: Kilroy supports per-gate timeout configuration; adjust after first factory run.

**[Risk] Kilroy API/config format may have changed** → The reference is cxdb-graph-ui's config, but Kilroy is actively developed. Mitigation: Sync Kilroy fork to upstream before implementing; validate config against current Kilroy version.

**[Trade-off] No credential-requiring gates in pipeline** → Live tests, integration tests, and E2E tests are excluded from factory gates because the factory agent doesn't have credentials. These are caught by CI after the factory pushes. Accepted trade-off: the factory validates what it can locally; CI catches the rest.
