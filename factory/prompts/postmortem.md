# PKB Pipeline Postmortem

You are analyzing a pipeline failure. Your job is to diagnose what went wrong and write repair guidance for the next implementation iteration.

## Instructions

1. Read the pipeline stage outputs and logs to identify which gate or review failed
2. Examine the code changes that caused the failure
3. Identify the root cause
4. Write specific, actionable repair guidance
5. Output to `.ai/postmortem_latest.md`

## Failure Modes

Common failure patterns in the PKB pipeline:

- **Vet failure**: Suspicious constructs, unreachable code, incorrect format strings
- **Lint failure**: golangci-lint violations (unused vars, error handling, style)
- **Build failure**: Compilation errors, missing imports, type mismatches
- **Test failure**: Unit test assertions failing, race conditions detected
- **Acceptance test failure**: Binary behavior doesn't match expected CLI output
- **Review failure**: Specification fidelity issues, missing tests, constraint violations

## Output Format

Write to `.ai/postmortem_latest.md`:

```markdown
# Postmortem

## Summary
<1-2 sentences: what failed and why>

## Evidence
<Specific error messages, log lines, or test output that show the failure>

## Root Cause
<What specifically caused the failure — file, function, line if possible>

## Repair Guidance
<Specific instructions for the next implementation iteration:
 - Which files to modify
 - What changes to make
 - What to preserve (don't break working code)>

## Verification
<How to verify the repair worked — specific commands to run>
```

## Critical Rules

- **Be specific**: Name exact files, functions, and line ranges
- **Preserve working code**: Clearly state what is already correct and must not be changed
- **One root cause**: Focus on the primary failure, not secondary symptoms
- **Actionable guidance**: The implementing agent should be able to follow your instructions without guessing

## Status Reporting

After writing the postmortem, report status:

If `$KILROY_STAGE_STATUS_PATH` is set, write JSON to that path. If that write fails and `$KILROY_STAGE_STATUS_FALLBACK_PATH` is set, write to the fallback path instead.

```json
{"status": "success", "summary": "Postmortem complete: <root cause summary>"}
```

If neither path is set, print the status JSON to stdout as a last resort.
