## Context

This project is migrating from Claude Flow + Beads to OpenSpec with a Software Factory pattern. Claude Flow has been removed. Beads still has 7 open issues (5 auto-created CI failures, 2 real tasks). OpenSpec is installed with the expanded workflow profile (all 11 commands). A `software-factory` schema has been forked from `spec-driven` but not yet customized.

The project is a Go CLI app (personal knowledge base) that will also be consumed agentically via MCP, ACP, and CLI. The spec structure must serve both as generation input and as documentation for agent consumers.

## Goals / Non-Goals

**Goals:**
- Customize the `software-factory` schema to add holdout-scenarios artifact and update instructions for intent/contracts/constraints file convention
- Create the `factory` capability spec with content from this change's specs
- Scaffold empty capability directories for the six product domains
- Create `holdout-scenarios/` at the repo root
- Slim down CLAUDE.md to reference factory specs instead of inlining all rules

**Non-Goals:**
- Populating product capability specs with content (separate future changes)
- Migrating beads history into OpenSpec (separate future change)
- Implementing holdout scenario hiding/access control (future concern)
- Fixing OpenSpec CLI discovery for custom file names (tracked upstream: Issue #666)

## Decisions

### Decision 1: Domain-first over type-first spec organization
**Choice:** `openspec/specs/<capability>/intent.md` instead of `openspec/specs/intent/<capability>.md`
**Why:** Agents work on one capability at a time and need the full picture in one directory. OpenSpec's tooling expects `specs/<name>/` directories. Fighting the convention means fighting the workflow.
**Alternative considered:** Type-first (matching the factory guide's `spec/intent/`, `spec/contracts/`, `spec/constraints/`) — rejected because it scatters related specs across directories and conflicts with OpenSpec's discovery pattern.

### Decision 2: Factory taxonomy as files, not sections
**Choice:** Separate `intent.md`, `contracts.md`, `constraints.md` files per capability
**Why:** Each file type has a distinct purpose and stability profile. Constraints rarely change; intent evolves; contracts are precise API definitions. Separate files make diffs cleaner and allow creating only the files that are relevant per capability.
**Alternative considered:** Single `spec.md` with intent/contracts/constraints as sections — would work with OpenSpec's hardcoded CLI discovery but loses the structural clarity. Rejected in favor of the factory pattern's rigor, accepting the CLI limitation (Issue #666).

### Decision 3: Holdout scenarios follow the changes lifecycle
**Choice:** Authored in `openspec/changes/<name>/holdout-scenarios/`, archived to `holdout-scenarios/` at repo root
**Why:** Consistent with how OpenSpec manages delta specs — changes are proposed, iterated, then synced/archived to the canonical location. Holdout scenarios are durable WHEN/THEN assertions, siblings of specs, not design.
**Alternative considered:** Write directly to `holdout-scenarios/` at repo root — simpler but bypasses the change review workflow.

### Decision 4: Factory is a capability, not a separate OpenSpec instance
**Choice:** `openspec/specs/factory/` alongside product capabilities
**Why:** Avoids operational overhead of managing two OpenSpec instances. Factory changes that are significant enough to warrant spec discipline use the same workflow. Small tweaks (Makefile targets, CI adjustments) don't need proposals.
**Alternative considered:** Separate OpenSpec installation in `factory/openspec/` — clean separation but too heavy for our scale.

### Decision 5: Accept CLI discovery limitation for now
**Choice:** Use custom file names (intent.md, contracts.md, constraints.md) and accept that `openspec spec list/show/validate` won't discover them
**Why:** The workflow commands (/opsx:apply, /opsx:sync, /opsx:verify) use the schema's `generates` glob which matches `**/*.md`. The CLI commands are convenience tools. Upstream Issue #666 tracks the fix.
**Alternative considered:** Use `spec.md` everywhere to stay compatible — rejected because it sacrifices the factory pattern's structural clarity for a minor CLI convenience.

## Risks / Trade-offs

- [OpenSpec CLI commands won't discover our specs] → Mitigation: workflow commands work via glob; watch Issue #666 for upstream fix
- [Custom schema may break on OpenSpec upgrades] → Mitigation: schema is version-controlled and forked locally; test after upgrades
- [Seven capabilities may be too many or too few] → Mitigation: capabilities can be added/removed/merged as the project evolves; start with the current understanding and iterate
- [CLAUDE.md slimming may lose context for agents] → Mitigation: CLAUDE.md will reference factory specs explicitly; agents read both
