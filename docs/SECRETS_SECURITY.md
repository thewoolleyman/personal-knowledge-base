# Secrets Security for AI Agent Workflows

How to prevent secrets from leaking into state files committed by AI agent
tooling (Claude Code hooks or similar).

## The problem

AI coding agents need access to real secrets (API keys, OAuth tokens) to do
their work. The agent reads env vars, calls APIs, runs commands. But the same
tooling that makes agents useful also **logs what they do**:

- **Hook logs** capture full tool input/output (including bash stdout/stderr)
- **Context bundles** record commands the agent ran
- **Issue trackers** (Linear, Jira, etc.) store descriptions agents write
- **Memory databases** persist patterns and learnings across sessions
- **Metrics/state files** track operational data

If the agent runs `printenv` or `echo $SECRET`, the secret value appears in
the tool output, which gets logged. If the agent writes a secret into a
tracked file or memory store, it persists.

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

If using Claude Code hooks that log tool output, add a sed redaction pipeline
to strip known secret patterns before writing to disk. This is a best-effort
defense layer — novel patterns will slip through, which is what layers 2-4 catch.

### Layer 2: Gitignore isolation

**Files**: `.gitignore`

State files fall into two categories:

| Category | Examples | Git status |
|----------|----------|------------|
| **Config/metadata** (safe to share) | `.claude/settings.json`, `openspec/` | Tracked |
| **Runtime/learning** (machine-local) | Memory DBs, daemon PIDs, tokens | Gitignored |

Key gitignore entries:

```gitignore
# Root .gitignore
.env
token.json
*.db
*.db-shm
*.db-wal
```

### Layer 3: Pre-commit scanning

**Files**: `.git/hooks/pre-commit`, `.gitleaks.toml`, `mise.toml`

Gitleaks is managed via [mise](https://mise.jdx.dev/) so the version is pinned
in `mise.toml` and auto-installed -- no manual `brew install` needed.

```toml
# mise.toml
[tools]
gitleaks = "8.21"
```

The pre-commit hook resolves gitleaks via mise first, falls back to PATH.
It runs `gitleaks protect --staged` on all staged content.
If gitleaks detects secrets, the commit is blocked.

To install or reinstall the hook:

```bash
make setup-hooks
```

This writes `.git/hooks/pre-commit` with gitleaks scanning. The hook
source of truth is the `setup-hooks` Makefile target -- `.git/hooks/` is not
tracked by git.

The `.gitleaks.toml` config defines 14 rules covering:
- Anthropic, OpenAI, AWS, GCP, GitHub, Slack API keys/tokens
- Private keys (PEM), Bearer tokens, JWTs
- Google OAuth tokens and client secrets
- Generic API keys and passwords

It also allowlists:
- Gitignored paths (already excluded from commits)
- `.env.example` and test files (contain placeholder values)

### Layer 4: CI scanning

**File**: `.github/workflows/ci-cd.yml` (the `secrets-scan` job)

The secrets scan runs as a Stage 1 job in the CI/CD pipeline, alongside lint.
Stage 2 tests depend on both lint and secrets-scan passing. This uses the same
mise-managed gitleaks -- same version, same config:

```yaml
# Inside ci-cd.yml, Stage 1 (alongside lint)
secrets-scan:
  name: Secrets Scan
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
      with:
        fetch-depth: 0
    - uses: jdx/mise-action@v2
    - run: mise x -- gitleaks detect --source . --no-banner -c .gitleaks.toml
```

`jdx/mise-action@v2` installs mise and runs `mise install` to get the pinned
gitleaks version from `mise.toml`. `fetch-depth: 0` ensures full commit history
is scanned. Test jobs will not run if secrets are detected.

## Tracked state files: complete inventory

These are the files committed to git by project tooling:

### Claude Code (.claude/)

| File | Content | Secret risk |
|------|---------|-------------|
| `settings.json` | Editor mode, plugin config | Low -- no credentials |
| `commands/**/*.md` | Command/skill definitions | None |
| `skills/**/*.md` | OpenSpec workflow skills | None |

### Other

| File | Content | Secret risk |
|------|---------|-------------|
| `.mcp.json` | MCP server config with env vars | Low -- currently only non-secret vars |
| `.gitleaks.toml` | Secret scanning rules | None |

## Reproducing in other repos

### Minimum viable setup (any repo with agent tooling)

1. **Pin gitleaks via mise** -- add `mise.toml` to repo root:
   ```toml
   [tools]
   gitleaks = "8.21"
   ```
   Then `mise install`. No brew, no manual downloads. CI uses `jdx/mise-action@v2`.

2. **Add `.gitleaks.toml`** to repo root. Start with the gitleaks
   [default config](https://github.com/gitleaks/gitleaks/blob/master/config/gitleaks.toml)
   and add allowlists for your agent state paths:
   ```toml
   [allowlist]
     paths = [
       # add your agent tooling's gitignored paths
     ]
   ```

3. **Add pre-commit hook** that resolves gitleaks via `mise x --` first, then
   falls back to PATH. See this repo's `.git/hooks/pre-commit` for the pattern.

4. **Add CI scanning** -- add a `secrets-scan` job to your CI/CD pipeline using
   `jdx/mise-action@v2` + `mise x -- gitleaks detect`. Gate downstream jobs on it.

5. **Gitignore agent state** -- ensure all runtime/learning data is excluded:
   ```gitignore
   # Agent state that should never be committed
   *.db
   *.db-wal
   *.db-shm
   ```

### Adding hook-level redaction (Claude Code specific)

If your agent tooling uses Claude Code hooks that log tool output, add the sed
redaction pipeline to any script that writes `tool_response` data to disk.

The pipeline from this repo (13 patterns, single sed invocation) can be copied
directly. Adapt the patterns for your secret types.

Key file to modify: whatever script handles your `PostToolUse` hook event.

### For other agent state managers

The same principles apply to any tool that persists agent state to git:

1. **Identify what's tracked vs gitignored** -- `git ls-files` your state dirs
2. **Add redaction at the logging boundary** -- before data hits disk
3. **Add scanning at the commit boundary** -- pre-commit hook
4. **Add scanning in CI** -- catches everything else
5. **Gitignore machine-local state** -- PIDs, daemon state, memory DBs

## Make targets

```bash
make setup-hooks          # Install pre-commit hook (gitleaks scanning)
make scan-secrets         # Run gitleaks detect on the full repo
make scan-secrets-staged  # Run gitleaks on staged files only
```

## Limitations

- **Redaction is pattern-based** -- novel secret formats will pass through.
  The gitleaks rules and sed patterns need periodic updates.
- **Agent behavior is not controllable** -- the agent may store secrets in
  tracked files. Scanning catches this
  after the fact, not before.
- **Pre-commit hooks can be skipped** -- `git commit --no-verify` bypasses
  them. CI is the backstop.
- **Local disk exposure** -- even with gitignore, secrets in hook logs exist
  on the developer's machine. Disk encryption and access controls are the
  mitigation there, not this tooling.
