# Infrastructure — Intent

Configuration, networking, deployment, and CI/CD.

## Purpose

Infrastructure covers everything needed to build, deploy, and run the system. This includes configuration management, the CI/CD pipeline, the release process, networking via Tailscale, and the production deployment.

## Current Capabilities

- **Configuration** — Environment-based config via `.env` with auto-loading on startup. `.env.example` documents all required entries.
- **CI/CD** — Single `ci-cd.yml` GitHub Actions pipeline: lint (golangci-lint, gitleaks, go vet, actionlint), unit tests, acceptance tests, live API tests, e2e tests, deployment. Auto-creates P0 bug beads on main-branch failures.
- **Release** — Cross-compiled CLI binaries via GitHub Release workflow. Version management for release tagging.
- **Deployment** — Production server deployed and accessible via Tailscale. Dev server on port 8080, production on port 9000 (planned).
- **Networking** — Tailscale for secure remote access without exposing ports to the public internet.

## Domain Model

- **Environment config** — `.env` file with credentials, paths, and settings
- **Pipeline** — the CI/CD workflow that validates, builds, and deploys
- **Release** — a tagged version with cross-compiled binaries
- **Production server** — the deployed instance accessible via Tailscale

## Behavioral Narratives

### Local development
A developer clones the repo, copies `.env.example` to `.env`, fills in credentials, and runs `make` targets. `make help` shows all available commands. `make run` builds and runs with arguments.

### CI pipeline
Every push to main triggers the pipeline: lint → unit tests → acceptance tests → live API tests → e2e tests → deploy. If any stage fails, a P0 bug bead is auto-created. CI never silently skips tests.

### Production deployment
The server runs on a machine accessible via Tailscale. Remote clients connect through Tailscale to access the web UI and API. The production port (9000) does not conflict with the dev server port (8080).
