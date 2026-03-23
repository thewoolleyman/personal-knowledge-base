# PKB Software Factory Guide

## Architecture

PKB uses the Software Factory pattern described in the [Software Factory Practitioners Guide](https://gitlab.com/cwoolley-gitlab/software-factory-practitioners-guide):

- **Humans** write specifications (intent, contracts, constraints) and holdout scenarios
- **Kilroy** orchestrates the factory pipeline — a graph of agent nodes and deterministic tool gates
- **Factory agents** implement code from specification, never seeing holdout scenarios
- **Validation agents** evaluate implementation against holdout scenarios, never seeing code
- **CI/CD** (GitHub Actions) runs after factory commits are pushed, providing the final quality gate

```
Specifications (human)
        │
        ▼
┌──────────────────────────────────────────────┐
│  Kilroy Pipeline                             │
│                                              │
│  implement → vet → lint → build → test →     │
│  test-accept → review_final → exit           │
│                                              │
│  (postmortem + human_gate on failure)        │
└──────────────────────────────────────────────┘
        │
        ▼
  Git push → GitHub Actions CI → Deploy
```

### Three Validation Tiers

1. **Deterministic tool gates** (strongest): `make vet`, `make lint`, `make build`, `make test`, `make test-accept`
2. **LLM semantic review**: Independent review agent checks specification fidelity
3. **Postmortem loop**: On failure, analysis agent diagnoses root cause and guides repair

### Holdout Scenario Isolation

During factory runs, `holdout-scenarios/` is hidden from factory agents via git sparse-checkout. This prevents implementation from being optimized for visible tests rather than genuine specification compliance.

## Prerequisites

### Kilroy

Kilroy is the pipeline runner. It lives at `../kilroy`.

```bash
cd ../kilroy
go build ./cmd/kilroy
# Verify it runs:
./kilroy --help
```

### CXDB (Optional — for observability)

CXDB provides real-time pipeline execution observability. It lives at `../cxdb`.

```bash
# Start CXDB (binary protocol on :9109, HTTP API on :9110)
cd ../cxdb
# Follow CXDB setup instructions in its README
```

CXDB is optional — the factory runs without it, but you won't see real-time execution status.

## Pipeline Configuration

Pipeline configuration lives in `factory/`:

```
factory/
├── pipeline-config.yaml   # Pipeline graph (nodes, edges, gates, model config)
├── run.yaml               # Execution config (repo, CXDB, git, LLM provider)
├── prompts/
│   ├── implement.md       # Core implementation prompt
│   ├── review_final.md    # Semantic review prompt
│   ├── postmortem.md      # Failure analysis prompt
│   └── human_gate.md      # Human decision prompt
```

`pipeline.dot` is a **generated artifact** — never edit it directly. It's produced from `pipeline-config.yaml`.

## Running the Factory

### 1. Generate the pipeline DOT file

```bash
cd /path/to/personal-knowledge-base
../kilroy/kilroy generate-pipeline
```

This reads `factory/pipeline-config.yaml` and writes `pipeline.dot`.

### 2. Start a factory run

```bash
../kilroy/kilroy run
```

This:
- Creates a run branch (`attractor/run/<timestamp>`)
- Hides `holdout-scenarios/` via sparse-checkout
- Executes the pipeline graph from `start` to `exit`
- Commits after each node

### 3. Check status

```bash
../kilroy/kilroy status
```

For verbose output with stage traces:
```bash
../kilroy/kilroy status --verbose
```

### 4. Resume a paused run

If the pipeline paused at a human gate:
```bash
../kilroy/kilroy resume
```

### 5. Land the result

After a successful factory run, merge the run branch to main:
```bash
git checkout main
git merge attractor/run/<branch-name>
git push
```

Then verify CI passes on the push.

## Relationship to CI

The factory and CI are complementary:

- **Factory** validates what it can locally: vet, lint, build, unit tests, acceptance tests
- **CI** (GitHub Actions) runs after push: same gates plus live tests, E2E tests, secret scanning, and deployment
- Factory commits go through the same CI pipeline as human commits
- If CI fails after a factory push, the factory run should be revised

### Gates not in the factory pipeline

These require credentials or infrastructure the factory agent doesn't have:
- `make test-int` — Component integration tests (requires Google Drive credentials)
- `make test-live` — Live API tests (requires real credentials and token)
- `make test-e2e` — Playwright E2E tests (requires credentials + browser)
- `make scan-secrets` — Gitleaks secret scanning (run by pre-commit hook)

## Known Limitations

- **Spec backfill in progress**: Not all capabilities have complete specifications yet. The factory becomes more effective as specs are backfilled (Phase 2 vertical slices).
- **No automated trigger**: Factory runs are manually initiated with `kilroy run`.
- **Holdout validation requires separate run**: After implementation, holdout scenarios must be validated in a separate process.
- **Credential-requiring tests**: Live, integration, and E2E tests are only validated by CI, not the factory.
