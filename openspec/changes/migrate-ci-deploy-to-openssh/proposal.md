## Why

The production deployment formerly joined Tailscale as `tag:ci` and used
Tailscale SSH to a VPS tagged `tag:vps`. The tailnet tag apparatus has since
been retired fleet-wide (maintainer ruling, 2026-08-17): `tag:ci` and `tag:vps`
no longer exist, and the VPS is now an ordinary untagged `autogroup:member`.
That retirement makes the old tag-based deployment authorization invalid.

The tailscale-admin policy now provides reachability through a single flat
`autogroup:member -> autogroup:member` grant: once the deploy runner is itself
an untagged member, it reaches the untagged VPS on all ports (TCP/22 included)
with no source tag. (How an ephemeral GitHub Actions runner becomes an
`autogroup:member` is unsolved and deferred — see `tasks.md`; this change does
not solve it.) This change must migrate the deployment to ordinary OpenSSH
authentication over that private path.

## What Changes

- Configure a dedicated, least-privilege OpenSSH deploy identity for the
  production VPS; do not restore or invent a destination device tag.
- Update the GitHub Actions deployment to load that identity and connect to
  the VPS's Tailscale address using ordinary `sshd`.
- Pin and verify the VPS SSH host key instead of disabling strict host-key
  checking.
- Rely on the flat `autogroup:member` tailnet grant for encrypted network
  reachability once the runner is a member; do not reintroduce a source tag,
  and do not add a Tailscale `ssh` policy rule.
- Fail deployment clearly when required credentials or host-key material is
  absent; do not silently skip it.
- Verify the deploy and port-9000 health check end to end before declaring the
  migration complete.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `infrastructure`: Production deployment authentication changes from
  Tailscale SSH to ordinary key-authenticated OpenSSH over Tailscale.
- `authentication`: Defines the dedicated CI deploy identity, host-key
  verification, and secret-handling boundary.

## Impact

- Affected files include `.github/workflows/ci-cd.yml`, deployment acceptance
  coverage, and deployment documentation.
- A human must install the reviewed public key on the VPS and enter private-key
  and known-host material into GitHub Actions secrets. Repository agents are
  forbidden from setting CI/CD secrets by the factory constraints.
- The production deployment is expected to fail between retirement of the
  `tag:vps` Tailscale SSH path and completion of this change. That outage is
  explicitly accepted; it must remain visible rather than being bypassed with
  a temporary destination tag or public-network SSH route.
- This pattern applies to future CI deployments to ordinary tailnet machines:
  once the runner is an untagged `autogroup:member`, the flat member-to-member
  grant provides reachability to the machine's Tailscale address, while ordinary
  SSH independently authenticates the workload. No source tag is used.
