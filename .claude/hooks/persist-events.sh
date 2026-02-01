#!/usr/bin/env bash
# persist-events.sh — Appends a JSONL line per PostToolUse event.
# Called by Claude Code hooks; reads hook JSON from stdin.
set -euo pipefail

EVENTS_DIR=".claude-flow/learning"
EVENTS_FILE="${EVENTS_DIR}/events.jsonl"

# Read stdin JSON (Claude Code passes tool invocation details)
INPUT="$(cat)"

# Bail out if stdin was empty
[ -z "$INPUT" ] && exit 0

TOOL_NAME="$(printf '%s' "$INPUT" | jq -r '.tool_name // empty')"
[ -z "$TOOL_NAME" ] && exit 0

TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

mkdir -p "$EVENTS_DIR"

case "$TOOL_NAME" in
  Write|Edit|MultiEdit)
    FILE_PATH="$(printf '%s' "$INPUT" | jq -r '.tool_input.file_path // empty')"
    [ -z "$FILE_PATH" ] && exit 0
    printf '{"ts":"%s","type":"edit","file":"%s"}\n' "$TS" "$FILE_PATH" >> "$EVENTS_FILE"
    ;;
  Task)
    DESC="$(printf '%s' "$INPUT" | jq -r '.tool_input.description // empty')"
    PROMPT="$(printf '%s' "$INPUT" | jq -r '.tool_input.prompt // empty')"
    AGENT="$(printf '%s' "$INPUT" | jq -r '.tool_input.subagent_type // empty')"
    LABEL="${DESC:-$PROMPT}"
    # Truncate long prompts to keep JSONL lines reasonable
    LABEL="$(printf '%s' "$LABEL" | head -c 200)"
    [ -z "$LABEL" ] && exit 0
    # Escape for JSON safety
    LABEL="$(printf '%s' "$LABEL" | jq -Rs '.')"
    printf '{"ts":"%s","type":"task","desc":%s,"agent":"%s"}\n' "$TS" "$LABEL" "$AGENT" >> "$EVENTS_FILE"
    ;;
  Bash)
    CMD="$(printf '%s' "$INPUT" | jq -r '.tool_input.command // empty')"
    [ -z "$CMD" ] && exit 0
    # Skip hook-infrastructure commands to avoid recursive noise
    case "$CMD" in
      npx\ @claude-flow*|npx\ -y\ @claude-flow*|.claude/hooks/*) exit 0 ;;
    esac
    # Truncate and escape
    CMD_SHORT="$(printf '%s' "$CMD" | head -c 200)"
    CMD_SHORT="$(printf '%s' "$CMD_SHORT" | jq -Rs '.')"
    printf '{"ts":"%s","type":"command","cmd":%s}\n' "$TS" "$CMD_SHORT" >> "$EVENTS_FILE"
    ;;
esac

exit 0
