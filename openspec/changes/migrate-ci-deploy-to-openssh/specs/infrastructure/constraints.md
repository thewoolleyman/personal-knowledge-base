## MODIFIED Requirements

### Requirement: Private production deployment transport
The production deployment SHALL connect to the untagged VPS using ordinary
OpenSSH over its Tailscale address. The ephemeral runner SHALL join as
`tag:ci`, whose tailnet policy grants only TCP/22 to the VPS `/32`. The
workflow SHALL NOT require a destination tag or Tailscale SSH authorization,
and SHALL NOT fall back to the VPS's public interface.

#### Scenario: CI deploys over the private tailnet path
- **WHEN** a release deployment runs with its required credentials
- **THEN** it SHALL join Tailscale as `tag:ci` and invoke ordinary OpenSSH against the configured Tailscale destination

#### Scenario: Private path is unavailable
- **WHEN** the runner cannot reach the VPS Tailscale address on TCP/22
- **THEN** deployment SHALL fail without retrying through a public address

### Requirement: Deployment remains mandatory when configured
Required OpenSSH identity and host-verification inputs SHALL be validated before
deployment. Missing required inputs SHALL fail the job and SHALL NOT silently
skip production deployment.

#### Scenario: Deploy credential is missing
- **WHEN** the deployment job lacks its private key or known-host input
- **THEN** the job SHALL fail before attempting SSH
