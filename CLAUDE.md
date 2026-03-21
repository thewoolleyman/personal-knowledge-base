# Claude Code Configuration

## Factory Specs (READ THESE FIRST)

This project uses the Software Factory pattern with OpenSpec. All rules, conventions, and constraints are specified in:

- **`openspec/specs/factory/intent.md`** — How the factory works, domain model, behavioral narratives
- **`openspec/specs/factory/contracts.md`** — Schema artifact flow, spec file conventions, capability list, repository structure
- **`openspec/specs/factory/constraints.md`** — TDD, testing pyramid, CI rules, secrets handling, file organization

Read the factory specs before starting work. They are the source of truth.

## OpenSpec Workflow

Changes flow through: propose → specs → holdout-scenarios + design → tasks → apply → verify → archive

Use `/opsx:propose` to start a new change. Use `/opsx:apply` to implement. The custom `software-factory` schema is the default.

## File Organization Rules

**NEVER save to root folder. Use these directories:**
- `cmd/pkb` - CLI entry point
- `internal/` - Internal packages
- `tests/acceptance/` - Acceptance tests
- `docs/` - Documentation
- `openspec/specs/` - Specifications
- `holdout-scenarios/` - Validation scenarios

# Important Instruction Reminders
Do what has been asked; nothing more, nothing less.
NEVER create files unless they're absolutely necessary for achieving your goal.
ALWAYS prefer editing an existing file to creating a new one.
NEVER proactively create documentation files (*.md) or README files. Only create documentation files if explicitly requested by the User.
Never save working files, text/mds and tests to the root folder.
