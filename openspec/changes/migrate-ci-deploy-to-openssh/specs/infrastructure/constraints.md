## MODIFIED Requirements

### Requirement: Private production deployment transport
The production deployment SHALL connect to the untagged VPS using ordinary
OpenSSH over its Tailscale address. The ephemeral runner SHALL reach the VPS as
an untagged `autogroup:member`, whose tailnet policy grant
(`autogroup:member -> autogroup:member`) covers reachability without a source
tag. The workflow SHALL NOT require a source or destination tag or Tailscale SSH
authorization, and SHALL NOT fall back to the VPS's public interface.

#### Scenario: CI deploys over the private tailnet path
- **WHEN** a release deployment runs with its required credentials
- **THEN** it SHALL reach the VPS as an untagged Tailscale `autogroup:member` and invoke ordinary OpenSSH against the configured Tailscale destination

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
