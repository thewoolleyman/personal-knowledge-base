# Project Management Workflow

How this project tracks work, makes decisions, and coordinates development.

## The Tools

This project has two management systems installed. They serve different purposes
and do not conflict.

### Beads (Issue Tracking)

**What it is:** Git-native issue tracking stored in `.beads/`. Issues are JSONL
files synced via git, with a SQLite database for local queries.

**What it does:**
- Track work items: bugs, features, tasks, epics
- Manage dependencies between issues (`bd dep add`)
- Prioritize work (P0-P4)
- Surface ready work (`bd ready` = unblocked items)
- Sync state across sessions and machines via git

**When to use it:**
- Any work that spans sessions or needs to survive context loss
- Multi-step efforts with dependencies
- Anything a human collaborator should be able to see and pick up
- Epics that decompose into sub-tasks

**Commands:** `bd create`, `bd list`, `bd show`, `bd update`, `bd close`,
`bd dep add`, `bd ready`, `bd sync`, `bd stats`

### Claude Flow (Agent Coordination + Memory)

**What it is:** An npm-based CLI (`@claude-flow/cli`) that coordinates AI agent
swarms, stores cross-session memory, and provides hooks for learning.

**What it actually does in this project:**
- **Memory:** Stores and searches patterns across sessions
  (`npx @claude-flow/cli@latest memory store/search`)
- **Hooks:** Pre-task and post-task lifecycle events for learning
- **Swarm init:** Configures agent topology when spawning multiple agents

**What it does NOT do in this project:**
- It does not replace beads for issue tracking
- It does not run DDD analysis on our Go code (the DDD agent is a spec, not a scanner)
- It does not enforce ADRs automatically (the ADR agent is a template, not a linter)

## DDD Domains: What They Are and Whether We Need Them

### What the Claude Flow DDD agent actually is

The file `.claude/agents/v3/ddd-domain-expert.md` is an **agent specification**
originally written for the Claude Flow V3 project (a TypeScript swarm
coordination framework). It defines bounded contexts like Swarm, Agent, Task,
Memory, Neural, Security, MCP, and CLI -- these are Claude Flow's own internal
domains, not ours.

### Do we need DDD for PKB?

**Not yet.** Our current architecture is straightforward:

```
cmd/pkb/          CLI entry point (Cobra)
internal/
  connectors/     Interface + implementations (gdrive, gmail, ...)
  search/         Fan-out search across connectors
  config/         Environment-based configuration
  server/         HTTP API
  tui/            Bubble Tea interactive UI
  auth/           Authentication helpers
```

This is a clean package-per-concern layout. DDD adds value when you have complex
business logic with multiple interacting models, competing invariants, and team
boundaries. A search aggregator with pluggable connectors doesn't have that
complexity.

**When to reconsider:** If we add features like saved searches, user
preferences, sharing/collaboration, or content indexing, the domain model gets
complex enough to benefit from explicit bounded contexts and aggregates.

### If we ever adopt DDD

We could spawn the `ddd-domain-expert` agent to help identify bounded contexts
for PKB specifically. But we'd need to write our own domain model, not use the
Claude Flow V3 one. The bounded contexts would be something like:

- **Search** (Core) -- query construction, fan-out, result merging
- **Connectors** (Supporting) -- service-specific adapters
- **Configuration** (Generic) -- credentials, preferences

This is a future consideration, not current work.

## ADRs: What They Are and How to Use Them

### What ADRs are

Architecture Decision Records are short documents that capture significant
technical decisions: what was decided, why, and what the tradeoffs are. The
format is simple (MADR 3.0):

```markdown
# ADR-NNN: Title

## Status
Proposed | Accepted | Deprecated | Superseded

## Context
Why is this decision needed?

## Decision
What did we decide?

## Consequences
What becomes easier or harder?
```

### What the Claude Flow ADR agent actually is

The file `.claude/agents/v3/adr-architect.md` defines a template and lists 10
ADRs (ADR-001 through ADR-010). These are about **Claude Flow V3's own
architecture** (deep integration, modular DDD, security-first design, etc.).
They are not decisions about PKB.

### Should we write ADRs for PKB?

**Yes, selectively.** ADRs are useful for decisions that:
- Are hard to reverse
- Affect the whole codebase
- Were non-obvious (future-you needs to know *why*)
- Had meaningful alternatives that were rejected

Examples of PKB decisions worth documenting:
- "Use Go instead of TypeScript" (already made, could retroactively document)
- "Connector interface design" (why this interface shape)
- "Acceptance tests build a real binary" (why not just unit tests)
- "Use beads instead of GitHub Issues" (why git-native tracking)

### How to write an ADR

1. Create `docs/adr/NNN-short-title.md` using the MADR template above
2. Number sequentially (001, 002, ...)
3. Keep it short -- a good ADR is half a page, not five pages
4. Store in claude-flow memory for cross-session recall:
   ```bash
   npx @claude-flow/cli@latest memory store \
     --namespace decisions \
     --key "adr-001-go-language" \
     --value "Chose Go for CLI performance, single binary, strong stdlib"
   ```

