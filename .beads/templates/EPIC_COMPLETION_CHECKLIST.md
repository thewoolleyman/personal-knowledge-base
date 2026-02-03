# Epic Completion Checklist

Epic ID: `________`
Epic Title: `________`
Completed By: `________`
Date: `________`

## 1. Core Requirements
- [ ] All child beads closed
- [ ] CI green on main
- [ ] Code merged
- [ ] Documentation updated

## 2. User-Facing Functionality Check

**Does this epic add or modify user-facing features?**
- [ ] YES - Complete section 3 below (MANDATORY)
- [ ] NO - Skip to section 4

## 3. Acceptance Test Requirements (MANDATORY for user-facing)

### 3.1 Test Existence
- [ ] Acceptance tests written in `tests/acceptance/`
- [ ] Tests use `//go:build acceptance` tag
- [ ] Tests build real binary with `buildBinary(t)`
- [ ] Tests execute as subprocess with `exec.Command`
- [ ] Tests verify stdout/stderr/exit codes
- [ ] Tests NEVER import `internal/*` packages

### 3.2 Test Coverage
- [ ] Every new CLI command has test: `TestAcceptance_<Command>*`
- [ ] Every new HTTP endpoint has test: `TestAcceptance_<Endpoint>*`
- [ ] Every README example has corresponding test
- [ ] Error cases tested (bad input, missing config, etc.)

### 3.3 Test Execution
Run and verify:
```bash
make test-accept
```
- [ ] All tests pass
- [ ] No tests skipped (unless documented)
- [ ] Tests would catch regression if feature broke

### 3.4 Manual Verification
- [ ] Followed README steps manually
- [ ] Built binary: `make build`
- [ ] Ran each new command
- [ ] Verified output matches expectations
- [ ] Checked error handling

## 4. Sign-off

I verify that:
- All checklist items above are complete
- If user-facing, acceptance tests exist and pass
- Epic can be safely closed

Signature: `________`
Date: `________`

---

**IMPORTANT:** If any acceptance tests cannot be written yet, file a blocker bead and DO NOT close this epic.
