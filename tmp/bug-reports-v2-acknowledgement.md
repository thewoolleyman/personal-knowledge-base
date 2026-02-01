# Bug Report v2 -- Acknowledgement

**Fixed**: 2026-01-31
**Base commit**: `f2956b1` (HEAD of `main`)
**Report**: `tmp/bug-reports-v2.md`

---

## Disposition of All 18 Bugs

| Bug | Severity | Status | Notes |
|-----|----------|--------|-------|
| BUG-001 | High | **Fixed** | Reverted DB path to `.swarm/memory.db` in `recall-memory.sh:26` |
| BUG-002 | High | **Fixed** | Explicit `f.Close()` error check in `SaveToken`; 3 tests added |
| BUG-003 | Medium | **Fixed** | `m.err = nil` on successful search in TUI; test added |
| BUG-004 | Medium | **Fixed** | Stop-check preserves CLI exit code; only falls back on 127 (not found) |
| BUG-005 | Medium | **Fixed** | Recall hook checks both `.prompt` and `.tool_input.prompt` |
| BUG-006 | Medium | **Fixed** | `config.Load()` error checked and propagated in `buildSearchFn` |
| BUG-007 | Medium | **Fixed** | Graceful shutdown via SIGINT/SIGTERM; `http.ErrServerClosed` treated as clean exit; test added |
| BUG-008 | Medium | **Fixed** | `TestLoadToken_InvalidJSON` added covering decode error path |
| BUG-009 | Medium | **Improved** | Coverage up (config 100%, tui 82.9%, gdrive 64.3%, cmd/pkb 63.2%); remaining gaps are integration-test-only or `main()` |
| BUG-010 | Low | **Fixed** | Removed `exec` from session-end command so `\|\| true` is reachable |
| BUG-011 | Low | **Fixed** | Read/Write/Edit paths in `build-context-bundle.sh` now escape via `jq -Rs` |
| BUG-012 | Low | **Fixed** | `m.cancel = nil` in both success and error paths of `handleSearchResult` |
| BUG-013 | Low | **Fixed** | `Init()` returns `textinput.Blink` directly instead of closure wrapper |
| BUG-014 | Low | **Fixed** | `search.Engine.Search` returns `[]Result{}` instead of `nil` for zero connectors |
| BUG-015 | Low | **Fixed** | `noopSearch` stub replaces `nil` in `TestRun_ReturnsNilOnSuccess` and `TestSearchCommand_NoQuery` |
| BUG-016 | Low | **Fixed** | `TestLoad_TokenPathDefault` and `TestLoad_TokenPathEnvOverride` added |
| BUG-017 | Low | **Fixed** | `verify-hooks` counts lines across all matching bundle files to handle hour boundaries |
| BUG-018 | Low | **No change** | `config.Load()` returning `(*Config, error)` with nil error is acceptable Go forward-compatibility idiom. Callers now check the error (BUG-006), so adding validation later requires no signature change. |

## Rejected / No-Change Decisions

- **BUG-018**: No code change. The `error` return is a forward-compatibility idiom. The report itself said "keeping the error return for forward compatibility is acceptable if BUG-006 is also fixed" -- BUG-006 is fixed.
- **BUG-009**: Coverage mandate is 100% on *new code*. All new code written in this fix round is fully covered. Remaining gaps (`NewAPIClient` 0%, `SearchFiles` 0%) require real Google credentials (integration tests, not unit tests). `main()` at 0% is expected per project conventions.

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
| `cmd/pkb` | 63.2% |

## Files Modified

### Shell/Hooks
- `.claude/hooks/recall-memory.sh` (BUG-001, BUG-005)
- `.claude/hooks/hook-bridge.sh` (BUG-004, BUG-010)
- `.claude/hooks/build-context-bundle.sh` (BUG-011)

### Go Source
- `internal/connectors/gdrive/oauth.go` (BUG-002)
- `internal/tui/tui.go` (BUG-003, BUG-012, BUG-013)
- `cmd/pkb/main.go` (BUG-006, BUG-007)
- `internal/search/search.go` (BUG-014)

### Tests
- `internal/connectors/gdrive/oauth_test.go` (BUG-002, BUG-008)
- `internal/tui/tui_test.go` (BUG-003, BUG-012, BUG-013)
- `cmd/pkb/main_test.go` (BUG-006, BUG-007, BUG-015)
- `internal/config/config_test.go` (BUG-016)
- `internal/search/search_test.go` (BUG-014)

### Build
- `Makefile` (BUG-017)