We do NOT need the full ADR agent infrastructure (enforcement, compliance
scanning, ReasoningBank integration). Just write markdown files.

## The Canonical Workflow

### Starting a session

```bash
bd ready                    # What's available to work on?
bd show <id>                # Read the details
bd update <id> --status=in_progress

# Check if anyone already solved a similar problem
npx @claude-flow/cli@latest memory search --query "<bead keywords>"
```

### Planning new work

1. **Create an epic bead** for the overall effort
2. **Create task beads** for each sub-piece
3. **Add dependencies** so work is sequenced correctly
4. **Write an ADR** if the work involves a significant architectural choice

```bash
bd create --title="Epic: Add Slack connector" --type=feature --priority=2
bd create --title="Define Slack OAuth flow" --type=task --priority=2
bd create --title="Implement Slack search" --type=task --priority=2
bd dep add <search-id> <oauth-id>   # search depends on oauth
```

If there's a non-obvious decision (e.g., "use Socket Mode vs. Web API"):
write `docs/adr/NNN-slack-api-mode.md`.

### Doing the work

Follow TDD (see CLAUDE.md). Write failing test, make it pass, refactor.

### Completing work

Before closing an epic, follow the Epic Completion Checklist below.

```bash
bd close <id>               # Mark bead done

# Store what you learned
npx @claude-flow/cli@latest memory store \
  --namespace patterns \
  --key "pattern-<short-name>" \
  --value "<what worked, key files, gotchas>"

bd sync                     # Push beads state to git
git add <files> && git commit -m "..." && git push
```

## Epic Completion Checklist

Before closing an epic with `bd close <id>`, verify:

### For ALL Epics:
- [ ] All child beads are closed
- [ ] No blocking dependencies remain
- [ ] CI is green on main
- [ ] Code is merged and deployed
- [ ] Documentation updated

### For User-Facing Features (MANDATORY):
- [ ] **Acceptance tests exist** (see Testing Pyramid in CLAUDE.md)
- [ ] Tests build real binary as subprocess
- [ ] Tests verify from user's perspective (black box)
- [ ] Tests mirror README examples
- [ ] Run `make test-accept` - all pass
- [ ] Manual verification: follow README steps yourself

### For Internal Refactors:
- [ ] Unit tests updated
- [ ] Integration tests if needed
- [ ] No user-facing behavior changed
- [ ] Performance benchmarks if applicable

### Red Flags That Require Acceptance Tests:
- Added new CLI command or flag
- Added new HTTP endpoint
- Changed CLI output format
- Modified error messages users see
- Updated README with new examples
- Fixed a bug a user reported

**If you can't write acceptance tests yet:**
1. File a blocker bead explaining why
2. Update epic description with blocker details
3. DO NOT close epic until blocker resolved

### Session close checklist

```
[ ] git status                    (check what changed)
[ ] git add <files>               (stage changes)
[ ] bd sync                       (commit beads state)
[ ] git commit                    (commit code)
[ ] bd sync                       (commit any new beads changes)
[ ] git push                      (push to remote)
[ ] Verify CI green               (gh run list --limit=1)
[ ] Store patterns in memory      (anything learned this session)
```

## What Goes Where

| Thing to track | Tool | Why |
|---|---|---|
| Bug, feature, task, epic | **Beads** (`bd create`) | Persists in git, survives context loss, has dependencies |
| Quick in-session subtask | **TodoWrite** | Ephemeral, just for current session execution |
| Architectural decision | **ADR** (`docs/adr/NNN-*.md`) | Permanent record of why, not just what |
| Pattern / lesson learned | **Claude Flow memory** | Cross-session recall, searchable |
| Agent coordination | **Claude Flow swarm** | Only when spawning multiple agents |

## What We Do NOT Use

These Claude Flow features exist but are not part of our workflow:

- **DDD domain modeling** -- Our codebase is too simple to need it
- **ADR compliance scanning** -- We write ADRs manually; no automated enforcement
- **Neural pattern training** -- The hooks infrastructure exists but adds no value yet
- **Background workers** (ultralearn, consolidate, etc.) -- Overhead without payoff at current scale
- **Hive-mind consensus** -- We don't have multi-agent voting scenarios
- **Performance benchmarking** -- Relevant to Claude Flow itself, not to PKB

If any of these become useful as the project grows, we can adopt them
incrementally. The infrastructure is already configured.

## Summary

**Beads** is our issue tracker. It tracks what needs to be done, what's blocked,
and what's ready.

**ADRs** document why we made important decisions. They live in `docs/adr/` as
plain markdown.

**Claude Flow memory** stores patterns and lessons across sessions. It
complements beads by remembering how things were done, not just what was done.

**DDD, swarm coordination, neural learning, and other Claude Flow features** are
available but not currently needed. We'll adopt them if and when the project's
complexity warrants it.
