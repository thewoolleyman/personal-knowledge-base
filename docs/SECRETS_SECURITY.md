# Secrets Security for AI Agent Workflows

How to prevent secrets from leaking into state files committed by AI agent
tooling (Claude Code hooks, beads, claude-flow, or similar).

## The problem

AI coding agents need access to real secrets (API keys, OAuth tokens) to do
their work. The agent reads env vars, calls APIs, runs commands. But the same
tooling that makes agents useful also **logs what they do**:

- **Hook logs** capture full tool input/output (including bash stdout/stderr)
- **Context bundles** record commands the agent ran
- **Issue trackers** (beads, Linear, Jira) store descriptions agents write
- **Memory databases** persist patterns and learnings across sessions
- **Metrics/state files** track operational data

If the agent runs `printenv` or `echo $SECRET`, the secret value appears in
the tool output, which gets logged. If the agent writes a secret into a bead
description or memory store, it persists.

**You cannot prevent the agent from reading secrets** -- it needs them. The
defense must be at the **output boundary**: redact before logging, scan before
committing, catch in CI.

## Architecture: defense in depth

```
Agent reads secret from env
        |
        v
  [Layer 1] Hook-level redaction
  sed pipeline strips known secret patterns from
  tool_response before writing to disk
        |
        v
  [Layer 2] Gitignore isolation
  Hook logs, context bundles, memory DBs, daemon state
  are all gitignored -- never enter the index
        |
        v
  [Layer 3] Pre-commit scanning
  gitleaks scans staged content for secret patterns
  before every commit -- catches anything in tracked files
        |
        v
  [Layer 4] CI scanning
  GitHub Actions runs gitleaks on push/PR -- catches
  secrets that bypass local hooks (direct push, other machines)
```

No single layer is sufficient. Together they catch secrets at every exit point.

## What this repo does

### Layer 1: Hook-level redaction

**Files**: `.claude/hooks/log-hook-event.sh`, `.claude/hooks/build-context-bundle.sh`

Both scripts run a sed pipeline before writing to disk:

```bash
INPUT="$(printf '%s' "$INPUT" | sed \
  -e 's/sk-ant-[a-zA-Z0-9_-]\{10,\}/[REDACTED]/g' \
  -e 's/sk-[a-zA-Z0-9]\{20,\}/[REDACTED]/g' \
  -e 's/AKIA[0-9A-Z]\{16\}/[REDACTED]/g' \
  -e 's/ghp_[a-zA-Z0-9]\{36\}/[REDACTED]/g' \
  -e 's/gho_[a-zA-Z0-9]\{36\}/[REDACTED]/g' \
  -e 's/-----BEGIN [A-Z ]*PRIVATE KEY-----/[REDACTED]/g' \
  -e 's/Bearer [a-zA-Z0-9._-]\{20,\}/Bearer [REDACTED]/g' \
  -e 's/ya29\.[a-zA-Z0-9._-]\{1,\}/[REDACTED]/g' \
  -e 's/SECRET[=:][^ "\\]*/SECRET=[REDACTED]/g' \
  -e 's/TOKEN[=:][^ "\\]*/TOKEN=[REDACTED]/g' \
  -e 's/KEY[=:][^ "\\]*/KEY=[REDACTED]/g' \
  -e 's/PASSWORD[=:][^ "\\]*/PASSWORD=[REDACTED]/g' \
  -e 's/CREDENTIAL[=:][^ "\\]*/CREDENTIAL=[REDACTED]/g' \
)"
```

Design constraints:
- **POSIX BRE only** -- macOS BSD sed has no `\|` alternation or `I` flag.
  Use separate `-e` for each pattern.
- **Single invocation** -- one `sed` call with 13 `-e` flags, one process spawn.
- **Best-effort** -- this is a defense layer, not the only one. Novel patterns
  will slip through; that's what layers 2-4 catch.
