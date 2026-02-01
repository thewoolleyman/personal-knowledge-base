# Beads Workflow Context

> **Context Recovery**: Run `bd prime` after compaction, clear, or new session
> Hooks auto-call this in Claude Code when .beads/ detected

# 🚨 SESSION CLOSE PROTOCOL 🚨

**CRITICAL**: Before saying "done" or "complete", you MUST run this checklist:

```
[ ] 1. git status              (check what changed)
[ ] 2. git add <files>         (stage code changes)
[ ] 3. bd sync                 (commit beads changes)
[ ] 4. git commit -m "..."     (commit code)
[ ] 5. bd sync                 (commit any new beads changes)
[ ] 6. git push                (push to remote)
[ ] 7. Store patterns          (memory store for anything learned)
```

**NEVER skip this.** Work is not done until pushed and patterns are stored.

## Memory Protocol (MANDATORY for bead work)

**Before starting ANY bead:**
```bash
bd show <id>                                    # understand the issue
npx @claude-flow/cli@latest memory search \
  --query "<bead title and keywords>"           # find relevant patterns
# Apply any relevant patterns found to your approach
```

**After completing ANY bead:**
```bash
npx @claude-flow/cli@latest memory store \
  --namespace patterns \
  --key "pattern-<short-name>" \
  --value "<what worked, key decisions, file locations>"
```

**After completing ALL beads in a session:**
```bash
npx @claude-flow/cli@latest hooks post-task \
  --task-id "<session-summary>" \
  --success true --store-results true
```

Stored patterns persist across sessions and help future work go faster.
Search memory FIRST — don't reinvent approaches that already worked.

## Core Rules
- Track strategic work in beads (multi-session, dependencies, discovered work)
- Use `bd create` for issues, TodoWrite for simple single-session execution
- When in doubt, prefer bd—persistence you don't need beats lost context
- Git workflow: hooks auto-sync, run `bd sync` at session end
- Session management: check `bd ready` for available work

## Essential Commands

### Finding Work
- `bd ready` - Show issues ready to work (no blockers)
- `bd list --status=open` - All open issues
- `bd list --status=in_progress` - Your active work
- `bd show <id>` - Detailed issue view with dependencies

### Creating & Updating
- `bd create --title="..." --type=task|bug|feature --priority=2` - New issue
  - Priority: 0-4 or P0-P4 (0=critical, 2=medium, 4=backlog). NOT "high"/"medium"/"low"
- `bd update <id> --status=in_progress` - Claim work
- `bd update <id> --assignee=username` - Assign to someone
- `bd update <id> --title/--description/--notes/--design` - Update fields inline
- `bd close <id>` - Mark complete
- `bd close <id1> <id2> ...` - Close multiple issues at once (more efficient)
- `bd close <id> --reason="explanation"` - Close with reason
- **Tip**: When creating multiple issues/tasks/epics, use parallel subagents for efficiency
- **WARNING**: Do NOT use `bd edit` - it opens $EDITOR (vim/nano) which blocks agents

### Dependencies & Blocking
- `bd dep add <issue> <depends-on>` - Add dependency (issue depends on depends-on)
- `bd blocked` - Show all blocked issues
- `bd show <id>` - See what's blocking/blocked by this issue

### Sync & Collaboration
- `bd sync` - Sync with git remote (run at session end)
- `bd sync --status` - Check sync status without syncing

### Project Health
- `bd stats` - Project statistics (open/closed/blocked counts)
- `bd doctor` - Check for issues (sync problems, missing hooks)

## Common Workflows

**Starting work:**
```bash
bd ready           # Find available work
bd show <id>       # Review issue details
bd update <id> --status=in_progress  # Claim it
npx @claude-flow/cli@latest memory search --query "<bead title>"  # Check for patterns
```

**Completing work:**
```bash
bd close <id1> <id2> ...    # Close all completed issues at once
npx @claude-flow/cli@latest memory store --namespace patterns --key "pattern-<name>" --value "<learnings>"
bd sync                     # Push to remote
```

**Creating dependent work:**
```bash
# Run bd create commands in parallel (use subagents for many items)
bd create --title="Implement feature X" --type=feature
bd create --title="Write tests for X" --type=task
bd dep add beads-yyy beads-xxx  # Tests depend on Feature (Feature blocks tests)
```
