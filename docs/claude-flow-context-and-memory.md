# Claude Flow Context Management & Memory Persistence

Research findings and local workaround plan for managing context, surviving
auto-compaction, and getting real memory persistence from claude-flow V3.

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

**Local fix applied**: Updated `hook-bridge.sh` stop-check to call the real
CLI command. Added `session-end` handler. Added `SessionEnd` hook to
`settings.json`.

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
| `session-end` (SessionEnd) | writes session summary | Yes -- fires on session exit (after our fix) |
| `persist-events.sh` (PostToolUse) | `.claude-flow/learning/events.jsonl` | Yes -- fires on Edit/Bash/Task (our workaround) |
| `import-events.sh` (SessionEnd) | `.swarm/memory.db` | Yes -- batch-imports events at session end (our workaround) |
| `recall-memory.sh` (UserPromptSubmit) | reads from memory DB | Yes -- searches memory on every user prompt (our workaround) |

## Local Workaround Plan

### Problem

The hooks fire automatically on every tool use but don't persist anything.
The working persistence paths (`memory store`, `intelligence trajectory-*`)
require explicit CLI calls. We need automatic persistence without:

- Replicating the 1,144-line `learning-service.mjs`
- Adding native dependencies (`better-sqlite3`)
- Breaking when upstream eventually fixes this

### Approach: JSONL Event Log + Batch Import + Automatic Recall

A zero-dependency append log captures events at near-zero cost (~1ms per hook
via bash+jq file append). At session end, a batch import writes accumulated
events to the working `memory store` SQLite path. On each new user prompt,
a recall hook searches the memory DB and injects relevant past context.

**New files:**

- `.claude/hooks/persist-events.sh` -- captures PostToolUse events to JSONL
- `.claude/hooks/import-events.sh` -- batch-imports JSONL to `memory store`
- `.claude/hooks/recall-memory.sh` -- searches memory on each user prompt,
  injects relevant results into Claude's context via stdout

**Modified files:**

- `.claude/settings.json` -- additional PostToolUse hooks for persist-events.sh,
  import step added to SessionEnd, recall-memory.sh added to UserPromptSubmit
  and PreToolUse (Task matcher)

**Data flow:**

```
WRITE PATH (during session):
  Edit/Bash/Task completes
    -> Claude Code fires PostToolUse
    -> hook-bridge.sh post-edit (existing, calls CLI stub -- does nothing)
    -> persist-events.sh (appends to events.jsonl)
       ... session continues ...

WRITE PATH (session end):
  Session ends
    -> import-events.sh reads events.jsonl
    -> Calls `memory store` for session summary (namespace: sessions)
    -> Calls `memory store` for per-file edit counts (namespace: edit-patterns)
    -> events.jsonl rotated to .jsonl.bak

READ PATH (next session or same session):
  User types a prompt
    -> recall-memory.sh reads prompt from stdin JSON
    -> Runs `memory search --query "<prompt>" --limit 5`
    -> Prints results to stdout
    -> Claude Code injects stdout into <system-reminder> tag
    -> Claude sees relevant past context before starting work

READ PATH (agent spawn):
  Task agent is about to be spawned (PreToolUse)
    -> recall-memory.sh searches memory for task-relevant context
    -> Results injected into Claude's context before agent starts
```

**Why this approach:**

- **Closed loop** -- data written at session end is automatically surfaced on
  relevant future prompts, without relying on the AI to remember to check
- Zero overhead during editing (file append vs npx cold-start)
- No native dependencies (pure bash + jq)
- Doesn't modify hook-bridge.sh (upstream file)
- Additive settings.json changes (new hook groups, not modifying existing)
- JSONL is human-readable and debuggable

### Forward-compatibility

When upstream catches up, this shim becomes redundant:

- CLI hooks write to `.swarm/memory.db` (different namespace than our shim)
- Two writes to different namespaces is harmless
- The shim can be cleanly removed

## Upstream Tracking: When to Remove the Workaround

Monitor these for resolution:

### PR #1059 -- Hook handlers persist data
**Status**: Open, no maintainer response as of 2026-01-31
**What it fixes**: Makes `hooksPostEdit`, `hooksPostTask`, `hooksPostCommand`
call `storeEntry()` instead of returning stubs.
**When merged**: Update CLI version, verify hooks write to `.swarm/memory.db`,
then remove `persist-events.sh` and the extra PostToolUse hooks from
settings.json.

### Issue #967 -- MCP/CLI backend unification
**Status**: Open, no maintainer response
**What it fixes**: Unifies MCP and CLI to use the same SQLite backend.
**When fixed**: Can switch from CLI-only memory operations to MCP tools.

### Issue #419 -- Self-improving workflow
**Status**: Open (feature request)
**What it would provide**: First-class auto-learning with `--train-neural`,
`--store-pattern` flags on hooks.
**When implemented**: Would make both our shim AND the CLAUDE.md prompt
engineering unnecessary.

