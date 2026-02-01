#!/usr/bin/env bash
# import-events.sh — SessionEnd hook that summarizes events.jsonl into
# claude-flow memory, then rotates the log.
set -euo pipefail

EVENTS_DIR=".claude-flow/learning"
EVENTS_FILE="${EVENTS_DIR}/events.jsonl"
MAX_BACKUPS=5

# Exit early if no events recorded
[ ! -f "$EVENTS_FILE" ] && exit 0
[ ! -s "$EVENTS_FILE" ] && exit 0

TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
TS_SLUG="$(date -u +%Y%m%dT%H%M%SZ)"

# ── Gather edit stats ──────────────────────────────────────────────
EDIT_LINES="$(jq -r 'select(.type=="edit") | .file' "$EVENTS_FILE")"
if [ -n "$EDIT_LINES" ]; then
  UNIQUE_FILES="$(printf '%s\n' "$EDIT_LINES" | sort -u)"
  FILE_COUNT="$(printf '%s\n' "$UNIQUE_FILES" | wc -l | tr -d ' ')"
  FILE_LIST="$(printf '%s\n' "$UNIQUE_FILES" | head -20 | paste -sd, -)"
else
  FILE_COUNT=0
  FILE_LIST=""
fi

# ── Gather task stats ─────────────────────────────────────────────
TASK_LINES="$(jq -r 'select(.type=="task") | .desc' "$EVENTS_FILE")"
if [ -n "$TASK_LINES" ]; then
  UNIQUE_TASKS="$(printf '%s\n' "$TASK_LINES" | sort -u)"
  TASK_LIST="$(printf '%s\n' "$UNIQUE_TASKS" | head -10 | paste -sd'; ' -)"
else
  TASK_LIST=""
fi

# ── Gather command stats ──────────────────────────────────────────
CMD_COUNT="$(jq -r 'select(.type=="command")' "$EVENTS_FILE" | wc -l | tr -d ' ')"

# ── Build summary string ─────────────────────────────────────────
SUMMARY="Edited ${FILE_COUNT} files: ${FILE_LIST}. Tasks: ${TASK_LIST:-none}. Commands: ${CMD_COUNT}."

# ── Store session summary ─────────────────────────────────────────
npx @claude-flow/cli@latest memory store \
  --key "session-${TS_SLUG}" \
  --value "$SUMMARY" \
  --namespace sessions 2>/dev/null || true

# ── Store per-file edit counts ────────────────────────────────────
if [ -n "$EDIT_LINES" ]; then
  printf '%s\n' "$EDIT_LINES" | sort | uniq -c | sort -rn | head -20 | while read -r COUNT FILEPATH; do
    # Use a short hash of filepath as key to avoid special chars
    FHASH="$(printf '%s' "$FILEPATH" | md5 2>/dev/null || printf '%s' "$FILEPATH" | md5sum 2>/dev/null | cut -d' ' -f1)"
    npx @claude-flow/cli@latest memory store \
      --key "file-${FHASH}" \
      --value "${FILEPATH} edited ${COUNT} times" \
      --namespace edit-patterns 2>/dev/null || true
  done
fi

# ── Rotate events log ────────────────────────────────────────────
mv "$EVENTS_FILE" "${EVENTS_DIR}/events-${TS_SLUG}.jsonl.bak"

# Keep only the most recent backups
# shellcheck disable=SC2012
ls -1t "${EVENTS_DIR}"/events-*.jsonl.bak 2>/dev/null | tail -n +$((MAX_BACKUPS + 1)) | while read -r OLD; do
  rm -f "$OLD"
done

exit 0
