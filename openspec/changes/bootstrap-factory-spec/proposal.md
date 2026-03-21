## Why

This project is migrating from Claude Flow + Beads to OpenSpec with a Software Factory pattern. The factory — the system that builds the system — needs its own specification so that the rules, schema, artifact flow, and conventions are captured as durable, evolvable specs rather than ad-hoc configuration scattered across CLAUDE.md and settings files. Without this, the factory's design decisions exist only in conversation history and will be lost or contradicted over time.

## What Changes

- Create the `factory` capability with intent, contracts, and constraints specs
- Customize the `software-factory` OpenSpec schema to add `holdout-scenarios` as a first-class artifact
- Establish the spec file convention: domain-first organization with `intent.md`, `contracts.md`, `constraints.md` per capability (instead of a single `spec.md`)
- Migrate hard-won rules from CLAUDE.md into `factory/constraints.md` as proper spec content
- Add `holdout-scenarios/` directory at the repo root for archived holdout scenarios
- Scaffold empty spec files for the six product capabilities

## Capabilities

### New Capabilities
- `factory`: The meta-capability — specifies how the software factory operates, including the OpenSpec schema, artifact flow, spec file conventions, holdout scenario management, CI/CD rules, TDD requirements, and agent behavior constraints

### Modified Capabilities

(none — no existing specs yet)

## Impact

- `openspec/schemas/software-factory/schema.yaml` — add holdout-scenarios artifact, update specs artifact instructions for intent/contracts/constraints convention
- `openspec/specs/factory/` — new capability with intent.md, contracts.md, constraints.md
- `openspec/specs/` — scaffold empty files for six product capabilities (knowledge-retrieval, knowledge-ingestion, protocol-interfaces, connectors, authentication, infrastructure)
- `holdout-scenarios/` — new directory at repo root
- `CLAUDE.md` — slim down to a pointer to factory specs, keep only agent-facing directives that must be in CLAUDE.md
