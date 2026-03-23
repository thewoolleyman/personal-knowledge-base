## ADDED Requirements

### Requirement: Kilroy pipeline execution
The factory MUST have a Kilroy-compatible pipeline configuration that defines the full implement → verify → review loop for PKB's Go toolchain. The pipeline SHALL use the Attractor pattern with a linear (no-fanout) topology, deterministic tool gates for build verification, and LLM-driven nodes for implementation and review.

#### Scenario: Pipeline graph is valid
- **WHEN** Kilroy reads `factory/pipeline-config.yaml`
- **THEN** it generates a valid DOT graph with all nodes reachable from start and all paths terminating at exit or human_gate

#### Scenario: Factory run executes gate sequence
- **WHEN** a factory run starts from the implement node
- **THEN** it progresses through vet → lint → build → test → test-accept → review_final before reaching exit

### Requirement: Pipeline tool gates
The pipeline MUST define tool gates that invoke existing Makefile targets. Each gate SHALL have a tool_command, a timeout, and produce a deterministic pass/fail result.

#### Scenario: Vet gate runs go vet
- **WHEN** the verify_vet gate executes
- **THEN** it runs `make vet` and reports success if exit code is 0

#### Scenario: Lint gate runs golangci-lint
- **WHEN** the verify_lint gate executes
- **THEN** it runs `make lint` and reports success if exit code is 0

#### Scenario: Build gate compiles the binary
- **WHEN** the verify_build gate executes
- **THEN** it runs `make build` and reports success if exit code is 0

#### Scenario: Unit test gate runs tests with race detection
- **WHEN** the verify_test gate executes
- **THEN** it runs `make test` and reports success if exit code is 0

#### Scenario: Acceptance test gate runs user-perspective tests
- **WHEN** the verify_test_accept gate executes
- **THEN** it runs `make test-accept` and reports success if exit code is 0

### Requirement: Retry and error handling
The pipeline MUST retry on transient failures and escalate to a human gate after exhausting retries. The postmortem node MUST analyze failures and write repair guidance.

#### Scenario: Transient failure triggers retry
- **WHEN** a gate fails with a transient infrastructure error
- **THEN** the pipeline retries from the implement node up to 3 times

#### Scenario: Persistent failure routes to postmortem
- **WHEN** a gate fails with a non-transient error
- **THEN** the pipeline routes to the postmortem node for analysis

#### Scenario: Exhausted retries reach human gate
- **WHEN** the maximum retry count is exhausted
- **THEN** the pipeline presents a human gate with options to retry after manual fix or abort

### Requirement: Holdout scenario isolation
The factory MUST hide holdout scenarios from factory agents during implementation. The setup phase SHALL configure git sparse-checkout to exclude the `holdout-scenarios/` directory.

#### Scenario: Sparse checkout excludes holdout scenarios
- **WHEN** the factory run setup completes
- **THEN** the `holdout-scenarios/` directory is not visible in the factory agent's working tree

#### Scenario: Specification files remain visible
- **WHEN** the factory run setup completes
- **THEN** all files under `openspec/specs/` are visible to the factory agent

### Requirement: Factory agent prompts
The factory MUST provide agent prompts for each LLM-driven pipeline node: implement, review_final, postmortem, and human_gate. Each prompt SHALL be tailored to PKB's Go architecture and reference `openspec/specs/` as the specification source.

#### Scenario: Implement prompt directs agent to specifications
- **WHEN** the implement node executes
- **THEN** the agent prompt instructs reading `openspec/specs/` for requirements before writing code

#### Scenario: Implement prompt includes repair awareness
- **WHEN** a previous postmortem exists at `.ai/postmortem_latest.md`
- **THEN** the implement prompt instructs the agent to read and address the postmortem findings first

#### Scenario: Review prompt checks acceptance criteria
- **WHEN** the review_final node executes
- **THEN** the agent evaluates implementation against defined acceptance criteria and writes results to `.ai/review_final.md`

### Requirement: Execution configuration
The factory MUST have a `run.yaml` that configures Kilroy execution: repository path, CXDB connection, git branch management, and LLM provider settings.

#### Scenario: Run config specifies peer dependencies
- **WHEN** Kilroy reads `factory/run.yaml`
- **THEN** it finds CXDB connection details and LLM provider configuration

#### Scenario: Git commits per node
- **WHEN** a factory run progresses through nodes
- **THEN** each node completion creates a git commit on the run branch

### Requirement: Agent dispatch documentation
The repository MUST have an `AGENTS.md` at the root that orients agents to PKB's architecture, tech stack, specification structure, and path conventions.

#### Scenario: Agent reads dispatch guide
- **WHEN** an agent starts working on PKB
- **THEN** `AGENTS.md` provides the tech stack, directory structure, spec locations, and dependency information needed to orient

### Requirement: Factory operations guide
The repository MUST have a `docs/software-factory.md` that documents how to set up prerequisites, run the factory, check status, and land changes.

#### Scenario: Human follows factory guide
- **WHEN** a human reads `docs/software-factory.md`
- **THEN** they can set up Kilroy and CXDB, generate the pipeline, run a factory build, and land the result
