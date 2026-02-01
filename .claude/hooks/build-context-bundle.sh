#!/usr/bin/env bash
# build-context-bundle.sh — Extracts curated fields from hook events into
# a compact per-session context bundle. This is the signal layer that gets
# committed to git and used for recall.
#
# Output: .claude-flow/learning/context_bundles/{DAY_HOUR}_{session_id}.jsonl
#
# Called from PostToolUse (file ops, commands, tasks) and UserPromptSubmit.
# Accepts --type flag: "tool" (default) or "prompt".
set -euo pipefail

TYPE="tool"
while [ $# -gt 0 ]; do
  case "$1" in
    --type) TYPE="$2"; shift 2 ;;
    *) shift ;;
  esac
done

INPUT="$(cat 2>/dev/null)" || INPUT='{}'
[ -z "$INPUT" ] && exit 0

SESSION_ID="$(printf '%s' "$INPUT" | jq -r '.session_id // "unknown"')"
DAY_HOUR="$(date -u +%a_%H | tr '[:lower:]' '[:upper:]')"

BUNDLE_DIR=".claude-flow/learning/context_bundles"
mkdir -p "$BUNDLE_DIR"
BUNDLE_FILE="${BUNDLE_DIR}/${DAY_HOUR}_${SESSION_ID}.jsonl"

PROJECT_DIR="$(pwd)"

# Convert absolute path to relative
to_relative() {
  local ABS="$1"
  case "$ABS" in
    "${PROJECT_DIR}/"*) printf '%s' "${ABS#${PROJECT_DIR}/}" ;;
    *) printf '%s' "$ABS" ;;
  esac
}

if [ "$TYPE" = "prompt" ]; then
  PROMPT="$(printf '%s' "$INPUT" | jq -r '.prompt // empty')"
  [ -z "$PROMPT" ] && exit 0
  # Truncate to 500 chars, escape for JSON
  PROMPT_SHORT="$(printf '%s' "$PROMPT" | head -c 500 | jq -Rs '.')"
  printf '{"op":"prompt","text":%s}\n' "$PROMPT_SHORT" >> "$BUNDLE_FILE"
  exit 0
fi

# Tool events — extract based on tool_name
TOOL_NAME="$(printf '%s' "$INPUT" | jq -r '.tool_name // empty')"
[ -z "$TOOL_NAME" ] && exit 0

case "$TOOL_NAME" in
  Read)
    FILE_PATH="$(printf '%s' "$INPUT" | jq -r '.tool_input.file_path // empty')"
    [ -z "$FILE_PATH" ] && exit 0
    REL="$(to_relative "$FILE_PATH")"
    REL_JSON="$(printf '%s' "$REL" | jq -Rs '.')"
    printf '{"op":"read","file":%s}\n' "$REL_JSON" >> "$BUNDLE_FILE"
    ;;
  Write)
    FILE_PATH="$(printf '%s' "$INPUT" | jq -r '.tool_input.file_path // empty')"
    [ -z "$FILE_PATH" ] && exit 0
    REL="$(to_relative "$FILE_PATH")"
    REL_JSON="$(printf '%s' "$REL" | jq -Rs '.')"
    printf '{"op":"write","file":%s}\n' "$REL_JSON" >> "$BUNDLE_FILE"
    ;;
  Edit|MultiEdit)
    FILE_PATH="$(printf '%s' "$INPUT" | jq -r '.tool_input.file_path // empty')"
    [ -z "$FILE_PATH" ] && exit 0
    REL="$(to_relative "$FILE_PATH")"
    REL_JSON="$(printf '%s' "$REL" | jq -Rs '.')"
    printf '{"op":"edit","file":%s}\n' "$REL_JSON" >> "$BUNDLE_FILE"
    ;;
  Task)
    DESC="$(printf '%s' "$INPUT" | jq -r '.tool_input.description // empty')"
    PROMPT="$(printf '%s' "$INPUT" | jq -r '.tool_input.prompt // empty')"
    AGENT="$(printf '%s' "$INPUT" | jq -r '.tool_input.subagent_type // empty')"
    LABEL="${DESC:-$PROMPT}"
    LABEL="$(printf '%s' "$LABEL" | head -c 200 | jq -Rs '.')"
    [ "$LABEL" = '""' ] && exit 0
    AGENT_JSON="$(printf '%s' "$AGENT" | jq -Rs '.')"
    printf '{"op":"task","desc":%s,"agent":%s}\n' "$LABEL" "$AGENT_JSON" >> "$BUNDLE_FILE"
    ;;
  Bash)
    CMD="$(printf '%s' "$INPUT" | jq -r '.tool_input.command // empty')"
    [ -z "$CMD" ] && exit 0
    # Skip hook-infrastructure commands
    case "$CMD" in
      npx\ @claude-flow*|npx\ -y\ @claude-flow*|.claude/hooks/*) exit 0 ;;
    esac
    CMD_SHORT="$(printf '%s' "$CMD" | head -c 200 | jq -Rs '.')"
    printf '{"op":"command","cmd":%s}\n' "$CMD_SHORT" >> "$BUNDLE_FILE"
    ;;
esac

exit 0
