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
- **Obsidian** — searches an Obsidian vault mirrored to Google Drive. Restricts search to a specific folder. Requires `PKB_OBSIDIAN_FOLDER_ID` and Google OAuth credentials.
- **Notion** — searches Notion pages and databases via the Notion API. Requires `PKB_NOTION_TOKEN`.

### Future connectors (not yet implemented)

Slack, Google Keep, Dropbox, S3

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
| `make serve` | Build, start the dev server on `:8080`, and open the web UI in a browser |
| `make version` | Print the current version |
| `make tailscale-health` | Check the Tailscale production health endpoint (port 9000) |
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

#### Deployment

| Target | Description |
|--------|-------------|
| `make deploy-local` | Build, install binary, and restart the systemd service |
| `make deploy-status` | Check systemd service status and health endpoint (port 9000) |
| `make deploy-logs` | Show recent service logs |

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

### Obsidian setup

The Obsidian connector searches your vault via Google Drive. It requires:

1. An Obsidian vault mirrored to Google Drive (see [Obsidian vault mirroring](#obsidian-vault-mirroring) above)
2. The Google Drive folder ID of the mirror folder — get it from the URL when viewing the folder in Google Drive (the long string after `/folders/`)
3. Set `PKB_OBSIDIAN_FOLDER_ID` in your `.env`:

```bash
PKB_OBSIDIAN_FOLDER_ID="1tK3Z1ie-CZMAlvBdNb6hNlHfFyrY-mPJ"  # your folder ID
```

The Obsidian connector uses the same Google OAuth token as Drive/Gmail, so no additional auth is needed.

### Notion setup

The Notion connector searches pages and databases shared with your integration:

1. Go to [My Integrations](https://www.notion.so/my-integrations) and create a new **internal** integration
2. Give it a name (e.g. "PKB") and select your workspace
3. Under **Capabilities**, ensure **Read content** is checked (this is the only permission needed)
4. Copy the **Internal Integration Secret** (starts with `ntn_`)
5. Set `PKB_NOTION_TOKEN` in your `.env`:

```bash
PKB_NOTION_TOKEN="ntn_your-integration-token"
```

6. **Share pages/databases with the integration**: open each Notion page or database you want searchable, click the `···` menu → **Connections** → add your integration. Only pages explicitly shared (or child pages of shared pages) will appear in search results.

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
| `PKB_SERVER_ADDR` | `:8080` | HTTP server listen address (production uses `:9000` via systemd) |
| `PKB_GOOGLE_CLIENT_ID` | (none) | Google OAuth client ID |
| `PKB_GOOGLE_CLIENT_SECRET` | (none) | Google OAuth client secret |
| `PKB_TOKEN_PATH` | `~/.config/pkb/token.json` | Path to store OAuth token |
| `PKB_OBSIDIAN_FOLDER_ID` | (none) | Google Drive folder ID for Obsidian vault (enables `obsidian` source) |
| `PKB_NOTION_TOKEN` | (none) | Notion integration token (enables `notion` source) |
| `PKB_TAILSCALE` | `false` | Bind server to Tailscale interface for secure remote access |

## Remote Access via Tailscale

PKB can be securely accessed from anywhere (phone, laptop, other machines) using [Tailscale](https://tailscale.com/) — a zero-config mesh VPN built on WireGuard.

### Why Tailscale?

When `PKB_TAILSCALE=true`, the server binds **only** to your Tailscale network interface. This means:

- **Zero public attack surface** — the server is invisible to the internet
- **No auth middleware needed** — network access IS the authentication
- **All traffic encrypted** — WireGuard encryption between all devices
- **No ports to open** — no firewall rules, no reverse proxy, no TLS certificates to manage

This is critical for PKB because the server holds OAuth tokens and API keys that grant access to your Google Drive, Gmail, Notion, and other accounts.

### Setup

#### 1. Install Tailscale on your server

```bash
# Linux (Ubuntu/Debian)
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up

# macOS
brew install tailscale
```

Follow the auth URL printed to the terminal to add the machine to your tailnet.

#### 2. Install Tailscale on your devices

- **macOS**: `brew install tailscale` or download from [tailscale.com/download/mac](https://tailscale.com/download/mac)
- **iOS**: Install "Tailscale" from the [App Store](https://apps.apple.com/app/tailscale/id1470499037), sign in with the same account
- **Android**: Install "Tailscale" from the [Play Store](https://play.google.com/store/apps/details?id=com.tailscale.ipn), sign in with the same account
- **Windows**: Download from [tailscale.com/download/windows](https://tailscale.com/download/windows)
- **Linux**: `curl -fsSL https://tailscale.com/install.sh | sh && sudo tailscale up`

#### 3. Start PKB in Tailscale mode

```bash
PKB_TAILSCALE=true pkb serve
```

Or add to your `.env`:

```bash
PKB_TAILSCALE=true
```

The server will resolve your machine's Tailscale IP and bind to it automatically. If Tailscale is not running, the server exits with a clear error.

#### 4. Access from any device

Find your server's Tailscale IP:

```bash
tailscale status    # lists all devices and their IPs
```

Then open in a browser on any device connected to your tailnet:

```
http://<tailscale-ip>:9000
```

(Dev server uses port 8080; production uses 9000.)

#### Optional: MagicDNS for friendly hostnames

In the [Tailscale admin console](https://login.tailscale.com/admin/dns), enable MagicDNS to get a human-readable hostname like:

```
http://my-server.tailnet-name.ts.net:9000
```

#### Optional: HTTPS with Tailscale certificates

Enable HTTPS in the Tailscale admin console for automatic TLS certificates. Then access via:

```
https://my-server.tailnet-name.ts.net:9000
```

#### Optional: ACL lockdown

Restrict which devices can reach PKB by editing [Access Controls](https://login.tailscale.com/admin/acls) in the Tailscale admin console:

```json
{
  "acls": [
    {"action": "accept", "src": ["autogroup:owner"], "dst": ["tag:pkb:9000"]}
  ],
  "tagOwners": {
    "tag:pkb": ["autogroup:owner"]
  }
}
```

Then tag your server: `sudo tailscale up --advertise-tags=tag:pkb`

#### Run as a systemd user service (production)

The production service file is at `deploy/pkb.service`. It sets `PKB_SERVER_ADDR=:9000` so the production server does not conflict with a dev server on the default port 8080.

Deploy it with:

```bash
make deploy-local
```

This copies the binary to `~/.local/bin/pkb`, installs the systemd user service, and starts it. The service auto-restarts on failure.

To check status: `make deploy-status`
To view logs: `make deploy-logs`

## CI/CD Deployment

The CI pipeline (`.github/workflows/ci-cd.yml`) automatically deploys to production on every push to `main` after tests pass and a release is created.

### How it works

1. CI runs all tests (unit, acceptance, live, e2e)
2. Builds release artifacts for Linux, macOS, Windows
3. Creates a GitHub Release
4. SSHes to the production server via Tailscale and runs `make deploy-local`
5. Verifies the health endpoint responds on port 9000

### Required GitHub secrets

Set these at https://github.com/thewoolleyman/personal-knowledge-base/settings/secrets/actions:

| Secret | Description |
|--------|-------------|
| `TS_OAUTH_CLIENT_ID` | Tailscale OAuth client ID (created at https://login.tailscale.com/admin/settings/oauth) |
| `TS_OAUTH_SECRET` | Tailscale OAuth client secret |
| `DEPLOY_TARGET` | SSH target for deployment (e.g. `ubuntu@100.89.189.118`) |

### Tailscale OAuth setup for CI

1. Go to https://login.tailscale.com/admin/settings/oauth
2. Click **Generate OAuth Client**
3. Set scope: **`auth_keys` — Write**
4. Set tag: **`tag:ci`**
5. Click **Generate credential** and save the Client ID and Secret as GitHub secrets

### Tailscale ACL requirements

The `tag:ci` node needs SSH access to the deploy target. Add to your [ACLs](https://login.tailscale.com/admin/acls):

```json
{
  "tagOwners": {
    "tag:ci": ["autogroup:admin"]
  },
  "ssh": [
    {
      "action": "accept",
      "src": ["tag:ci"],
      "dst": ["autogroup:self"],
      "users": ["ubuntu"]
    }
  ]
}
```

The deploy target must have Tailscale SSH enabled: `sudo tailscale up --ssh`

## License

See [LICENSE](LICENSE).
