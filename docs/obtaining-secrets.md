# Obtaining and Configuring Secrets

When Claude discovers that a new credential is needed (or you know you need one), follow these steps.

## 1. Pour the formula to create a tracked bead

The `mol-obtain-secret` formula creates a molecule (epic + step beads) for obtaining any credential -- OAuth or plain token.

```bash
# Preview what the formula does
bd cook mol-obtain-secret --dry-run

# Pour it for your provider (see examples below)
bd --no-daemon mol pour mol-obtain-secret \
  --var provider=google \
  --var service="Google Drive + Gmail" \
  --var secret_type=oauth \
  --var console_url="https://console.cloud.google.com/" \
  --var apis="Google Drive API, Gmail API" \
  --var env_vars="PKB_GOOGLE_CLIENT_ID,PKB_GOOGLE_CLIENT_SECRET" \
  --var scopes="drive.readonly,gmail.readonly"
```

The pour output prints the root epic ID (e.g., `personal-knowledge-base-mol-6d8`). Note it for step 2.

## 2. Start a `claude --chrome` session and work the molecule

Molecule steps are hidden from plain `bd ready` -- use `--mol` to see them:

```bash
# Check which steps are ready (use the epic ID from step 1)
bd --no-daemon ready --mol <epic-id>
```

Then start Claude with browser access:

```bash
claude --chrome
```

Paste this prompt (replace `<epic-id>` with the actual ID from the pour output):

```
Run `bd --no-daemon ready --mol <epic-id>` to see the ready steps,
then walk me through each one. Reference: docs/obtaining-secrets.md
```

For the current Google OAuth molecule already poured, the epic ID is
`personal-knowledge-base-mol-6d8`.

Claude will:
1. Open each URL in your browser
2. Tell you exactly what to click and fill in
3. See your browser (via `--chrome`) if you get stuck
4. Ask you to paste credentials -- then write them to `.env` for you
5. Run the auth flow and verify everything works
6. Close each step bead as it completes

## 3. Close the molecule

Once all steps are done, Claude closes the epic. Run `bd sync` if not already synced.

## Pour examples for common providers

```bash
# OAuth: Google (client ID + client secret + consent screen)
bd --no-daemon mol pour mol-obtain-secret \
  --var provider=google \
  --var service="Google Drive + Gmail" \
  --var secret_type=oauth \
  --var console_url="https://console.cloud.google.com/" \
  --var apis="Google Drive API, Gmail API" \
  --var env_vars="PKB_GOOGLE_CLIENT_ID,PKB_GOOGLE_CLIENT_SECRET" \
  --var scopes="drive.readonly,gmail.readonly"

# Token: GitHub (single personal access token)
bd --no-daemon mol pour mol-obtain-secret \
  --var provider=github \
  --var service="GitHub" \
  --var secret_type=token \
  --var console_url="https://github.com/settings/tokens" \
  --var env_vars="PKB_GITHUB_TOKEN"

# OAuth: Slack (web app with client ID + secret)
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

## Formula variables reference

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

## Quick reference (for when you already know the steps)

```bash
cat .env.example                                    # see what vars are needed
cp .env.example .env                                # copy template
# fill in values, then:
source .env && export PKB_GOOGLE_CLIENT_ID PKB_GOOGLE_CLIENT_SECRET
make build && ./pkb auth                            # authenticate
./pkb search "test query"                           # verify
```

## Google OAuth manual steps (reference only)

These are the steps Claude walks you through. Listed here so you can do it without Claude if needed.

1. **Open Google Cloud Console** -- https://console.cloud.google.com/ -- sign in, create/select a project
2. **Enable APIs** -- https://console.cloud.google.com/apis/library -- enable Google Drive API and Gmail API
3. **Configure OAuth Consent Screen** -- https://console.cloud.google.com/apis/credentials/consent -- External user type, app name, emails, scopes (`drive.readonly`, `gmail.readonly`), add yourself as test user
4. **Create Credentials** -- https://console.cloud.google.com/apis/credentials -- Create Credentials > OAuth client ID > Desktop app, copy Client ID and Client Secret
5. **Configure locally** -- `cp .env.example .env`, paste credentials
6. **Authenticate and test** -- `source .env && export PKB_GOOGLE_CLIENT_ID PKB_GOOGLE_CLIENT_SECRET && make build && ./pkb auth && ./pkb search "test"`

Token is saved to `~/.config/pkb/token.json`.

## Troubleshooting

| Problem | Fix |
|---------|-----|
| "This app is blocked" during OAuth | Add your email as a **test user** in the consent screen |
| "credentials not configured" | `source .env && export PKB_GOOGLE_CLIENT_ID PKB_GOOGLE_CLIENT_SECRET` |
| Token expired | Re-run `./pkb auth` |
| Wrong account | Delete `~/.config/pkb/token.json` and re-run `./pkb auth` |
| Claude can't see browser | Make sure you started with `claude --chrome` |
