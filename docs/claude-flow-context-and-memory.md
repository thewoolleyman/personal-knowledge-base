# Claude Flow Context Management & Memory Persistence

Research findings and upstream tracking for claude-flow V3's memory and
hook systems.

Date: 2026-01-31

## Background

Claude Code auto-compacts conversation context at 95% capacity. When this
happens, the conversation is summarized and older messages are replaced with
the summary. This is expected behavior, not a bug.

Claude-flow's architecture has a two-tier context model:

- **Tier 1: Conversation context (200K tokens)** -- ephemeral working memory.
  Lost or summarized on compaction.
- **Tier 2: Memory database (`.swarm/memory.db`)** -- persistent SQLite
  storage. Survives compaction, session restarts, everything.

The intended design is that important state lives in Tier 2. The hooks system
is supposed to automate writing to Tier 2. In practice, it doesn't.

## Findings

### 1. Hook handlers are stubs that don't persist data

**Issue**: [#1058](https://github.com/ruvnet/claude-flow/issues/1058) (open)
**PR**: [#1059](https://github.com/ruvnet/claude-flow/pulls/1059) (open, not merged)

The V3 CLI hook handlers (`hooksPostEdit`, `hooksPostTask`, `hooksPostCommand`)
in `hooks-tools.ts` return success responses but write nothing to the database.

```typescript
// hooksPostEdit -- returns this, stores nothing
return {
  recorded: true,  // <-- nothing is actually recorded
  filePath,
  success,
  learningUpdate: success ? 'pattern_reinforced' : 'pattern_adjusted',
};
```

`hooksPostTask` goes further, returning fabricated data:

```typescript
duration: Math.floor(Math.random() * 300) + 60,  // random number
patternsUpdated: success ? 2 : 1,                 // fake
```

The statusline calculates pattern count as `file_size / 2KB` -- also fake.

PR #1059 (by ahmedibrahim085) fixes this by making handlers call
`getRealStoreFunction()` which writes to SQLite with HNSW indexing. As of
2026-01-31, it has no maintainer response.

### 2. MCP and CLI use separate backends

**Issue**: [#967](https://github.com/ruvnet/claude-flow/issues/967) (open)

- CLI writes to `.swarm/memory.db` (SQLite)
- MCP tools read from `.claude-flow/memory/store.json` (JSON file)
- No synchronization between them

Workaround: Use CLI via Bash for all memory operations, not MCP tools.

### 3. Neural train persistence was fixed

**Issue**: [#961](https://github.com/ruvnet/claude-flow/issues/961) (closed)

Fixed in v3.0.0-alpha.123. `neural train` now persists patterns to
`.claude-flow/neural/patterns.json`. This only affects the explicit
`neural train` command, not the hook-based auto-learning pipeline.

### 4. Self-improving workflow is aspirational

**Issue**: [#419](https://github.com/ruvnet/claude-flow/issues/419) (open, 6 months)

Feature request from the maintainer describing the desired architecture.
Contains a full settings.json template referencing CLI flags that don't exist
(`--predict-performance`, `--train-neural`, `--store-pattern`). This is a
design document, not implemented code.

### 5. The upstream repo uses a different pipeline than what `init` generates

The upstream repo's own `.claude/settings.json` has ~40 hook commands across
8 event types. The `init` generator produces ~10 hooks across 6 event types.

Key files in the upstream repo that are NOT generated for end users:

- `learning-hooks.sh` -- bash wrapper for the learning service
- `learning-service.mjs` -- 1,144-line Node.js SQLite learning engine
- `learning-optimizer.sh` -- periodic pattern quality boosting
- `pattern-consolidator.sh` -- dedup, prune, promote patterns
- Plus ~20 other helper scripts

The upstream learning pipeline uses `better-sqlite3` (native SQLite binding),
manages short-term and long-term pattern tables with promotion/pruning
lifecycle, and implements an in-memory HNSW index. None of this is available
to end users through `init`.

### 6. Stop hook was a no-op stub

The generated Stop hook returned `{"ok":true}` without calling the CLI's
`generateStopCheck()` (which checks for unconsolidated patterns). The upstream
repo's own settings call `session-end` from Stop with full state persistence.

Additionally, no `SessionEnd` hook was generated. Claude Code has separate
`Stop` (fires every turn, can block) and `SessionEnd` (fires once on true
session exit, cannot block) events. The init generator conflated them.

### 7. The "AUTO-LEARNING PROTOCOL" in CLAUDE.md is prompt engineering

The CLAUDE.md section that says to run `memory store` commands after tasks is
instructions for Claude (the AI) to follow. It's not describing automated
infrastructure. Whether Claude actually follows these instructions depends on
whether it reads and prioritizes them, which becomes unreliable after
compaction.

## What actually works for persistence

| Path | Persists to | Automatic? |
|------|------------|------------|
| `memory store --key X --value Y` | `.swarm/memory.db` (SQLite) | No -- explicit CLI call |
| `hooks intelligence trajectory-*` | `.swarm/sona-patterns.json` | No -- explicit MCP/CLI call |
| `neural train` (alpha.123+) | `.claude-flow/neural/patterns.json` | No -- explicit CLI call |
| `hooks post-edit` / `hooks post-task` | nowhere | "Automatic" but stubs |
| `session-restore` (SessionStart) | reads from memory DB | Yes -- fires on new session |
| `session-end` (SessionEnd) | writes session summary | Yes -- fires on session exit |
| `log-hook-event.sh` (all events) | `.claude-flow/learning/hook_logs/` | Yes -- raw firehose, gitignored |
| `build-context-bundle.sh` (PostToolUse, UserPromptSubmit) | `.claude-flow/learning/context_bundles/` | Yes -- curated bundles, committed to git |
| `recall-memory.sh` (UserPromptSubmit, PreToolUse:Task) | reads from memory DB + greps context bundles | Yes -- dual-strategy recall on every prompt |

The last three rows are provided by
[claude-flow-memory-hooks](https://github.com/thewoolleyman/claude-flow-memory-hooks).

## Local workaround: claude-flow-memory-hooks

The hook architecture, implementation details, data flow, and setup
instructions are documented in the
[claude-flow-memory-hooks](https://github.com/thewoolleyman/claude-flow-memory-hooks)
repo.

To install the hooks into this project:

```bash
~/workspace/claude-flow-memory-hooks/scripts/install ~/workspace/personal-knowledge-base
```

To uninstall:

```bash
~/workspace/claude-flow-memory-hooks/scripts/uninstall ~/workspace/personal-knowledge-base
```

## Upstream Tracking: When to Remove the Workaround

Monitor these for resolution:

### PR #1059 -- Hook handlers persist data
**Status**: Open, no maintainer response as of 2026-01-31
**What it fixes**: Makes `hooksPostEdit`, `hooksPostTask`, `hooksPostCommand`
call `storeEntry()` instead of returning stubs.
**When merged**: Update CLI version, verify hooks write to `.swarm/memory.db`,
then uninstall the workaround.

### Issue #967 -- MCP/CLI backend unification
**Status**: Open, no maintainer response
**What it fixes**: Unifies MCP and CLI to use the same SQLite backend.
**When fixed**: Can switch from CLI-only memory operations to MCP tools.

### Issue #419 -- Self-improving workflow
**Status**: Open (feature request)
**What it would provide**: First-class auto-learning with `--train-neural`,
`--store-pattern` flags on hooks.
**When implemented**: Would make both the hooks workaround AND the CLAUDE.md
prompt engineering unnecessary.

### Checklist for removing the workaround

1. [ ] PR #1059 merged and released
2. [ ] Verify `npx @claude-flow/cli@latest hooks post-edit --file X --success true` writes to `.swarm/memory.db`
3. [ ] Run `~/workspace/claude-flow-memory-hooks/scripts/uninstall ~/workspace/personal-knowledge-base`
4. [ ] Verify `memory search` returns data written by hooks
5. [ ] Decide whether to keep `.claude-flow/learning/context_bundles/` in git
   (may still be useful as a git-native activity log even after upstream fixes)

**Note:** `recall-memory.sh` and `build-context-bundle.sh` may be worth keeping
even after upstream fixes land. The upstream pipeline writes data to the DB but
doesn't automatically surface it on future prompts. The context bundles provide
a git-native fallback that works on fresh clones without any DB setup.

## References

- [claude-flow repo](https://github.com/ruvnet/claude-flow)
- [claude-flow-memory-hooks](https://github.com/thewoolleyman/claude-flow-memory-hooks) (our workaround)
- [elite-context-engineering repo](https://github.com/ruvnet/elite-context-engineering) (inspired the two-tier hook architecture)
- [Issue #1058 -- Hook stubs](https://github.com/ruvnet/claude-flow/issues/1058)
- [PR #1059 -- Hook persistence fix](https://github.com/ruvnet/claude-flow/pulls/1059)
- [Issue #967 -- Backend split](https://github.com/ruvnet/claude-flow/issues/967)
- [Issue #961 -- Neural persistence](https://github.com/ruvnet/claude-flow/issues/961) (closed)
- [Issue #419 -- Self-improving workflow](https://github.com/ruvnet/claude-flow/issues/419)
- [PR #828 -- V2 pattern persistence fix](https://github.com/ruvnet/claude-flow/pull/828) (merged, V2 only)
- [Session Persistence Wiki](https://github.com/ruvnet/claude-flow/wiki/session-persistence)
- [Memory System Wiki](https://github.com/ruvnet/claude-flow/wiki/Memory-System)
- [Development Patterns Wiki](https://github.com/ruvnet/claude-flow/wiki/Development-Patterns)
- [Our previous upstream PR #1061](https://github.com/ruvnet/claude-flow/pull/1061)