### Init generator improvements
No issue filed yet. The `settings-generator.ts` and `helpers-generator.ts`
need to produce the learning pipeline for end users, not just for the upstream
repo's own use.

### Checklist for removing the workaround

1. [ ] PR #1059 merged and released
2. [ ] Verify `npx @claude-flow/cli@latest hooks post-edit --file X --success true` writes to `.swarm/memory.db`
3. [ ] Remove `.claude/hooks/persist-events.sh`
4. [ ] Remove `.claude/hooks/import-events.sh`
5. [ ] Remove `.claude/hooks/recall-memory.sh` (only if upstream adds automatic context injection)
6. [ ] Remove extra PostToolUse hook groups from `.claude/settings.json`
7. [ ] Remove import-events call from SessionEnd hooks in settings.json
8. [ ] Remove recall-memory.sh from UserPromptSubmit and PreToolUse hooks in settings.json
   (only if upstream provides equivalent context injection)
9. [ ] Verify `memory search` returns data written by hooks

**Note:** `recall-memory.sh` may be worth keeping even after upstream fixes
land, since it provides automatic context injection that the upstream hooks
don't offer. The upstream pipeline writes data to the DB but doesn't
automatically surface it on future prompts.

## Also Fixed Locally

### Stop hook (no-op -> real CLI call)

**hook-bridge.sh**: `stop-check` now calls
`npx @claude-flow/cli@latest hooks stop-check` which invokes
`generateStopCheck()` in the ReasoningBank.

### SessionEnd hook (missing -> added)

**settings.json**: Added `SessionEnd` hook event calling
`hook-bridge.sh session-end` which runs
`npx @claude-flow/cli@latest hooks session-end --generate-summary true --persist-state true --export-metrics true`.

**hook-bridge.sh**: Added `session-end` case.

## Instructions for agents to implement these changes in their own claude-flow repo

These steps reproduce the JSONL event-logging workaround in any project that
uses claude-flow V3 with Claude Code hooks. Prerequisites: `jq` installed,
`npx @claude-flow/cli@latest` available.

### 1. Create the event-capture hook

Create `.claude/hooks/persist-events.sh` (make it executable). This script:

- Reads Claude Code PostToolUse stdin JSON
- Extracts `tool_name` and relevant fields from `tool_input`
- Appends one JSONL line per event to `.claude-flow/learning/events.jsonl`
- Skips hook-infrastructure commands (anything starting with
  `npx @claude-flow`, `npx -y @claude-flow`, or `.claude/hooks/`)

Event format by tool type:

| Tool | JSONL fields |
|------|-------------|
| `Write`, `Edit`, `MultiEdit` | `{"ts":"...","type":"edit","file":"<file_path>"}` |
| `Task` | `{"ts":"...","type":"task","desc":"<description or prompt>","agent":"<subagent_type>"}` |
| `Bash` | `{"ts":"...","type":"command","cmd":"<command>"}` |

Key implementation details:

- Use `jq -r '.tool_name // empty'` to read stdin; exit early if empty
- For Task events, prefer `description` over `prompt`; truncate to 200 chars
- Use `jq -Rs '.'` to safely escape strings for JSON embedding
- Create the output directory with `mkdir -p` on every invocation

### 2. Create the session-end import hook

Create `.claude/hooks/import-events.sh` (make it executable). This script
runs at SessionEnd and:

1. Exits early if `events.jsonl` doesn't exist or is empty
2. Extracts unique edited files and counts edits per file
3. Extracts unique task descriptions
4. Counts command events
5. Stores a session summary via:
   ```
   npx @claude-flow/cli@latest memory store \
     --key "session-<timestamp>" \
     --value "<summary>" \
     --namespace sessions
   ```
6. Stores per-file edit counts via:
   ```
   npx @claude-flow/cli@latest memory store \
     --key "file-<md5-of-path>" \
     --value "<filepath> edited N times" \
     --namespace edit-patterns
   ```
7. Rotates `events.jsonl` to `events-<timestamp>.jsonl.bak`
8. Deletes backups beyond the 5 most recent

### 3. Create the memory-recall hook

Create `.claude/hooks/recall-memory.sh` (make it executable). This is the
**read side** — it closes the loop by searching memory on every user prompt
and injecting relevant past context. This script:

- Reads the user's prompt from stdin JSON (`.prompt` field)
- Skips prompts under 15 characters (not worth a search for "hi" or "yes")
- Runs `npx @claude-flow/cli@latest memory search --query "<prompt>" --limit 5`
- Prints matching results to stdout
- Claude Code injects hook stdout into `<system-reminder>` tags, so the
  results become part of Claude's context for that turn

Key implementation details:

- Check that `.swarm/memory.db` exists and is non-empty before calling npx
- Truncate the search query to 200 chars
- Count data rows in the CLI table output (rows starting with `|`, minus 1
  for the header row) — only output if there are actual data rows
- All errors are suppressed (`2>/dev/null`, `|| exit 0`) to avoid blocking
  the user's prompt

