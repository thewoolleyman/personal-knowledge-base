#!/usr/bin/env bash
# log-hook-event.sh — Logs the full hook JSON payload to a per-session,
# per-hook-event JSONL file. This is the raw observability layer.
#
# Output: .claude-flow/learning/hook_logs/{session_id}/{hook_event_name}.jsonl
#
# Each line is: {"ts":"...","payload":{...full stdin JSON...}}
set -euo pipefail

INPUT="$(cat 2>/dev/null)" || INPUT='{}'
[ -z "$INPUT" ] && exit 0

SESSION_ID="$(printf '%s' "$INPUT" | jq -r '.session_id // "unknown"')"
HOOK_NAME="$(printf '%s' "$INPUT" | jq -r '.hook_event_name // "Unknown"')"

LOG_DIR=".claude-flow/learning/hook_logs/${SESSION_ID}"
mkdir -p "$LOG_DIR"

TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Append full payload as a single JSONL line with portable locking
LOCKFILE="${LOG_DIR}/${HOOK_NAME}.jsonl.lock"
LOGFILE="${LOG_DIR}/${HOOK_NAME}.jsonl"
if command -v flock >/dev/null 2>&1; then
  (
    flock 200
    jq -cn --arg ts "$TS" --argjson payload "$INPUT" \
      '{"ts": $ts, "payload": $payload}' >> "$LOGFILE" 2>/dev/null || true
  ) 200>"$LOCKFILE"
else
  # macOS: use mkdir as atomic lock (no flock available)
  while ! mkdir "$LOCKFILE.d" 2>/dev/null; do sleep 0.01; done
  jq -cn --arg ts "$TS" --argjson payload "$INPUT" \
    '{"ts": $ts, "payload": $payload}' >> "$LOGFILE" 2>/dev/null || true
  rmdir "$LOCKFILE.d" 2>/dev/null || true
fi

exit 0
