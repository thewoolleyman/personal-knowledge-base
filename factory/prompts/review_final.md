# PKB Final Review

You are the final semantic reviewer for the PKB implementation. All deterministic tool gates (vet, lint, build, test, test-accept) have already passed. Your job is to verify fidelity to the specification and adherence to factory constraints.

## Instructions

1. Read the specifications at `openspec/specs/` (intent, contracts, constraints for each capability)
2. Read the factory constraints at `openspec/specs/factory/constraints.md`
3. Examine the implementation code
4. Evaluate each acceptance criterion below
5. Write your review to `.ai/review_final.md`

## Acceptance Criteria

### TDD & Testing (AC-T1 through AC-T5)

- **AC-T1**: Every new function/method has a corresponding test
- **AC-T2**: Tests follow table-driven pattern where multiple cases exist
- **AC-T3**: Tests are in the same package (_test.go alongside source)
- **AC-T4**: Acceptance tests exist for all user-facing CLI commands and HTTP endpoints
- **AC-T5**: Acceptance tests build the real binary and execute as subprocess (never import internal packages)

### Code Quality (AC-Q1 through AC-Q5)

- **AC-Q1**: `main()` is a thin wrapper calling `run() error`
- **AC-Q2**: No `log.Fatal()` or `os.Exit()` outside of `main()`
- **AC-Q3**: All OS/system/network interaction is behind interfaces (mockable in tests)
- **AC-Q4**: Error handling uses returned errors, not panics
- **AC-Q5**: No hardcoded secrets, credentials, or absolute paths

### Specification Fidelity (AC-S1 through AC-S5)

- **AC-S1**: All requirements marked SHALL/MUST in specs are implemented
- **AC-S2**: API contracts (routes, request/response shapes) match spec definitions
- **AC-S3**: Constraint invariants are satisfied (SLOs, security requirements)
- **AC-S4**: No features added beyond what the specification requires
- **AC-S5**: CLI commands and flags match what the specification defines

### Build & CI (AC-B1 through AC-B3)

- **AC-B1**: `make vet` passes
- **AC-B2**: `make lint` passes
- **AC-B3**: `make test` and `make test-accept` both pass

## Output Format

Write your review to `.ai/review_final.md` in this format:

```markdown
# Final Review

## Summary
<1-2 sentence overall assessment>

## Acceptance Criteria Results

| ID | Description | Result | Notes |
|----|-------------|--------|-------|
| AC-T1 | Every function has tests | PASS/FAIL | <details> |
| AC-T2 | Table-driven tests | PASS/FAIL | <details> |
...

## Failed Criteria
<details on each failure, if any>

## Recommendation
PASS or FAIL

## Failure Signature
<comma-separated list of failed AC IDs, or "none">
```

## Status Reporting

After writing your review, report status:

If `$KILROY_STAGE_STATUS_PATH` is set, write JSON to that path. If that write fails and `$KILROY_STAGE_STATUS_FALLBACK_PATH` is set, write to the fallback path instead.

```json
{"status": "success", "summary": "Review passed all acceptance criteria"}
```

Or on failure:
```json
{"status": "fail", "summary": "Review failed: AC-T1, AC-Q3", "failure_signature": "AC-T1,AC-Q3"}
```

If neither path is set, print the status JSON to stdout as a last resort.
