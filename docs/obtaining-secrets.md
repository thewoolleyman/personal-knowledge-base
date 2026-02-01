# Obtaining and Configuring Secrets

## Interactive Walkthrough (Recommended)

Start Claude Code with the `--chrome` flag so Claude can see your browser and guide you:

```bash
claude --chrome
```

Then tell Claude which secret you need:

```
> Help me set up Google OAuth credentials for pkb
> Help me add Slack credentials to pkb
> Walk me through obtaining the GitHub token for pkb
```

Claude will:
1. Open each URL in your browser
2. Tell you exactly what to click and fill in
3. See your browser (via `--chrome`) if you get stuck and need help
4. Ask you to paste credentials -- then write them to `.env` for you
5. Run the auth flow and verify everything works

You never need to read this doc end-to-end. Just start `claude --chrome` and ask.

## Quick Reference (for when you already know the steps)

```bash
cat .env.example                                    # see what vars are needed
cp .env.example .env                                # copy template
# fill in values, then:
source .env && export PKB_GOOGLE_CLIENT_ID PKB_GOOGLE_CLIENT_SECRET
make build && ./pkb auth                            # authenticate
./pkb search "test query"                           # verify
```

## How the Formula System Works

Credential setup is backed by a reusable beads formula: `.beads/formulas/mol-obtain-secret.formula.json`. This formula handles both OAuth flows (client ID + secret + consent screen) and plain API tokens.

### The workflow

1. **Pour the formula** to create a tracked bead with sub-steps for the specific provider
2. **Start `claude --chrome`** and tell Claude to work on that bead
3. **Claude follows the formula steps**, guiding you through the provider's console
4. **Close the bead** when credentials are verified

### Pouring the formula for a new secret

Preview the formula first:

```bash
bd cook mol-obtain-secret --dry-run
```

Then pour it with the variables for your provider:

```bash
# OAuth example: Google (client ID + client secret + consent screen)
bd --no-daemon mol pour mol-obtain-secret \
  --var provider=google \
  --var service="Google Drive + Gmail" \
  --var secret_type=oauth \
  --var console_url="https://console.cloud.google.com/" \
  --var apis="Google Drive API, Gmail API" \
  --var env_vars="PKB_GOOGLE_CLIENT_ID,PKB_GOOGLE_CLIENT_SECRET" \
  --var scopes="drive.readonly,gmail.readonly"

# Token example: GitHub (single personal access token)
bd --no-daemon mol pour mol-obtain-secret \
  --var provider=github \
  --var service="GitHub" \
  --var secret_type=token \
  --var console_url="https://github.com/settings/tokens" \
  --var env_vars="PKB_GITHUB_TOKEN" \
  --var verify_command="./pkb search 'test'"

# OAuth example: Slack (web app with client ID + secret)
bd --no-daemon mol pour mol-obtain-secret \
  --var provider=slack \
  --var service="Slack" \
  --var secret_type=oauth \
  --var console_url="https://api.slack.com/apps" \
  --var apis="Web API" \
  --var env_vars="PKB_SLACK_CLIENT_ID,PKB_SLACK_CLIENT_SECRET" \
  --var scopes="search:read" \
  --var app_type=web
```

After pouring, run `bd ready` to see the new bead, then start `claude --chrome` and work through it.

### Formula variables reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `provider` | yes | -- | Provider name (google, github, slack, notion) |
| `service` | yes | -- | Display name (Google Drive + Gmail, GitHub, etc.) |
| `secret_type` | no | `oauth` | `oauth` (ID + secret + consent) or `token` (single key) |
| `console_url` | yes | -- | URL to the provider's developer console |
| `apis` | no | `""` | APIs to enable (comma-separated; empty to skip) |
| `env_vars` | yes | -- | Env var names to set (comma-separated) |
| `scopes` | no | `""` | OAuth scopes (ignored for token type) |
| `app_type` | no | `desktop` | OAuth app type (ignored for token type) |
| `verify_command` | no | `./pkb search 'test'` | Command to verify credentials work |

## Google OAuth Credentials (Manual Reference)

These are the steps Claude walks you through interactively. Listed here for reference only.

### 1. Open Google Cloud Console

```bash
open "https://console.cloud.google.com/"
```

Sign in. Create a new project or select existing.

### 2. Enable APIs

```bash
open "https://console.cloud.google.com/apis/library"
```

Search and enable: **Google Drive API** and **Gmail API**.

### 3. Configure OAuth Consent Screen

```bash
open "https://console.cloud.google.com/apis/credentials/consent"
```

- User type: **External**
- App name: anything (e.g., "pkb")
- User support email: your email
- Developer email: your email
- Scopes: add `drive.readonly` and `gmail.readonly`
- Test users: **add your own Google email** (required while app is in "Testing" mode)

### 4. Create Credentials

```bash
open "https://console.cloud.google.com/apis/credentials"
```

- **Create Credentials > OAuth client ID**
- Application type: **Desktop app**
- Name: anything (e.g., "pkb-local")
- **Copy** the Client ID and Client Secret

### 5. Configure Locally

Claude writes these to `.env` for you during the interactive flow. If doing it manually:

```bash
cp .env.example .env
# paste your Client ID and Client Secret into .env
```

### 6. Authenticate and Test

```bash
source .env && export PKB_GOOGLE_CLIENT_ID PKB_GOOGLE_CLIENT_SECRET
make build
./pkb auth          # opens browser for OAuth consent
./pkb search "test" # verify
```

Token is saved to `~/.config/pkb/token.json`.

## Troubleshooting

| Problem | Fix |
|---------|-----|
| "This app is blocked" during OAuth | Add your email as a **test user** in the consent screen |
| "credentials not configured" | `source .env && export PKB_GOOGLE_CLIENT_ID PKB_GOOGLE_CLIENT_SECRET` |
| Token expired | Re-run `./pkb auth` |
| Wrong account | Delete `~/.config/pkb/token.json` and re-run `./pkb auth` |
| Claude can't see browser | Make sure you started with `claude --chrome` |
