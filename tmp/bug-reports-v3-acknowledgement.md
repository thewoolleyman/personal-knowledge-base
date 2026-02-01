# Bug Report v3 -- Acknowledgement

**Fixed**: 2026-01-31
**Base commit**: `5b66c11` (HEAD of `main`)
**Report**: `tmp/bug-reports-v3.md`

---

## Disposition of All 6 Bugs

| Bug | Severity | Status | Notes |
|-----|----------|--------|-------|
| BUG-001 | Medium | **Fixed** | Added injectable `makeSignalCh` with cleanup; `defer stopSignals()` unregisters handler |
| BUG-002 | Medium | **Fixed** | Test injects test-owned channel via `makeSignalCh` override; no more `syscall.Kill` to process |
| BUG-003 | Low | **Fixed** | `agent` field in Task JSONL now escaped via `jq -Rs '.'` |
| BUG-004 | Low | **Fixed** | Removed `continueOnError: true` from all 18 hook handlers in `settings.json` |
| BUG-005 | Low | **Fixed** | Error message changed from `"write token file"` to `"save token file"` |
| BUG-006 | Low | **Fixed** | Updated 3 stale locations in docs that claimed hook-bridge.sh was unmodified |

## Rejected / No-Change Decisions

None. All 6 bugs were fixed as recommended.

## Verification

```
go vet ./...          # clean
go test -race ./...   # all pass, no races
```

| Package | Coverage |
|---------|----------|
| `internal/config` | 100.0% |
| `internal/search` | 100.0% |
| `internal/server` | 93.8% |
| `internal/tui` | 82.9% |
| `internal/connectors/gdrive` | 64.3% |
| `cmd/pkb` | 59.7% |

Note: `cmd/pkb` coverage decreased from 63.2% to 59.7% because the new `makeSignalCh` default function body (which calls `signal.Notify`) is production-only code -- tests correctly inject a mock channel instead of calling the real implementation.

## Files Modified

### Go Source
- `cmd/pkb/main.go` (BUG-001, BUG-002)
- `cmd/pkb/main_test.go` (BUG-002)
- `internal/connectors/gdrive/oauth.go` (BUG-005)

### Shell/Config
- `.claude/hooks/build-context-bundle.sh` (BUG-003)
- `.claude/settings.json` (BUG-004)

### Documentation
- `docs/claude-flow-context-and-memory.md` (BUG-006)