### 4. Add hooks to `.claude/settings.json`

Add three **new** PostToolUse hook groups (do not modify existing ones):

```json
{
  "matcher": "^(Write|Edit|MultiEdit)$",
  "hooks": [{
    "type": "command",
    "command": ".claude/hooks/persist-events.sh",
    "timeout": 2000,
    "continueOnError": true
  }]
},
{
  "matcher": "^Bash$",
  "hooks": [{
    "type": "command",
    "command": ".claude/hooks/persist-events.sh",
    "timeout": 2000,
    "continueOnError": true
  }]
},
{
  "matcher": "^Task$",
  "hooks": [{
    "type": "command",
    "command": ".claude/hooks/persist-events.sh",
    "timeout": 2000,
    "continueOnError": true
  }]
}
```

Add `recall-memory.sh` to `UserPromptSubmit` **before** the existing route
hook (so memory context is injected before routing):

```json
{
  "type": "command",
  "command": ".claude/hooks/recall-memory.sh",
  "timeout": 10000,
  "continueOnError": true
}
```

Optionally, add `recall-memory.sh` to `PreToolUse` with a `Task` matcher so
agents get relevant memory context before they start:

```json
{
  "matcher": "^Task$",
  "hooks": [{
    "type": "command",
    "command": ".claude/hooks/recall-memory.sh",
    "timeout": 10000,
    "continueOnError": true
  }]
}
```

Add `import-events.sh` to `SessionEnd` **before** any existing session-end
hooks (so events are imported before the session summary is generated):

```json
{
  "type": "command",
  "command": ".claude/hooks/import-events.sh",
  "timeout": 30000,
  "continueOnError": true
}
```

### 5. Update `.gitignore`

Add these patterns to prevent committing ephemeral event logs:

```
.claude-flow/learning/events.jsonl
.claude-flow/learning/*.jsonl.bak
```

### 6. Initialize the memory database

The memory DB may need initialization (or re-initialization if corrupt):

```bash
npx @claude-flow/cli@latest memory init --force --verbose
```

This creates the schema in `.swarm/memory.db`. Verify with:

```bash
npx @claude-flow/cli@latest memory store --key "test" --value "hello" --namespace test
npx @claude-flow/cli@latest memory search --query "hello" --namespace test
```

If search returns the entry, the DB is working.

### 7. Verify

Add a Makefile target (or run manually) that exercises the full pipeline:

```bash
# Clean slate
rm -f .claude-flow/learning/events.jsonl

# Simulate an Edit event
echo '{"tool_name":"Edit","tool_input":{"file_path":"/tmp/test.go"}}' \
  | .claude/hooks/persist-events.sh

# Simulate a Task event
echo '{"tool_name":"Task","tool_input":{"description":"Test","subagent_type":"coder"}}' \
  | .claude/hooks/persist-events.sh

# Simulate a Bash event
echo '{"tool_name":"Bash","tool_input":{"command":"go test ./..."}}' \
  | .claude/hooks/persist-events.sh

# Confirm 3 lines (hook commands should be skipped)
echo '{"tool_name":"Bash","tool_input":{"command":"npx @claude-flow/cli@latest memory search"}}' \
  | .claude/hooks/persist-events.sh
wc -l < .claude-flow/learning/events.jsonl  # expect: 3

# Run import
.claude/hooks/import-events.sh

# Confirm rotation
ls .claude-flow/learning/events-*.jsonl.bak  # expect: one backup file
test ! -f .claude-flow/learning/events.jsonl  # expect: original removed

# Test recall-memory.sh returns relevant results
echo '{"prompt":"How do hooks persist data to the memory database?"}' \
  | .claude/hooks/recall-memory.sh  # expect: "Relevant memory from past sessions"

# Test short prompts are skipped (no npx cost)
OUTPUT=$(echo '{"prompt":"hi"}' | .claude/hooks/recall-memory.sh)
test -z "$OUTPUT"  # expect: empty (skipped)
```

### Notes

- `persist-events.sh` must be fast (<2ms) since it fires on every tool use.
  File append with `printf >> file` is effectively free. Do not call `npx` or
  any network operation from this hook.
- `recall-memory.sh` calls `npx` (~200ms) on every user prompt. This is
  acceptable because it runs once per user message (not per tool use), and
  the latency is masked by the time the user spends reading the response.
  Short prompts (<15 chars) are skipped to avoid unnecessary searches.
- `import-events.sh` calls `npx` (with cold-start cost) but only runs once at
  session end, so this is acceptable.
- These hooks are additive -- they don't modify `hook-bridge.sh` or any
  upstream-generated files. They can be removed cleanly when upstream fixes
  land (see "Checklist for removing the workaround" above).
- The memory DB (`.swarm/memory.db`) must be initialized before the recall
  hook can return results. Run `memory init --force` if the DB is missing or
  corrupt (the file header says SQLite but tables are missing).

## References

- [claude-flow repo](https://github.com/ruvnet/claude-flow)
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
