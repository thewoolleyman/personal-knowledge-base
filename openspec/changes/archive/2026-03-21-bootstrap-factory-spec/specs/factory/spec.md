## ADDED Requirements

### Requirement: Software factory pattern with OpenSpec workflow
The factory SHALL use the Software Factory Practitioners Guide pattern integrated with OpenSpec as the workflow engine. OpenSpec manages the lifecycle of changes (propose → specs → design → tasks → apply → verify → archive). The Software Factory pattern governs repository structure and the human/machine boundary.

#### Scenario: New feature change
- **WHEN** a developer proposes a new feature
- **THEN** the change flows through proposal → specs → holdout-scenarios/design → tasks → apply → verify → archive

#### Scenario: Factory configuration change
- **WHEN** a significant change to the build system itself is needed
- **THEN** it is proposed as a change against the `factory` capability using the same OpenSpec workflow

### Requirement: Domain-first capability organization with factory taxonomy
Specs SHALL be organized by capability (domain-first), with the Software Factory taxonomy (intent, contracts, constraints) as files within each capability directory. The capability directories are: `knowledge-retrieval`, `knowledge-ingestion`, `protocol-interfaces`, `connectors`, `authentication`, `infrastructure`, and `factory`.

#### Scenario: New capability spec structure
- **WHEN** a new capability is created
- **THEN** its directory SHALL contain `intent.md`, `contracts.md`, and/or `constraints.md` as appropriate — not every capability requires all three files

#### Scenario: Capability discovery
- **WHEN** an agent needs to understand a capability
- **THEN** it reads all files in `openspec/specs/<capability>/` to get the full picture of intent, contracts, and constraints

### Requirement: Custom software-factory schema
The project SHALL use a custom OpenSpec schema named `software-factory` (forked from `spec-driven`) that defines the artifact flow: proposal → specs → holdout-scenarios + design → tasks → apply.

#### Scenario: Schema defines holdout-scenarios artifact
- **WHEN** the schema is loaded
- **THEN** it SHALL include a `holdout-scenarios` artifact that requires `specs` and is NOT required by `tasks`

#### Scenario: Schema artifact dependency graph
- **WHEN** a change is created
- **THEN** the artifact dependencies SHALL be: proposal has no dependencies; specs requires proposal; holdout-scenarios requires specs; design requires proposal; tasks requires specs and design; apply requires tasks

### Requirement: Holdout scenarios as first-class artifact
Holdout scenarios SHALL be managed through the OpenSpec changes lifecycle. They are authored as part of a change, then synced/archived to `holdout-scenarios/` at the repo root when the change is synced/archived.

#### Scenario: Holdout scenario authoring
- **WHEN** specs are complete for a change
- **THEN** holdout scenarios MAY be authored as WHEN/THEN scenarios that verify the specs

#### Scenario: Holdout scenario archival
- **WHEN** a change is archived
- **THEN** holdout scenarios from the change SHALL be merged into `holdout-scenarios/` at the repo root

#### Scenario: Holdout scenario visibility during implementation
- **WHEN** an agent is implementing tasks (during /opsx:apply)
- **THEN** holdout scenarios MUST NOT be referenced — they are invisible to implementation agents

### Requirement: Intent file convention
Each capability's `intent.md` SHALL describe the capability's purpose, domain model, behavioral narratives, and key decisions. This corresponds to the Software Factory guide's `spec/intent/` concept.

#### Scenario: Intent file content
- **WHEN** an `intent.md` is authored
- **THEN** it SHALL contain natural-language specification describing what the capability accomplishes and why, not how

### Requirement: Contracts file convention
Each capability's `contracts.md` SHALL define the capability's API boundaries, input/output shapes, and protocol definitions. This corresponds to the Software Factory guide's `spec/contracts/` concept.

#### Scenario: Contracts file content
- **WHEN** a `contracts.md` is authored
- **THEN** it SHALL define the capability's upstream and downstream interfaces precisely enough to be implemented without ambiguity

### Requirement: Constraints file convention
Each capability's `constraints.md` SHALL capture non-negotiable invariants: SLOs, security requirements, operational limits, and immutable principles. This corresponds to the Software Factory guide's `spec/constraints/` concept.

#### Scenario: Constraints file content
- **WHEN** a `constraints.md` is authored
- **THEN** it SHALL contain hard boundaries that are not suggestions — they are invariants the implementation MUST satisfy

### Requirement: TDD is mandatory for all implementation
ALL code in this project MUST be developed using strict Test-Driven Development. No code without a test. No exceptions.

#### Scenario: New function implementation
- **WHEN** a new function is implemented during /opsx:apply
- **THEN** a failing test SHALL be written first (RED), minimum code written to pass (GREEN), then refactored with tests passing (REFACTOR)

#### Scenario: Test coverage
- **WHEN** new code is added
- **THEN** 100% line coverage SHALL be maintained — every line exists because a test demanded it

### Requirement: Testing pyramid enforcement
The project SHALL use a three-level testing pyramid: unit tests (many, fast, isolated), component integration tests (fewer, real interactions), and acceptance tests (fewest, user perspective).

#### Scenario: New CLI command
- **WHEN** a new CLI command or user-facing behavior is added
- **THEN** an acceptance test SHALL exist that builds the real binary, executes it as a subprocess, and checks stdout/stderr/exit codes as a black box

#### Scenario: New internal function
- **WHEN** a new internal function is added
- **THEN** a unit test SHALL exist with mocks for external dependencies

### Requirement: CI pipeline must be green
CI MUST pass on every push. CI failures are hard failures, never silently skipped.

#### Scenario: Missing secrets in CI
- **WHEN** CI tests require secrets that are not configured
- **THEN** the pipeline SHALL fail hard (exit 1), never skip gracefully

#### Scenario: CI failure on main
- **WHEN** CI fails on the main branch
- **THEN** the failure MUST be fixed before any new work begins

### Requirement: Agents must never manage secrets
Agents SHALL never set, write, or modify CI/CD secrets. Only the human enters secret values.

#### Scenario: CI needs new secret
- **WHEN** a CI test requires a new secret
- **THEN** the agent SHALL provide the human with the settings page URL and secret name, and the human enters the value manually