- **Env var keyword matching** -- `SECRET=`, `TOKEN=`, `KEY=`, `PASSWORD=`,
  `CREDENTIAL=` catch `printenv`/`env` output where lines are `VAR_NAME=value`.

### Layer 2: Gitignore isolation

**Files**: `.gitignore`, `.claude-flow/.gitignore`, `.claude-flow/learning/.gitignore`, `.beads/.gitignore`

State files fall into two categories:

| Category | Examples | Git status |
|----------|----------|------------|
| **Config/metadata** (safe to share) | `.beads/config.yaml`, `.claude-flow/config.yaml`, `.claude/settings.json` | Tracked |
| **Runtime/learning** (machine-local) | Hook logs, context bundles, memory DBs, daemon PIDs | Gitignored |

Key gitignore entries:

```gitignore
# Root .gitignore
.env
token.json
*.db
*.db-shm
*.db-wal
.claude-flow/learning/hook_logs/
.claude-flow/learning/context_bundles/

# .claude-flow/.gitignore
daemon-state.json
daemon.pid
data/
logs/
sessions/
neural/
*.log

# .claude-flow/learning/.gitignore (defense-in-depth)
*
!.gitignore

# .beads/.gitignore
*.db*
daemon.lock
daemon.log
daemon.pid
sync-state.json
```

The `.claude-flow/learning/.gitignore` with `*` is defense-in-depth -- even if
someone removes the root gitignore entries, learning data stays untracked.

### Layer 3: Pre-commit scanning

**Files**: `.git/hooks/pre-commit`, `.gitleaks.toml`

The pre-commit hook runs `gitleaks protect --staged` before every commit:

```bash
# Gitleaks secret scanning (runs before beads flush)
if command -v gitleaks >/dev/null 2>&1; then
    REPO_ROOT="$(git rev-parse --show-toplevel)"
    CONFIG="${REPO_ROOT}/.gitleaks.toml"
    if [ -f "$CONFIG" ]; then
        gitleaks protect --staged --no-banner -c "$CONFIG"
    else
        gitleaks protect --staged --no-banner
    fi
    # Non-zero exit = secrets found, block the commit
fi
```

