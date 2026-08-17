> **NOTE 2026-08-17:** `tag:ci` itself is now GONE from `tailscale-admin/policy.hujson`
> (PR #33, tailnet tag apparatus retired fleet-wide) — not merely deprecated for
> this deployment's SSH usage. Task 3.2's premise ("keep `tag:ci` only for
> Tailscale network reachability") no longer holds: there is no `tag:ci` left to
> keep, for reachability or anything else. This repo's `.github/workflows/ci-cd.yml`
> deploy stage's three Tailscale-dependent steps (Setup Tailscale, Deploy via
> Tailscale SSH, Verify deployment health) are parked (commented out) as of the
> same date so CI stops requesting an undefined tag on every push. This migration
> now needs a plain host/IP-scoped path with NO ephemeral source tag at all (the
> `autogroup:member -> autogroup:member` grant already covers reachability to the
> ordinary-member VPS), not a `tag:ci`-sourced network grant as originally scoped.
> Re-scope task 3.2 accordingly before resuming this work.

## 1. Define and test the workflow contract

- [ ] 1.1 Add tests that require ordinary OpenSSH inputs, pinned host verification, and a Tailscale-only destination.
- [ ] 1.2 Prove tests reject `StrictHostKeyChecking=no`, public-address fallback, and silent skipping for missing deploy inputs.

## 2. Prepare the human-managed identity

- [ ] 2.1 Document the reviewed dedicated deploy identity and least-privilege server-side authorization.
- [ ] 2.2 Have the maintainer install the public identity on the VPS.
- [ ] 2.3 Have the maintainer set the private identity and pinned known-host material as GitHub Actions secrets; agents SHALL NOT perform this task.

## 3. Migrate deployment

- [ ] 3.1 Update the workflow to load the OpenSSH identity and known-host data without logging either.
- [ ] 3.2 Keep `tag:ci` only for Tailscale network reachability and remove all Tailscale SSH assumptions.
- [ ] 3.3 Update deployment documentation to describe the reusable CI-to-untagged-tailnet-host pattern.

## 4. Verify end to end

- [ ] 4.1 Run repository quality gates and confirm the workflow fails safely when credentials or the host key are wrong.
- [ ] 4.2 Complete a production deployment over the Tailscale `/32` and verify the service health endpoint on port 9000.
- [ ] 4.3 Confirm no public-network SSH fallback and no destination tag were introduced.
