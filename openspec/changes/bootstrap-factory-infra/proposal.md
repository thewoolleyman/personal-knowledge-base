## Why

The PKB project has factory specs (intent, contracts, constraints) and the OpenSpec workflow engine, but lacks the execution infrastructure to actually run as a software factory. Without a Kilroy pipeline, agent prompts, and tool gates, changes still require manual human implementation rather than autonomous factory runs. This change bootstraps the factory execution layer so that subsequent vertical-slice changes can backfill specifications and holdout scenarios, with each slice immediately testable by running the factory end-to-end.

## What Changes

- Add `factory/pipeline-config.yaml` defining the Kilroy pipeline graph for PKB's Go toolchain (implement → fmt → vet → lint → test → test-accept → review_final → exit, with retry loops and human gate fallback)
- Add `factory/run.yaml` with execution config (repo path, git settings, CXDB connection, sparse-checkout to hide `holdout-scenarios/` from factory agents)
- Add `factory/prompts/implement.md` — core implementation prompt tailored to PKB's Go architecture (CLI, internal packages, connectors, server, TUI, web UI)
- Add `factory/prompts/review_final.md` — semantic review prompt verifying fidelity to PKB specs
- Add `factory/prompts/postmortem.md` — failure diagnosis and repair guidance prompt
- Add `factory/prompts/human_gate.md` — human decision point when retries are exhausted
- Add `AGENTS.md` — agent dispatch guide documenting PKB's tech stack, dependencies (Kilroy, CXDB), spec structure, and path conventions
- Add `docs/software-factory.md` — PKB-specific factory guide explaining the architecture, how to run the pipeline, skills reference, and known limitations
- Wire existing Makefile targets (`fmt`, `vet`, `lint`, `test`, `test-accept`) as pipeline tool gates

## Capabilities

### New Capabilities

_(none — no new application capabilities are introduced)_

### Modified Capabilities

- `factory`: Adding execution infrastructure (pipeline config, agent prompts, run config) to the existing factory meta-capability. The factory intent, contracts, and constraints specs may need minor updates to reference the new Kilroy integration.

## Impact

- **New files**: `factory/` directory (6 files), `AGENTS.md`, `docs/software-factory.md`
- **Dependencies**: Requires Kilroy CLI (peer directory `../kilroy`) and CXDB (peer directory `../cxdb`) to be up-to-date before the factory can execute
- **CI/CD**: No changes to existing GitHub Actions pipeline. Factory runs are a separate execution path; commits pushed by the factory go through the same CI as human commits
- **Existing code**: No application code changes. This is pure infrastructure addition
