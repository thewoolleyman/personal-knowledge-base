# Personal Knowledge Base (PKB)

A personal searchable knowledge base that aggregates data across services (Google Drive, Gmail, Slack, Notion, etc.) with an AI-native CLI/TUI interface. Built in Go. Runs locally during development, designed to deploy to a VPS later.

## Installation

Download a pre-built binary from [GitHub Releases](https://github.com/thewoolleyman/personal-knowledge-base/releases/latest).

### macOS

```bash
# Set version (find latest at https://github.com/thewoolleyman/personal-knowledge-base/releases/latest)
VERSION=0.3.26  # replace with latest version

# Download (replace arm64 with amd64 for Intel Macs)
curl -L -O https://github.com/thewoolleyman/personal-knowledge-base/releases/download/v${VERSION}/pkb-darwin-arm64-v${VERSION}.zip

# Extract (creates pkb-darwin-arm64-v${VERSION}/ subdirectory)
unzip pkb-darwin-arm64-v${VERSION}.zip

# Remove macOS quarantine if present (set by browsers, not curl), make executable, and install
cd pkb-darwin-arm64-v${VERSION}
xattr -d com.apple.quarantine pkb 2>/dev/null || true
chmod +x pkb
sudo mv pkb /usr/local/bin/
```

### Linux

```bash
# Set version (find latest at https://github.com/thewoolleyman/personal-knowledge-base/releases/latest)
VERSION=0.3.26  # replace with latest version

# Download (replace amd64 with arm64 for ARM)
curl -L -O https://github.com/thewoolleyman/personal-knowledge-base/releases/download/v${VERSION}/pkb-linux-amd64-v${VERSION}.zip

# Extract and install
unzip pkb-linux-amd64-v${VERSION}.zip
cd pkb-linux-amd64-v${VERSION}
chmod +x pkb
sudo mv pkb /usr/local/bin/
```

### Windows

Download the `pkb-windows-amd64-v<VERSION>.zip` for your version from the [latest release](https://github.com/thewoolleyman/personal-knowledge-base/releases/latest), extract the `pkb.exe` from the subdirectory, and add it to your PATH.

### Verify

```bash
pkb version
```

## Architecture

```
CLI / TUI Client (Bubble Tea)
  - Natural language & keyword search
  - Results with source links
        │
        │ HTTP (localhost during dev)
        ▼
Go API Server
  - Fan-out search to connected services
  - Aggregate & rank results
  - Credential management (OAuth tokens)
        │
   ┌────┼────┬────┐
   ▼    ▼    ▼    ▼
Google Gmail Slack Notion  ... (future)
Drive  API   API   API
 API
```

All consumers (CLI, TUI, web UI) go through the same HTTP API. The `search` and `interactive` commands start an embedded server on an ephemeral port, make HTTP requests via `apiclient`, and shut down on exit. The `serve` command runs a long-lived server for the web UI and external clients.

### Key packages

| Package | Purpose |
|---------|---------|
| `cmd/pkb` | CLI entry point (Cobra) with `search`, `serve`, `interactive`, `auth`, and `version` commands |
| `internal/apiclient` | HTTP client for the PKB API — used by CLI and TUI to dogfood the server |
| `internal/server` | HTTP API server with `/health` and `/search` endpoints |
| `internal/search` | Search engine — fans out queries to connectors concurrently, supports source filtering |
| `internal/connectors` | `Connector` interface that each data source implements |
| `internal/connectors/gdrive` | Google Drive connector (search via Drive API) |
| `internal/connectors/gmail` | Gmail connector (search via Gmail API) |
| `internal/auth` | OAuth2 authorization code flow with local callback server |
| `internal/config` | Configuration loading from environment variables |
| `internal/tui` | Interactive Bubble Tea TUI for search |
| `internal/web` | Embedded web UI (HTML/JS/CSS) served from the Go binary |

### Current connectors

- **Google Drive** — searches files via `fullText contains` query. Requires OAuth2 credentials.
- **Gmail** — searches email messages via Gmail API. Uses same OAuth2 token as Drive.

### Future connectors (not yet implemented)

Slack, Notion, Google Keep, Dropbox, S3

### Obsidian vault mirroring

A launchd agent rsyncs your Obsidian vault to a Google Drive mount every 10 minutes. Google Drive indexes the files, making them searchable via the `google-drive` connector.

```bash
# Install and start the mirror (interactive — prompts for confirmation)
bash docs/setup-obsidian-gdrive-mirror.sh

# Verify it's running
launchctl print gui/$(id -u)/com.user.rsync-obsidian-to-gdrive

# Check logs
tail ~/.local/log/rsync-obsidian.log
```

See [docs/obsidian-vault-mirroring-notes.md](docs/obsidian-vault-mirroring-notes.md) for details.

## Development

Strict TDD (Red-Green-Refactor). Every line of implementation code exists because a test demanded it.

### Prerequisites

- Go 1.25+ (`brew install go`)
- `make` (pre-installed on macOS)
- [mise](https://mise.jdx.dev/) for tool management (`brew install mise && mise install`)
- Google Cloud project with Drive API and Gmail API enabled (for integration/live tests)

### Quick start

```bash
make help          # see all available targets
make test          # unit tests with race detection
make test-accept   # acceptance tests (builds binary, tests like a user)
make build         # compile the pkb binary
```

### Make targets

All developer commands live in the `Makefile`. Run `make help` for the full list.

#### Build & run

| Target | Description |
|--------|-------------|
| `make build` | Compile the `pkb` binary |
| `make run` | Build and run `pkb` with args (e.g. `make run search "agentic"`) |
| `make serve` | Build, start the server on `:8080`, and open the web UI in a browser |
| `make version` | Print the current version |
| `make clean` | Remove build artifacts |

#### Testing

| Target | Credentials needed | Description |
|--------|-------------------|-------------|
| `make test` | None | Unit tests with race detection and coverage |
| `make test-accept` | None | Acceptance tests — builds real binary, tests from user perspective |
| `make test-all` | None | Unit + acceptance tests |
| `make test-int` | OAuth token | Component integration tests against real Google APIs |
| `make test-live` | OAuth token | Live API tests with real credentials and token |
| `make test-e2e` | OAuth token + Playwright | Playwright E2E tests for the web UI |
| `make test-full` | OAuth token | Unit + acceptance + integration tests |

#### Manual exploratory testing

A test page exists across multiple connected services for manual smoke-testing. Search for it to verify connectors are working end-to-end:

```bash
pkb search "PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE"
```

Expected output (results may vary as connectors are added):

```
1. PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE (Gmail)
   source: gmail This is a test page for live integration testing of https://git...
   https://mail.google.com/mail/u/0/#inbox/19c1d1362b867da8
   [gmail]

2. PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE (google-drive)
   https://drive.google.com/drive/folders/18CsdYxWehQgrDQ6inaH4bGOUM9wo_uN1
   [google-drive]
```

If you see results from each configured connector, the integration is healthy.

#### Code quality

| Target | Description |
|--------|-------------|
| `make lint` | Run `golangci-lint` and `actionlint` (via mise) |
| `make lint-actions` | Lint GitHub Actions workflow files only |
| `make vet` | Run `go vet` |
| `make tidy` | Tidy `go.mod` and verify no uncommitted changes |

#### Security & hooks

| Target | Description |
|--------|-------------|
| `make scan-secrets` | Run gitleaks to detect hardcoded secrets in the repo |
| `make scan-secrets-staged` | Run gitleaks on staged files only (same check as pre-commit hook) |
| `make setup-hooks` | Install the pre-commit hook (gitleaks + beads export) |
| `make verify-hooks` | Verify two-tier logging, context bundles, and recall work end-to-end |

#### CI

| Target | Description |
|--------|-------------|
| `make open-cicd-webpage` | Open the GitHub Actions CI/CD page in a browser |

### Google OAuth setup

Required for `make test-int`, `make test-live`, and `make test-e2e`:

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a project (or use existing)
3. Enable the **Google Drive API** and **Gmail API**
4. Create OAuth 2.0 credentials (Desktop application type)
5. Copy `.env.example` to `.env` and fill in values:

```bash
PKB_GOOGLE_CLIENT_ID="your-client-id"
PKB_GOOGLE_CLIENT_SECRET="your-client-secret"
```

### Using the CLI

```bash
make run search "meeting notes"    # search (builds first)
make serve                         # start server + open web UI
make run interactive               # launch TUI (alias: make run tui)
```

### HTTP API endpoints

When the server is running (`make serve` or `./pkb serve`):

| Endpoint | Description |
|----------|-------------|
| `GET /` | Web UI |
| `GET /health` | Health check (200 OK) |
| `GET /search?q=<query>` | Search all sources, returns JSON |
| `GET /search?q=<query>&sources=gdrive` | Filter to specific connectors (comma-separated) |

## Configuration

All config is via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PKB_SERVER_ADDR` | `:8080` | HTTP server listen address |
| `PKB_GOOGLE_CLIENT_ID` | (none) | Google OAuth client ID |
| `PKB_GOOGLE_CLIENT_SECRET` | (none) | Google OAuth client secret |
| `PKB_TOKEN_PATH` | `~/.config/pkb/token.json` | Path to store OAuth token |

## License

See [LICENSE](LICENSE).
