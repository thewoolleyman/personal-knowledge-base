## 1. Prerequisites

- [x] 1.1 Sync Kilroy fork against upstream (add upstream remote if missing, fetch, merge, push)
- [x] 1.2 Sync CXDB against latest master (`git pull` in `../cxdb`)
- [x] 1.3 Verify Kilroy builds and runs (`cd ../kilroy && go build ./cmd/kilroy`)

## 2. Pipeline Configuration

- [x] 2.1 Create `factory/pipeline-config.yaml` with PKB pipeline graph (start → implement → vet → lint → build → test → test-accept → review_final → exit, plus postmortem and human_gate error handling)
- [x] 2.2 Define tool gates mapping to Makefile targets: verify_vet (`make vet`), verify_lint (`make lint`), verify_build (`make build`), verify_test (`make test`), verify_test_accept (`make test-accept`)
- [x] 2.3 Define model stylesheet (claude-sonnet-4-6 for all nodes, 65536 max tokens)
- [x] 2.4 Set graph_id, graph_goal, topology (no-fanout), default_max_retry (3), retry_target, fallback_retry_target

## 3. Execution Configuration

- [x] 3.1 Create `factory/run.yaml` with repo path, CXDB connection (binary_addr, http_base_url), LLM provider (anthropic/api), git config (commit_per_node, run_branch_prefix)
- [x] 3.2 Add setup command for sparse-checkout to hide `holdout-scenarios/` from factory agents

## 4. Agent Prompts

- [x] 4.1 Create `factory/prompts/implement.md` — core implementation prompt covering PKB's Go architecture, spec reading instructions, repair awareness (postmortem check), deliverables, acceptance checks, status contract
- [x] 4.2 Create `factory/prompts/review_final.md` — semantic review prompt with numbered acceptance criteria (TDD compliance, testing pyramid, factory constraints, build/test passing)
- [x] 4.3 Create `factory/prompts/postmortem.md` — failure analysis prompt (summary, evidence, root cause, repair guidance, verification steps)
- [x] 4.4 Create `factory/prompts/human_gate.md` — human decision prompt directing to status and postmortem, offering retry or abort

## 5. Documentation

- [x] 5.1 Create `AGENTS.md` at repo root — agent dispatch guide covering tech stack (Go, Cobra, bubbletea, web UI), dependency locations (Kilroy, CXDB), spec structure (openspec/specs/), internal packages, path conventions, Make targets
- [x] 5.2 Create `docs/software-factory.md` — PKB factory guide covering architecture overview, prerequisites (Kilroy, CXDB setup), pipeline generation, running the factory, checking status, resuming, landing changes, relationship to CI

## 6. Validation

- [x] 6.1 Run `kilroy validate` against the pipeline config to verify it parses correctly
- [x] 6.2 Generate `pipeline.dot` and verify the graph structure
- [x] 6.3 Run each tool gate command manually to confirm they pass on the current codebase
- [x] 6.4 Verify sparse-checkout setup command works (holdout-scenarios hidden, specs visible)
