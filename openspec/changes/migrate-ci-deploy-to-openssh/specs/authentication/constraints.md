## ADDED Requirements

### Requirement: Dedicated OpenSSH deploy identity
CI SHALL authenticate to the VPS with a dedicated deploy key or certificate
whose server-side authorization is limited to the production deployment use
case. The private credential SHALL exist only in GitHub Actions secret storage
and runtime memory and SHALL never be committed or printed.

#### Scenario: CI authenticates to sshd
- **WHEN** the runner reaches the VPS on TCP/22
- **THEN** ordinary sshd SHALL authenticate the dedicated CI identity without relying on Tailscale SSH user authorization

### Requirement: VPS host identity is verified
The workflow SHALL verify a pinned VPS SSH host key. It SHALL NOT use
`StrictHostKeyChecking=no` or automatically trust an unverified key obtained in
the same deployment run.

#### Scenario: VPS host key does not match
- **WHEN** the presented host key differs from the reviewed pinned value
- **THEN** SSH and the deployment SHALL fail before remote commands execute