If gitleaks is not installed, it warns and continues (doesn't block
collaborators who haven't installed it -- CI is the backstop).

The `.gitleaks.toml` config defines 14 rules covering:
- Anthropic, OpenAI, AWS, GCP, GitHub, Slack API keys/tokens
- Private keys (PEM), Bearer tokens, JWTs
- Google OAuth tokens and client secrets
- Generic API keys and passwords

It also allowlists:
- Gitignored paths (already excluded from commits)
- `.env.example` and test files (contain placeholder values)

**Install**: `brew install gitleaks`

### Layer 4: CI scanning

**File**: `.github/workflows/secrets-scan.yml`

```yaml
name: Secret Scanning
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
jobs:
  gitleaks:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0    # full history for thorough scanning
      - uses: gitleaks/gitleaks-action@v2
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

The action automatically uses `.gitleaks.toml` from the repo root. `fetch-depth: 0`
ensures it scans full commit history, not just the latest commit.

## Tracked state files: complete inventory

These are the files committed to git by beads and claude-flow tooling:

### Beads (.beads/)

| File | Content | Secret risk |
|------|---------|-------------|
| `issues.jsonl` | Issue titles, descriptions, close reasons | Medium -- agent may write secrets into descriptions |
| `interactions.jsonl` | Interaction history | Medium -- same risk |
| `config.yaml` | Issue prefix, daemon settings | None |
| `metadata.json` | DB pointer | None |

### Claude Flow (.claude-flow/)

| File | Content | Secret risk |
|------|---------|-------------|
| `config.yaml` | Topology, memory backend config | None |
| `pair-config.json` | TDD mode, coverage settings | None |
| `metrics/learning.json` | Pattern counts, routing accuracy | None |
| `metrics/swarm-activity.json` | Agent counts, swarm status | None |
| `metrics/v3-progress.json` | Implementation progress | None |
| `security/audit-status.json` | CVE counts, scan status | None |
| `CAPABILITIES.md` | Generated reference doc | None |

### Claude Code (.claude/)

| File | Content | Secret risk |
|------|---------|-------------|
| `settings.json` | Hook config, model preferences | Low -- no credentials |
| `agents/**/*.md` | Agent definitions | None |
| `commands/**/*.md` | Command docs | None |
| `hooks/*.sh` | Hook scripts (code, not data) | None |

### Other

| File | Content | Secret risk |
|------|---------|-------------|
| `.mcp.json` | MCP server config with env vars | Low -- currently only non-secret vars |
| `.gitleaks.toml` | Secret scanning rules | None |

## Reproducing in other repos

### Minimum viable setup (any repo with agent tooling)

1. **Install gitleaks**: `brew install gitleaks`

2. **Add `.gitleaks.toml`** to repo root. Start with the gitleaks
   [default config](https://github.com/gitleaks/gitleaks/blob/master/config/gitleaks.toml)
   and add allowlists for your agent state paths:
   ```toml
   [allowlist]
     paths = [
       '''\.claude-flow/learning/''',
       '''\.swarm/''',
       # add your agent tooling's gitignored paths
     ]
   ```

3. **Add pre-commit hook** -- either manually in `.git/hooks/pre-commit` or
   via [pre-commit framework](https://pre-commit.com/):
   ```yaml
   # .pre-commit-config.yaml
   repos:
     - repo: https://github.com/gitleaks/gitleaks
       rev: v8.21.2
       hooks:
         - id: gitleaks
   ```

4. **Add CI workflow** -- `.github/workflows/secrets-scan.yml` using
   `gitleaks/gitleaks-action@v2`.

5. **Gitignore agent state** -- ensure all runtime/learning data is excluded:
   ```gitignore
   # Agent state that should never be committed
   *.db
   *.db-wal
   *.db-shm
   .claude-flow/learning/
   .swarm/state.json
   ```

### Adding hook-level redaction (Claude Code specific)

If your agent tooling uses Claude Code hooks that log tool output, add the sed
redaction pipeline to any script that writes `tool_response` data to disk.

The pipeline from this repo (13 patterns, single sed invocation) can be copied
directly. Adapt the patterns for your secret types.

Key file to modify: whatever script handles your `PostToolUse` hook event.

### Beads-specific considerations

Beads tracks `issues.jsonl` in git by design (for cross-machine sync). This
means bead descriptions are committed. Mitigations:

- The pre-commit gitleaks scan catches secrets in `issues.jsonl`
- Agent instructions (CLAUDE.md or similar) should discourage writing secrets
  into bead descriptions, but this is not enforceable
- The gitleaks CI scan is the ultimate backstop

### For other agent state managers

The same principles apply to any tool that persists agent state to git:

1. **Identify what's tracked vs gitignored** -- `git ls-files` your state dirs
2. **Add redaction at the logging boundary** -- before data hits disk
3. **Add scanning at the commit boundary** -- pre-commit hook
4. **Add scanning in CI** -- catches everything else
5. **Gitignore machine-local state** -- PIDs, daemon state, memory DBs

## Make targets

```bash
make scan-secrets    # Run gitleaks detect on the full repo
make verify-hooks    # Verify hook logging, bundles, and recall work
```

## Limitations

- **Redaction is pattern-based** -- novel secret formats will pass through.
  The gitleaks rules and sed patterns need periodic updates.
- **Agent behavior is not controllable** -- the agent may store secrets in
  bead descriptions, memory, or other tracked files. Scanning catches this
  after the fact, not before.
- **Pre-commit hooks can be skipped** -- `git commit --no-verify` bypasses
  them. CI is the backstop.
- **Local disk exposure** -- even with gitignore, secrets in hook logs exist
  on the developer's machine. Disk encryption and access controls are the
  mitigation there, not this tooling.
