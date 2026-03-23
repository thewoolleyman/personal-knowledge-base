## Holdout Scenarios for factory

<!-- Reference: specs/factory/intent.md -->

### Scenario: Pipeline config is parseable by Kilroy
**Verifies requirement:** Kilroy pipeline execution
- **WHEN** `kilroy validate factory/pipeline-config.yaml` is run
- **THEN** it exits 0 with no errors

### Scenario: Pipeline DOT generation succeeds
**Verifies requirement:** Kilroy pipeline execution
- **WHEN** `kilroy generate-pipeline` is run in the PKB repo
- **THEN** it produces a `pipeline.dot` file with nodes matching those defined in `pipeline-config.yaml`

### Scenario: All tool gate commands succeed on a clean checkout
**Verifies requirement:** Pipeline tool gates
- **WHEN** each tool gate command from `pipeline-config.yaml` is run on a clean checkout of main
- **THEN** every gate exits 0 (since the existing codebase already passes all checks)

### Scenario: Lint gate catches a real lint error
**Verifies requirement:** Pipeline tool gates
- **WHEN** a Go file with a lint violation (e.g., unused variable) is present
- **THEN** the verify_lint gate exits non-zero

### Scenario: Sparse checkout is idempotent
**Verifies requirement:** Holdout scenario isolation
- **WHEN** the setup command from `run.yaml` is run twice in succession
- **THEN** the second run produces the same result without errors

### Scenario: Implement prompt references all spec capability directories
**Verifies requirement:** Factory agent prompts
- **WHEN** the implement prompt at `factory/prompts/implement.md` is read
- **THEN** it instructs the agent to read specifications from `openspec/specs/`

### Scenario: Postmortem writes to expected path
**Verifies requirement:** Retry and error handling
- **WHEN** the postmortem prompt is followed by an agent
- **THEN** the output is written to `.ai/postmortem_latest.md`

### Scenario: Review prompt produces structured output
**Verifies requirement:** Factory agent prompts
- **WHEN** the review_final prompt is followed by an agent
- **THEN** the output at `.ai/review_final.md` contains pass/fail for each acceptance criterion

### Scenario: Run config has correct peer directory conventions
**Verifies requirement:** Execution configuration
- **WHEN** `factory/run.yaml` is read
- **THEN** it uses relative paths (`.` for repo, `../kilroy` or `../cxdb` patterns) and never hardcodes absolute paths

### Scenario: AGENTS.md covers all internal packages
**Verifies requirement:** Agent dispatch documentation
- **WHEN** `AGENTS.md` is read
- **THEN** it references the key internal packages (auth, config, connectors, search, server, tui, web) and their purposes

### Scenario: Factory guide includes prerequisite setup steps
**Verifies requirement:** Factory operations guide
- **WHEN** `docs/software-factory.md` is read
- **THEN** it includes steps for building Kilroy, starting CXDB, and verifying both are ready before running the factory

### Scenario: Human gate prompt directs to diagnostics
**Verifies requirement:** Retry and error handling
- **WHEN** the human_gate prompt is displayed
- **THEN** it tells the operator to check pipeline status and the postmortem file before choosing retry or abort
