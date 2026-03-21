# Factory — Contracts

## Schema Artifact Flow

The `software-factory` schema defines these artifacts and their dependencies:

```
proposal (no dependencies)
├── specs (requires: proposal)
│     ├── holdout-scenarios (requires: specs)
│     └── design (requires: proposal)
│           └── tasks (requires: specs, design)
│                 └── [apply] (requires: tasks)
```

Holdout scenarios and design are siblings — both flow from specs/proposal respectively, neither depends on the other. Tasks depend on specs and design, NOT on holdout scenarios.

## Spec File Convention

Each capability directory under `openspec/specs/<capability>/` contains:

| File | Purpose | Corresponds to |
|------|---------|---------------|
| `intent.md` | Purpose, domain model, behavioral narratives | Factory guide: `spec/intent/` |
| `contracts.md` | API boundaries, input/output shapes, protocols | Factory guide: `spec/contracts/` |
| `constraints.md` | SLOs, security requirements, invariants | Factory guide: `spec/constraints/` |

Not every capability requires all three files. Create only what is relevant.

## Capability List

| Capability | Description |
|------------|-------------|
| `knowledge-retrieval` | Search, query, context assembly — the core value for agent consumers |
| `knowledge-ingestion` | Connectors, sync, indexing — how knowledge enters the system |
| `protocol-interfaces` | MCP, ACP, REST API, CLI — how consumers talk to the system |
| `connectors` | Individual source adapters (Obsidian, Google, etc.) |
| `authentication` | OAuth, API keys, agent identity, access control |
| `infrastructure` | Config, networking, deployment, Tailscale |
| `factory` | The meta-capability — the build system itself |

## Holdout Scenario Convention

Holdout scenarios are organized by capability, mirroring the specs structure:

```
holdout-scenarios/
├── knowledge-retrieval/
│   └── scenarios.md
├── authentication/
│   └── scenarios.md
└── ...
```

During a change, holdout scenarios live in `openspec/changes/<name>/holdout-scenarios/<capability>/scenarios.md` and are archived to the repo root `holdout-scenarios/` directory when the change is archived.

## Repository Structure

```
project-root/
├── openspec/
│   ├── specs/              # Main specs (human-managed)
│   │   ├── factory/
│   │   ├── knowledge-retrieval/
│   │   └── ...
│   ├── changes/            # Active changes (OpenSpec workflow)
│   ├── schemas/
│   │   └── software-factory/
│   └── config.yaml
├── holdout-scenarios/      # Archived validation scenarios
├── cmd/                    # Implementation output
├── internal/               # Implementation output
├── tests/acceptance/       # Visible acceptance tests
├── docs/                   # Documentation
└── .claude/                # Agent orchestration config
```
