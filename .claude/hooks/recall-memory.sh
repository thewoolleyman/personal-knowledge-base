#!/usr/bin/env bash
# recall-memory.sh — UserPromptSubmit hook that searches memory for context
# relevant to the user's current prompt and outputs it so Claude sees it.
#
# Claude Code injects hook stdout into <system-reminder> tags, which means
# anything this script prints becomes part of Claude's context for the
# current turn.
set -euo pipefail

# Read stdin JSON from Claude Code
INPUT="$(cat 2>/dev/null)" || INPUT='{}'
[ -z "$INPUT" ] && exit 0

# Extract the user's prompt
PROMPT="$(printf '%s' "$INPUT" | jq -r '.prompt // empty')"
[ -z "$PROMPT" ] && exit 0

# Skip very short prompts (greetings, single words) — not worth a search
[ "${#PROMPT}" -lt 15 ] && exit 0

# Truncate prompt to first 200 chars for the search query
QUERY="$(printf '%s' "$PROMPT" | head -c 200)"

DB=".swarm/memory.db"
# Skip if memory DB doesn't exist or is empty
[ ! -f "$DB" ] && exit 0
[ ! -s "$DB" ] && exit 0

# Search across all namespaces for relevant entries.
# Use --limit 5 to keep context injection small.
# Capture output; suppress errors (corrupt DB, missing tables, etc.)
RESULTS="$(npx @claude-flow/cli@latest memory search --query "$QUERY" --limit 5 2>/dev/null)" || exit 0

# Only output if there are actual results (not just header/footer lines).
# The CLI table format uses | for header and data rows, + for separators.
# Count rows starting with |, subtract 1 for the header row.
if printf '%s' "$RESULTS" | grep -q '|.*|.*|.*|'; then
  ROW_COUNT="$(printf '%s' "$RESULTS" | grep -c '^|' || true)"
  DATA_ROWS=$(( ROW_COUNT - 1 ))
  [ "$DATA_ROWS" -le 0 ] && exit 0

  echo ""
  echo "Relevant memory from past sessions (${DATA_ROWS} entries):"
  echo "$RESULTS"
  echo ""
fi

exit 0
