# Factory — Intent

The factory is the system that builds the system. It defines how changes flow from human intent to working software, what rules govern that process, and how the process itself evolves.

## Purpose

This project uses the [Software Factory Practitioners Guide](https://gitlab.com/cwoolley-gitlab/software-factory-practitioners-guide) pattern integrated with [OpenSpec](https://github.com/Fission-AI/OpenSpec) as the workflow engine.

- **OpenSpec** manages the lifecycle of changes: propose → specs → design → tasks → apply → verify → archive
- **The Software Factory pattern** governs repository structure and the human/machine boundary: humans express intent and verify correctness; machines implement

## Domain Model

### Capabilities
The unit of specification. Each capability represents a bounded domain of the system (e.g., `knowledge-retrieval`, `authentication`, `factory`). Capabilities are organized as directories under `openspec/specs/` with up to three files:

- **intent.md** — What the capability accomplishes and why
- **contracts.md** — API boundaries and interface definitions
- **constraints.md** — Non-negotiable invariants (SLOs, security, operational limits)

### Changes
The unit of work in OpenSpec. A change proposes modifications to one or more capabilities and flows through the artifact sequence defined by the schema.

### Holdout Scenarios
WHEN/THEN validation assertions that verify specs are correctly implemented. Authored alongside specs but invisible to implementation agents. Archived to `holdout-scenarios/` at the repo root.

### The Schema
The `software-factory` schema defines the artifact flow:
- `proposal` — why this change is needed
- `specs` — what the system should do (intent/contracts/constraints)
- `holdout-scenarios` — how to verify it (invisible to implementation)
- `design` — how to build it (architecture, decisions)
- `tasks` — implementation checklist

## Behavioral Narratives

### Proposing a change
A human (or agent) identifies something that needs to change. They create a proposal describing the motivation, the affected capabilities, and the impact. This is the seed — everything else flows from it.

### Specifying the change
From the proposal, specs are authored for each affected capability. Specs use the factory taxonomy: intent for purpose, contracts for interfaces, constraints for invariants. Each requirement has WHEN/THEN scenarios that define testable behavior.

### Verifying correctness
Holdout scenarios are authored after specs. They test the same requirements from different angles — edge cases, failure modes, integration boundaries. During implementation, the agent cannot see these scenarios. After implementation, a separate validation process runs them.

### Evolving the factory
The factory itself is a capability (`openspec/specs/factory/`). Significant changes to the build system — new schema artifacts, new conventions, new rules — go through the same propose → specs → implement workflow. Small tweaks (Makefile targets, CI adjustments) do not require proposals.
