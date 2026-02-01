# Bug Report v5 -- Acknowledgement

**Fixed**: 2026-01-31
**Base commit**: `5921a5d` (HEAD of `main`)
**Report**: `tmp/bug-reports-v5.md`

---

## Disposition of All 9 Bugs

| Bug | Severity | Status | Notes |
|-----|----------|--------|-------|
| BUG-001 | Medium | **Fixed** | README updated with `serve` and `interactive` docs; package table updated; acceptance tests added for both commands |
| BUG-002 | Medium | **Fixed** | CLAUDE.md directory list replaced with actual Go project structure (`cmd/pkb`, `internal/`, `tests/acceptance/`, `docs/`) |
| BUG-003 | Low | **Fixed** | Hardcoded `/Users/cwoolley/` and personal email replaced with `<your-username>` and `<your-email>` placeholders |
| BUG-004 | Low | **Fixed** | All `--flag value` pairs in hook-bridge.sh and recall-memory.sh changed to `--flag=value` syntax |
| BUG-005 | Low | **Fixed** | log-hook-event.sh now uses `jq -cn --arg/--argjson` for safe JSON construction |
| BUG-006 | Low | **Fixed** | Combined with BUG-005; log writes wrapped in `flock` for atomic concurrent access |
| BUG-007 | Low | **Fixed** | Removed redundant `-r` flag from grep in recall-memory.sh |
| BUG-008 | Low | **Fixed** | build-context-bundle.sh emits warning on unknown arguments instead of silently discarding |
| BUG-009 | Low | **Fixed** | Row-counting in recall-memory.sh now excludes separator rows via `grep -vc '^|[-+ ]*|$'` |

## Rejected / No-Change Decisions

None. All 9 bugs were fixed as recommended.

## Verification

```
go vet ./...              # clean
go test -race -cover ./.. # all pass, no races
```

| Package | Coverage |
|---------|----------|
| `internal/config` | 100.0% |
| `internal/search` | 100.0% |
| `internal/server` | 93.8% |
| `internal/tui` | 82.9% |
| `internal/connectors/gdrive` | 64.3% |
| `cmd/pkb` | 59.7% |

No coverage change from v4 -- v5 bugs were docs and shell scripts (not Go source).

## Files Modified

### Documentation
- `README.md` (BUG-001, BUG-003)
- `CLAUDE.md` (BUG-002)

### Shell Hooks
- `.claude/hooks/hook-bridge.sh` (BUG-004)
- `.claude/hooks/log-hook-event.sh` (BUG-005, BUG-006)
- `.claude/hooks/recall-memory.sh` (BUG-004, BUG-007, BUG-009)
- `.claude/hooks/build-context-bundle.sh` (BUG-008)

### Tests
- `tests/acceptance/cli_test.go` (BUG-001 -- new acceptance tests for serve and interactive)
