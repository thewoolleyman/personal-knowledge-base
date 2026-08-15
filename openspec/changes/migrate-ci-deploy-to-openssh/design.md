## Context

Tailscale separates network grants from Tailscale SSH authorization. A tagged
CI node can receive TCP reachability to an untagged member, but Tailscale SSH
will not authorize tagged-to-user-owned SSH. The VPS therefore keeps ordinary
`sshd`, and CI supplies conventional cryptographic authentication.

## Goals / Non-Goals

**Goals**

- Restore private CI deployment without tagging the VPS.
- Authenticate CI independently of tailnet reachability.
- Verify the server host key and keep all secret values human-managed.

**Non-Goals**

- Change tailnet policy in this repository.
- Expose or use public SSH for deployment.
- Have an agent create or write GitHub Actions secrets.

## Decisions

### 1. Use ordinary OpenSSH over the Tailscale IP

The network layer remains WireGuard-protected and policy-scoped, while sshd
handles identity. This preserves the flat user-owned destination model and
avoids a destination tag introduced solely as a protocol workaround.

### 2. Pin host identity

The current workflow disables host-key verification. Migration must instead
use maintainer-reviewed known-host material stored as a GitHub secret or other
reviewed secure input. `ssh-keyscan` during the same run is not authentication.

### 3. Human performs secret mutations

Repository constraints prohibit agents from setting CI secrets. Implementation
may name and validate inputs, but the maintainer installs the public key and
enters the private key and host-key values.

## Risks / Trade-offs

- **Deployment remains broken until human setup** -> Keep failure explicit and
  provide exact validation steps during implementation.
- **Long-lived key theft** -> Use a dedicated identity, restrict its server-side
  authorization, and rotate/revoke it independently.
- **Tailscale IP change** -> Coordinate the fixed `/32`, workflow destination,
  and known-host entry through reviewed changes.

## Migration Plan

1. Generate or select a dedicated deploy identity through a maintainer-reviewed
   procedure.
2. Install only its public identity on the VPS with least-privilege sshd
   authorization.
3. Have the maintainer enter the private identity and reviewed host key into
   GitHub Actions secrets.
4. Update the workflow and tests, then deploy over the VPS Tailscale address.
5. Verify the remote update, systemd service restart, and `/health` response.
6. Remove obsolete Tailscale SSH-specific documentation and secrets only after
   successful verification.

## Open Questions

- Whether server-side least privilege should use a dedicated Unix account,
  `authorized_keys` restrictions, or an SSH certificate principal must be
  decided during review of this change.
