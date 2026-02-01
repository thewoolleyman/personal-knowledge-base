#!/usr/bin/env bash
# hook-bridge.sh - Bridges Claude Code hooks (stdin JSON) to claude-flow CLI
#
# Claude Code passes hook data as JSON via stdin, NOT as env vars.
# This script extracts the relevant fields and calls the CLI correctly.
#
# Usage: hook-bridge.sh <mode>

MODE="${1:-}"

# Read JSON from stdin (Claude Code always pipes hook data this way)
INPUT=$(cat 2>/dev/null) || INPUT='{}'
[ -z "$INPUT" ] && INPUT='{}'

# Safe JSON field extraction with fallback
jf() {
  echo "$INPUT" | jq -r "$1" 2>/dev/null | grep -v '^null$' || echo ""
}

case "$MODE" in
  # ── PreToolUse hooks ──────────────────────────────────────
  pre-edit)
    FILE=$(jf '.tool_input.file_path')
    [ -z "$FILE" ] && exit 0
    exec npx @claude-flow/cli@latest hooks pre-edit --file="$FILE"
    ;;

  pre-command)
    CMD=$(jf '.tool_input.command')
    [ -z "$CMD" ] && exit 0
    exec npx @claude-flow/cli@latest hooks pre-command --command="$CMD"
    ;;

  pre-task)
    DESC=$(jf '.tool_input.prompt // .tool_input.description')
    [ -z "$DESC" ] && exit 0
    exec npx @claude-flow/cli@latest hooks pre-task --description="$DESC"
    ;;

  # ── PostToolUse hooks ─────────────────────────────────────
  post-edit)
    FILE=$(jf '.tool_input.file_path')
    [ -z "$FILE" ] && exit 0
    exec npx @claude-flow/cli@latest hooks post-edit --file="$FILE" --success=true
    ;;

  post-command)
    CMD=$(jf '.tool_input.command')
    [ -z "$CMD" ] && exit 0
    exec npx @claude-flow/cli@latest hooks post-command --command="$CMD" --success=true
    ;;

  post-task)
    AGENT_ID=$(jf '.tool_input.description // .tool_input.prompt')
    [ -z "$AGENT_ID" ] && AGENT_ID="unknown"
    exec npx @claude-flow/cli@latest hooks post-task --task-id="$AGENT_ID" --success=true
    ;;

  # ── UserPromptSubmit ──────────────────────────────────────
  route)
    PROMPT=$(jf '.prompt')
    [ -z "$PROMPT" ] && exit 0
    exec npx @claude-flow/cli@latest hooks route --task="$PROMPT"
    ;;

  # ── SessionStart ──────────────────────────────────────────
  daemon-start)
    exec npx @claude-flow/cli@latest daemon start --quiet
    ;;

  session-restore)
    SID=$(jf '.session_id')
    if [ -n "$SID" ]; then
      exec npx @claude-flow/cli@latest hooks session-restore --session-id="$SID"
    else
      exec npx @claude-flow/cli@latest hooks session-restore --latest
    fi
    ;;

  # ── Stop ──────────────────────────────────────────────────
  stop-check)
    # Call the real CLI stop-check which checks for unconsolidated patterns
    # Exit code 2 = block Claude from stopping (patterns need consolidation)
    # Exit code 0 = ok to stop
    npx @claude-flow/cli@latest hooks stop-check 2>/dev/null
    RC=$?
    if [ "$RC" -eq 127 ]; then
        # CLI not found -- allow stop
        echo '{"ok":true}'
        exit 0
    fi
    exit "$RC"
    ;;

  # ── SessionEnd ──────────────────────────────────────────────
  session-end)
    npx @claude-flow/cli@latest hooks session-end \
      --generate-summary=true \
      --persist-state=true \
      --export-metrics=true 2>/dev/null || true
    ;;

  # ── Notification ──────────────────────────────────────────
  notify)
    MSG=$(jf '.message // .notification_message // .content')
    [ -z "$MSG" ] && exit 0
    exec npx @claude-flow/cli@latest memory store --namespace=notifications --key="notify" --value="$MSG"
    ;;

  *)
    echo "Unknown hook mode: $MODE" >&2
    exit 1
    ;;
esac
