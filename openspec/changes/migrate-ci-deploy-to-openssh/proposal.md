## Why

The production deployment currently joins Tailscale as `tag:ci` and uses
Tailscale SSH to a VPS tagged `tag:vps`. The tailnet has intentionally retired
persistent tags for non-Nix machines, so the VPS is becoming an ordinary
untagged member. Tailscale SSH prohibits a tagged source from targeting a
user-owned device, which makes the existing deployment authorization invalid.

The tailscale-admin policy now provides only network reachability from
`tag:ci` to the VPS's fixed Tailscale `/32` on TCP/22. This change must migrate
the deployment to ordinary OpenSSH authentication over that private path.

## What Changes

- Configure a dedicated, least-privilege OpenSSH deploy identity for the
  production VPS; do not restore or invent a destination device tag.
- Update the GitHub Actions deployment to load that identity and connect to
  the VPS's Tailscale address using ordinary `sshd`.
- Pin and verify the VPS SSH host key instead of disabling strict host-key
  checking.
- Keep the ephemeral `tag:ci` Tailscale session only for encrypted network
  reachability. Do not add a Tailscale `ssh` policy rule.
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
- The production deployment is expected to fail between removal of
  `tag:vps`/its Tailscale SSH rule and completion of this change. That outage is
  explicitly accepted; it must remain visible rather than being bypassed with
  a temporary destination tag or public-network SSH route.
- This pattern applies to future CI deployments to ordinary tailnet machines:
  a narrowly scoped `tag:ci` network grant targets the machine's Tailscale
  `/32` and port, while ordinary SSH independently authenticates the workload.
