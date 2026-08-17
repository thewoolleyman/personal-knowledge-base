## Why

The CI/CD Pipeline's deploy stage authenticates to the Tailscale network with
`tags: tag:ci` via `tailscale/github-action@v3`, then SSHes to the deploy
target over that tailnet connection. `tag:ci` was removed from the peer
`tailscale-admin` repository's policy on 2026-08-17 (maintainer ruling
retiring the tailnet tag apparatus fleet-wide — see that repo's PR #33 and
`AGENTS.md` "Related Repos"). The deploy stage's Tailscale setup step is
parked (commented out) as of this proposal rather than left to fail on an
undefined tag every push/PR run.

## What Changes

- Replace the Tailscale-mediated deploy path (`tailscale/github-action@v3`
  with `tags: tag:ci`, then `ssh` over the resulting tailnet connection) with
  ordinary keyed OpenSSH over a host/IP-scoped network path — no tailnet tag
  involved on either side.
- Until this lands, the deploy stage is a documented no-op (parked, not
  silently broken): the "Setup Tailscale" and "Deploy via Tailscale SSH"
  steps are disabled with a pointer to this proposal.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `ci-cd`: deploy stage's connectivity mechanism changes from
  tag-scoped Tailscale to plain keyed OpenSSH.

## Impact

- `.github/workflows/ci-cd.yml` deploy stage.
- Deploy-target host's `authorized_keys` (or equivalent) needs the CI
  deploy key once this lands; no tailnet-side change is needed since the
  ordinary-member `autogroup:member -> autogroup:member` grant already
  covers reachability, mirroring `tailscale-admin`'s own guidance in
  `AGENTS.md` "SSH ACL limitations": "For CI deployments to an untagged
  member, use a network grant from the ephemeral source tag to a stable
  host/IP selector on TCP/22, then authenticate with ordinary OpenSSH."
