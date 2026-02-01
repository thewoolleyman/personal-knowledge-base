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
| `log-hook-event.sh` (Pre/PostToolUse, all events) | `.claude-flow/learning/hook_logs/{session}/{event}.jsonl` | Yes -- raw firehose, gitignored (our workaround) |
| `build-context-bundle.sh` (PostToolUse, UserPromptSubmit) | `.claude-flow/learning/context_bundles/{DAY_HOUR}_{session}.jsonl` | Yes -- curated bundles, committed to git (our workaround) |
| `recall-memory.sh` (UserPromptSubmit, PreToolUse:Task) | reads from memory DB + greps context bundles | Yes -- dual-strategy recall on every prompt (our workaround) |

## Local Workaround Plan

### Problem

The hooks fire automatically on every tool use but don't persist anything.
The working persistence paths (`memory store`, `intelligence trajectory-*`)
require explicit CLI calls. We need automatic persistence without:

- Replicating the 1,144-line `learning-service.mjs`
- Adding native dependencies (`better-sqlite3`)
- Breaking when upstream eventually fixes this

### Approach: Two-Tier Logging + Git-Native Context Bundles + Automatic Recall

Inspired by the [elite-context-engineering](https://github.com/ruvnet/elite-context-engineering)
repo's Python hook scripts, this uses a two-tier architecture:

- **Tier 1 (raw firehose):** `log-hook-event.sh` logs every hook payload to
  per-session, per-event JSONL files. Full observability for debugging. Gitignored.
- **Tier 2 (curated signal):** `build-context-bundle.sh` extracts compact,
  relevant fields (file paths, commands, prompts) into context bundles that are
  committed to git and survive across clones.
- **Read side:** `recall-memory.sh` uses two search strategies (semantic SQLite
  search + keyword grep of context bundles) to inject relevant past context on
  every user prompt.

**Hook files (all in `.claude/hooks/`):**

| File | Purpose | Fires on |
|------|---------|----------|
| `log-hook-event.sh` | Tier 1: raw payload logging | All events (wildcard matcher) |
| `build-context-bundle.sh` | Tier 2: curated context bundles | PostToolUse (Read/Write/Edit/Bash/Task), UserPromptSubmit |
| `recall-memory.sh` | Read side: dual-strategy recall | UserPromptSubmit, PreToolUse (Task) |
| `hook-bridge.sh` | Upstream bridge (unchanged) | Various pre/post events |

**Data flow:**

```
TIER 1 — RAW LOGGING (every hook event):
  Any hook fires (Pre/PostToolUse, UserPromptSubmit, etc.)
    -> log-hook-event.sh appends full JSON payload
    -> .claude-flow/learning/hook_logs/{session_id}/{HookName}.jsonl
    -> Gitignored — debugging and observability only

TIER 2 — CURATED BUNDLES (during session):
  PostToolUse fires for Read/Write/Edit/Bash/Task
    -> build-context-bundle.sh extracts compact fields
    -> Converts absolute paths to project-relative paths
    -> Skips hook-infrastructure commands (npx @claude-flow, .claude/hooks/)
    -> Appends to .claude-flow/learning/context_bundles/{DAY_HOUR}_{session_id}.jsonl
    -> Committed to git — survives across clones

  UserPromptSubmit fires
    -> build-context-bundle.sh --type prompt
    -> Records truncated prompt text to the same bundle file

READ PATH (every user prompt):
  User types a prompt
    -> recall-memory.sh reads prompt from stdin JSON
    -> Strategy 1: Semantic search via `memory search` (if .swarm/memory.db exists)
    -> Strategy 2: Keyword grep of context bundles (git-native fallback)
    -> Prints results to stdout
    -> Claude Code injects stdout into <system-reminder> tag
    -> Claude sees relevant past context before starting work

READ PATH (agent spawn):
  Task agent is about to be spawned (PreToolUse)
    -> recall-memory.sh searches for task-relevant context
    -> Results injected into Claude's context before agent starts
```

**Why this approach:**

- **Git-native persistence** -- context bundles are committed to git, so context
  survives across clones and machines without depending on a SQLite DB
- **Closed loop** -- curated context is automatically surfaced on relevant
  future prompts via grep fallback, even without a memory DB
- **Two strategies** -- semantic search (if DB available) + keyword grep (always
  available) means recall works on fresh clones and corrupt DBs
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
then follow the removal checklist below.

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
3. [ ] Remove `.claude/hooks/log-hook-event.sh`
4. [ ] Remove `.claude/hooks/build-context-bundle.sh`
5. [ ] Remove `.claude/hooks/recall-memory.sh` (only if upstream adds automatic context injection)
6. [ ] Remove build-context-bundle.sh hook groups from PostToolUse and UserPromptSubmit in `.claude/settings.json`
7. [ ] Remove log-hook-event.sh wildcard hooks from all event types in settings.json
8. [ ] Remove recall-memory.sh from UserPromptSubmit and PreToolUse hooks in settings.json
   (only if upstream provides equivalent context injection)
9. [ ] Verify `memory search` returns data written by hooks
10. [ ] Decide whether to keep `.claude-flow/learning/context_bundles/` in git
    (may still be useful as a git-native activity log even after upstream fixes)

**Note:** `recall-memory.sh` and `build-context-bundle.sh` may be worth keeping
even after upstream fixes land. The upstream pipeline writes data to the DB but
doesn't automatically surface it on future prompts. The context bundles provide
a git-native fallback that works on fresh clones without any DB setup.

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

These steps reproduce the two-tier logging workaround in any project that
uses claude-flow V3 with Claude Code hooks. Prerequisites: `jq` installed,
`npx @claude-flow/cli@latest` available.

### 1. Create the raw event logger (Tier 1)

Create `.claude/hooks/log-hook-event.sh` (make it executable). This is the
raw observability layer that logs every hook payload for debugging:

- Reads the full hook stdin JSON
- Extracts `session_id` and `hook_event_name`
- Appends the timestamped payload to
  `.claude-flow/learning/hook_logs/{session_id}/{HookName}.jsonl`
- Creates the session directory with `mkdir -p`

Output format (one line per event):
```json
{"ts":"2026-01-31T22:00:00Z","payload":{...full hook JSON...}}
```

Key implementation details:

- Use `jq -r '.session_id // "unknown"'` and `.hook_event_name // "Unknown"`
- All errors exit silently (`set -euo pipefail`, `|| exit 0`)
- Must be fast (<2ms) since it fires on every hook event via wildcard matcher

### 2. Create the context bundle builder (Tier 2)

Create `.claude/hooks/build-context-bundle.sh` (make it executable). This is
the curated signal layer that extracts compact, meaningful fields:

- Accepts `--type tool` (default) or `--type prompt` argument
- For tool events: extracts operation type, file paths, commands, task descriptions
- For prompt events: records truncated user prompt text
- Converts absolute paths to project-relative paths
- Skips hook-infrastructure commands (`npx @claude-flow*`, `.claude/hooks/*`)
- Appends to `.claude-flow/learning/context_bundles/{DAY_HOUR}_{session_id}.jsonl`

Bundle entry formats by operation:

| Tool | Bundle entry |
|------|-------------|
| `Read` | `{"op":"read","file":"internal/server/server.go"}` |
| `Write` | `{"op":"write","file":"internal/server/server.go"}` |
| `Edit`, `MultiEdit` | `{"op":"edit","file":"internal/server/server.go"}` |
| `Task` | `{"op":"task","desc":"Research X","agent":"researcher"}` |
| `Bash` | `{"op":"command","cmd":"go test ./..."}` |
| User prompt | `{"op":"prompt","text":"How do I implement OAuth?"}` |

Key implementation details:

- Use `$(pwd)` to compute project root for relative path conversion
- For Task events, prefer `description` over `prompt`; truncate to 200 chars
- Use `jq -Rs '.'` to safely escape strings for JSON embedding
- The `DAY_HOUR` prefix (e.g., `FRI_18`) groups activity by time window
- Context bundles are committed to git (NOT gitignored)

### 3. Create the memory-recall hook (Read side)

Create `.claude/hooks/recall-memory.sh` (make it executable). This closes
the loop by searching past context on every user prompt and injecting it:

**Strategy 1: Semantic search via memory DB**
- Checks if `.swarm/memory.db` exists and is non-empty
- Runs `npx @claude-flow/cli@latest memory search --query "<prompt>" --limit 5`
- Counts data rows in CLI table output (rows starting with `|`, minus 1 for header)
- Outputs results if any data rows exist

**Strategy 2: Keyword grep of context bundles (git-native fallback)**
- Extracts keywords (4+ chars) from the prompt
- Builds an alternation pattern and greps all `.jsonl` files in `context_bundles/`
- Returns up to 10 unique matching lines

Both strategies run in sequence. Either, both, or neither may produce output.

Key implementation details:

- Reads `.prompt` from stdin JSON; skips prompts under 15 characters
- Truncates search query to 200 chars
- All errors suppressed (`2>/dev/null`, `|| MATCHES=""`) — never blocks the prompt
- Claude Code injects hook stdout into `<system-reminder>` tags

### 4. Add hooks to `.claude/settings.json`

**PreToolUse hooks:**

Add `log-hook-event.sh` as a wildcard matcher (fires on all tool uses):
```json
{
  "matcher": "*",
  "hooks": [{
    "type": "command",
    "command": ".claude/hooks/log-hook-event.sh",
    "timeout": 2000,
    "continueOnError": true
  }]
}
```

Add `recall-memory.sh` with a `Task` matcher (so agents get context):
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

**PostToolUse hooks:**

Add `build-context-bundle.sh` for file/command/task events:
```json
{
  "matcher": "^(Read|Write|Edit|MultiEdit|Bash|Task)$",
  "hooks": [{
    "type": "command",
    "command": ".claude/hooks/build-context-bundle.sh --type tool",
    "timeout": 2000,
    "continueOnError": true
  }]
}
```

Add `log-hook-event.sh` as a wildcard matcher:
```json
{
  "matcher": "*",
  "hooks": [{
    "type": "command",
    "command": ".claude/hooks/log-hook-event.sh",
    "timeout": 2000,
    "continueOnError": true
  }]
}
```

**UserPromptSubmit hooks:**

Add `recall-memory.sh` **before** the route hook:
```json
{
  "type": "command",
  "command": ".claude/hooks/recall-memory.sh",
  "timeout": 10000,
  "continueOnError": true
}
```

Add `build-context-bundle.sh` with `--type prompt`:
```json
{
  "type": "command",
  "command": ".claude/hooks/build-context-bundle.sh --type prompt",
  "timeout": 2000,
  "continueOnError": true
}
```

Add `log-hook-event.sh`:
```json
{
  "type": "command",
  "command": ".claude/hooks/log-hook-event.sh",
  "timeout": 2000,
  "continueOnError": true
}
```

**SessionStart, Stop, SessionEnd, Notification:**

Add `log-hook-event.sh` to each of these event types with the same
wildcard configuration shown above.

### 5. Update `.gitignore`

Add raw hook logs (large, per-session, debugging only):
```
.claude-flow/learning/hook_logs/
```

Do **NOT** gitignore `context_bundles/` — these are the curated, compact
bundles that should be committed to git for cross-clone persistence.

### 6. Optionally initialize the memory database

The SQLite memory DB enhances recall but is not required (the grep fallback
works without it):

```bash
npx @claude-flow/cli@latest memory init --force --verbose
```

This creates the schema in `.swarm/memory.db`. Verify with:

```bash
npx @claude-flow/cli@latest memory store --key "test" --value "hello" --namespace test
npx @claude-flow/cli@latest memory search --query "hello" --namespace test
```

If search returns the entry, the DB is working. If the DB is missing or
corrupt, the grep fallback in recall-memory.sh still provides context
from committed context bundles.

### 7. Verify

Run `make verify-hooks` to exercise the full pipeline, or run the checks
manually:

```bash
# Test raw event logging (Tier 1)
echo '{"session_id":"test","hook_event_name":"PostToolUse","tool_name":"Edit"}' \
  | .claude/hooks/log-hook-event.sh
cat .claude-flow/learning/hook_logs/test/PostToolUse.jsonl
# expect: {"ts":"...","payload":{...}}

# Test context bundle for Edit event (Tier 2)
echo '{"session_id":"test","tool_name":"Edit","tool_input":{"file_path":"'$(pwd)'/internal/server/server.go"}}' \
  | .claude/hooks/build-context-bundle.sh --type tool
# expect: {"op":"edit","file":"internal/server/server.go"} (relative path)

# Test context bundle for Bash command
echo '{"session_id":"test","tool_name":"Bash","tool_input":{"command":"go test ./..."}}' \
  | .claude/hooks/build-context-bundle.sh --type tool
# expect: {"op":"command","cmd":"go test ./..."}

# Test hook-infrastructure commands are skipped
echo '{"session_id":"test","tool_name":"Bash","tool_input":{"command":"npx @claude-flow/cli@latest memory search"}}' \
  | .claude/hooks/build-context-bundle.sh --type tool
# expect: no new line added (skipped)

# Test user prompt capture
echo '{"session_id":"test","prompt":"How do I implement OAuth?"}' \
  | .claude/hooks/build-context-bundle.sh --type prompt
# expect: {"op":"prompt","text":"How do I implement OAuth?"}

# Test recall with grep fallback
echo '{"prompt":"How do hooks persist data to the memory database?"}' \
  | .claude/hooks/recall-memory.sh
# expect: "Context from past sessions:" with matching bundle lines

# Test short prompts are skipped
OUTPUT=$(echo '{"prompt":"hi"}' | .claude/hooks/recall-memory.sh)
test -z "$OUTPUT"  # expect: empty (skipped)
```

### Notes

- `log-hook-event.sh` and `build-context-bundle.sh` must be fast (<2ms)
  since they fire on every tool use. File append with `printf >> file` is
  effectively free. Neither calls `npx` or any network operation.
- `recall-memory.sh` calls `npx` (~200ms) for Strategy 1, but only on
  user prompts (not per tool use). Short prompts (<15 chars) are skipped.
  Strategy 2 (grep) is nearly instant. Both are acceptable latency.
- Context bundles are committed to git. They're compact (~50-100 bytes per
  entry) and provide cross-clone persistence without any database setup.
- Raw hook logs are gitignored. They contain full payloads and grow quickly.
  Use them for debugging hook behavior, then delete when no longer needed.
- These hooks are additive — they don't modify `hook-bridge.sh` or any
  upstream-generated files. They can be removed cleanly when upstream fixes
  land (see "Checklist for removing the workaround" above).
- The memory DB (`.swarm/memory.db`) is optional. If missing or corrupt,
  recall-memory.sh falls back to grepping context bundles. Run
  `memory init --force` to (re)create the DB if you want semantic search.

## References

- [claude-flow repo](https://github.com/ruvnet/claude-flow)
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
