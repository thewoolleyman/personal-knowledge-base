#!/usr/bin/env bash
# recall-memory.sh — UserPromptSubmit hook that searches for context
# relevant to the user's current prompt and outputs it so Claude sees it.
#
# Two search strategies:
#   1. Semantic search via claude-flow memory DB (if available)
#   2. Keyword grep of context bundles (git-native fallback)
#
# Claude Code injects hook stdout into <system-reminder> tags.
set -euo pipefail

INPUT="$(cat 2>/dev/null)" || INPUT='{}'
[ -z "$INPUT" ] && exit 0

# Extract the user's prompt
PROMPT="$(printf '%s' "$INPUT" | jq -r '.prompt // empty')"
[ -z "$PROMPT" ] && exit 0

# Skip very short prompts (greetings, single words)
[ "${#PROMPT}" -lt 15 ] && exit 0

QUERY="$(printf '%s' "$PROMPT" | head -c 200)"
FOUND=""

# ── Strategy 1: Semantic search via memory DB ──────────────────────
DB=".claude/memory.db"
if [ -f "$DB" ] && [ -s "$DB" ]; then
  RESULTS="$(npx @claude-flow/cli@latest memory search --query "$QUERY" --limit 5 2>/dev/null)" || RESULTS=""
  if printf '%s' "$RESULTS" | grep -q '|.*|.*|.*|'; then
    ROW_COUNT="$(printf '%s' "$RESULTS" | grep -c '^|' || true)"
    DATA_ROWS=$(( ROW_COUNT - 1 ))
    if [ "$DATA_ROWS" -gt 0 ]; then
      echo ""
      echo "Relevant memory (${DATA_ROWS} entries):"
      echo "$RESULTS"
      echo ""
      FOUND="yes"
    fi
  fi
fi

# ── Strategy 2: Grep context bundles (git-native fallback) ────────
BUNDLE_DIR=".claude-flow/learning/context_bundles"
if [ -d "$BUNDLE_DIR" ]; then
  # Extract 2-3 keywords from the prompt (longest words, likely most specific)
  KEYWORDS="$(printf '%s' "$QUERY" | tr -cs '[:alnum:]' '\n' | awk 'length >= 4' | sort -u | head -5)"
  if [ -n "$KEYWORDS" ]; then
    # Build a grep pattern matching any keyword
    PATTERN="$(printf '%s' "$KEYWORDS" | paste -sd'|' -)"
    MATCHES="$(grep -rhi "$PATTERN" "$BUNDLE_DIR"/*.jsonl 2>/dev/null | sort -u | tail -10)" || MATCHES=""
    if [ -n "$MATCHES" ]; then
      echo ""
      echo "Context from past sessions:"
      echo "$MATCHES"
      echo ""
      FOUND="yes"
    fi
  fi
fi

exit 0
