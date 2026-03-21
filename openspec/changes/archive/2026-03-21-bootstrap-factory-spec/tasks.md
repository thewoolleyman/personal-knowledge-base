## 1. Customize the software-factory schema

- [x] 1.1 Update `openspec/schemas/software-factory/schema.yaml` to add `holdout-scenarios` artifact (requires: specs, not required by tasks)
- [x] 1.2 Update the `specs` artifact instruction to reference intent.md/contracts.md/constraints.md convention instead of single spec.md
- [x] 1.3 Update the `specs` artifact template to reflect the factory taxonomy files
- [x] 1.4 Set `software-factory` as the default schema in `openspec/config.yaml`
- [x] 1.5 Validate the custom schema: `npx openspec schema validate software-factory`

## 2. Create factory capability specs

- [x] 2.1 Create `openspec/specs/factory/intent.md` — purpose of the factory, Software Factory + OpenSpec integration philosophy
- [x] 2.2 Create `openspec/specs/factory/contracts.md` — schema artifact flow, spec file conventions, capability list
- [x] 2.3 Create `openspec/specs/factory/constraints.md` — TDD rules, CI rules, testing pyramid, secrets handling (migrated from CLAUDE.md)

## 3. Scaffold product capability directories

- [x] 3.1 Create `openspec/specs/knowledge-retrieval/` with empty intent.md
- [x] 3.2 Create `openspec/specs/knowledge-ingestion/` with empty intent.md
- [x] 3.3 Create `openspec/specs/protocol-interfaces/` with empty intent.md
- [x] 3.4 Create `openspec/specs/connectors/` with empty intent.md
- [x] 3.5 Create `openspec/specs/authentication/` with empty intent.md
- [x] 3.6 Create `openspec/specs/infrastructure/` with empty intent.md

## 4. Create holdout-scenarios directory

- [x] 4.1 Create `holdout-scenarios/` at repo root with a `.gitkeep`
- [x] 4.2 Add `holdout-scenarios/` to `.gitignore` exclusion if needed (ensure it's tracked)

## 5. Slim down CLAUDE.md

- [x] 5.1 Remove rules from CLAUDE.md that are now captured in `factory/constraints.md`
- [x] 5.2 Add reference in CLAUDE.md pointing agents to `openspec/specs/factory/` for full rules
- [x] 5.3 Keep only directives in CLAUDE.md that must be there for agent bootstrapping (file organization, important instruction reminders, beads viewer reference)

## 6. Verify and commit

- [x] 6.1 Run `npx openspec status --change bootstrap-factory-spec` to confirm all artifacts complete
- [x] 6.2 Commit all changes
- [x] 6.3 Push and verify CI
