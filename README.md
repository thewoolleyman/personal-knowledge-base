# Personal Knowledge Base (PKB)

A personal searchable knowledge base that aggregates data across services (Google Drive, Gmail, Slack, Notion, etc.) with an AI-native CLI/TUI interface. Built in Go. Runs locally during development, designed to deploy to a VPS later.

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

### Current connectors

- **Google Drive** — searches files via `fullText contains` query. Requires OAuth2 credentials.
- **Gmail** — searches email messages via Gmail API. Uses same OAuth2 token as Drive.

### Future connectors (not yet implemented)

Slack, Notion, Google Keep, Dropbox, S3

## Development

All development on this project uses [Claude Flow](https://github.com/ruvnet/claude-flow) with strict TDD (Red-Green-Refactor). Every line of implementation code exists because a test demanded it.

### Prerequisites

- Go 1.25+ (`brew install go`)
- `golangci-lint` (optional, for `make lint`: `brew install golangci-lint`)
- `make` (pre-installed on macOS)
- Google Cloud project with Drive API enabled (for Google Drive search)

### Quick start

```bash
make help          # see all available targets
make test          # run unit tests
make test-accept   # run acceptance tests (builds real binary, tests UX)
make build         # compile the pkb binary
```

### All make targets

Run `make help` to see what's available:

```
make build         Compile the pkb binary
make test          Run unit tests with race detection and coverage
make test-accept   Run acceptance tests (builds real binary, tests from user perspective)
make test-int      Run component integration tests (requires Google Drive credentials)
make test-all      Run unit, acceptance, and integration tests
make lint          Run golangci-lint
make vet           Run go vet
make tidy          Tidy and verify go.mod
make clean         Remove build artifacts
make run           Build and run pkb --help
make verify-hooks  Prove two-tier logging, context bundles, and recall work end-to-end
```

### CLI search

```bash
make build
./pkb search "meeting notes"
```

### HTTP API server

```bash
make build
./pkb serve              # listens on :8080 by default
./pkb serve --addr :3000 # custom port
```

Endpoints:
- `GET /health` — returns 200 OK
- `GET /search?q=<query>` — returns JSON array of results
- `GET /search?q=<query>&sources=gdrive` — filter to specific connectors (comma-separated)

### Interactive TUI

```bash
make build
./pkb interactive   # or: ./pkb tui
```

## Exploratory testing and acceptance for humans

These steps verify things work from a user's perspective. They mirror the automated acceptance tests in `tests/acceptance/`.

### 1. Verify the project builds and tests pass

```bash
cd personal-knowledge-base
make test            # unit tests — everything should pass
make test-accept     # acceptance tests — builds real binary, tests like a user would
```

Expected: all pass, no race conditions detected.

### 2. Build and try the CLI

```bash
make run             # builds and runs ./pkb --help
```

Expected: prints help text with `search`, `serve`, and `interactive` subcommands listed.

### 3. Try the search command (without credentials)

```bash
make build
./pkb search "test query"
```

Expected: tells you exactly which environment variables to set, with copy-pasteable `export` commands.

### 4. Set up Google Drive OAuth credentials

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a project (or use existing)
3. Enable the **Google Drive API** and **Gmail API**
4. Create OAuth 2.0 credentials (Desktop application type)
5. Set environment variables (see `.env.example` for reference):

```bash
export PKB_GOOGLE_CLIENT_ID="your-client-id"
export PKB_GOOGLE_CLIENT_SECRET="your-client-secret"
```

**Tip:** Add these to `~/.zshrc` or `~/.bashrc` to persist across sessions.

### 5. Run integration tests against real Google Drive

```bash
# Set credentials first (see step 4), then:
make test-int
```

Expected: tests search your actual Google Drive and return results for known files.

### 6. Verify the Obsidian sync is working

```bash
# Check if the launch agent is active
launchctl print gui/$(id -u)/com.user.rsync-obsidian-to-gdrive

# Check recent sync logs
tail -20 ~/.local/log/rsync-obsidian.log

# Dry-run to see what would sync
rsync -avn --delete \
  "/Users/<your-username>/-Obsidian-Default-Vault/" \
  "/Users/<your-username>/Library/CloudStorage/GoogleDrive-<your-email>/My Drive/Personal_Knowledge_Base_Mirrors/Obsidian_Default_Vault/"
```

Expected: launch agent is active, logs show recent successful syncs, dry-run shows no pending changes (already in sync).

### 7. Search for an Obsidian note via Google Drive

Once sync is running and OAuth is configured:

```bash
./pkb search "some known note title from your vault"
```

Expected: returns the matching file from Google Drive with a link to view it.

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
