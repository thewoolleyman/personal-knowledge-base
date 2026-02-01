# Obtaining and Configuring Secrets

## Quick Reference

```bash
# See what env vars are needed
cat .env.example

# Copy template and fill in values
cp .env.example .env

# Load into current shell
source .env && export PKB_GOOGLE_CLIENT_ID PKB_GOOGLE_CLIENT_SECRET

# Authenticate
make build && ./pkb auth

# Verify
./pkb search "test query"
```

## Google OAuth Credentials (Step by Step)

### 1. Open Google Cloud Console

```bash
open "https://console.cloud.google.com/"
```

Sign in. Create a new project or select existing.

### 2. Enable APIs

```bash
open "https://console.cloud.google.com/apis/library"
```

Search and enable:
- **Google Drive API**
- **Gmail API**

### 3. Configure OAuth Consent Screen

```bash
open "https://console.cloud.google.com/apis/credentials/consent"
```

- User type: **External**
- App name: anything (e.g., "pkb")
- User support email: your email
- Developer email: your email
- Scopes: add `drive.readonly` and `gmail.readonly`
- Test users: **add your own Google email** (required while in "Testing" status)

### 4. Create Credentials

```bash
open "https://console.cloud.google.com/apis/credentials"
```

- Click **Create Credentials > OAuth client ID**
- Application type: **Desktop app**
- Name: anything (e.g., "pkb-local")
- Click **Create**
- **Copy** the Client ID and Client Secret

### 5. Configure Locally

```bash
cp .env.example .env
# Edit .env — paste in your Client ID and Client Secret
```

Your `.env` should look like:
```
PKB_GOOGLE_CLIENT_ID="123456789-xxxxx.apps.googleusercontent.com"
PKB_GOOGLE_CLIENT_SECRET="GOCSPX-xxxxxxxxxxxxxxxx"
```

### 6. Authenticate and Test

```bash
source .env && export PKB_GOOGLE_CLIENT_ID PKB_GOOGLE_CLIENT_SECRET
make build
./pkb auth          # Opens browser — authorize the app
./pkb search "test" # Verify it works
```

Token is saved to `~/.config/pkb/token.json` (persists across sessions).

## Using the Beads Formula Template

This process is captured as a reusable beads formula. To use it for future secrets:

```bash
# Preview the template
bd cook mol-obtain-oauth-secret --dry-run

# Instantiate for Google (what we did above)
bd mol pour mol-obtain-oauth-secret \
  --var provider=google \
  --var service="Google Drive + Gmail" \
  --var console_url="https://console.cloud.google.com/" \
  --var apis="Google Drive API, Gmail API" \
  --var env_client_id=PKB_GOOGLE_CLIENT_ID \
  --var env_client_secret=PKB_GOOGLE_CLIENT_SECRET \
  --var scopes="drive.readonly,gmail.readonly" \
  --var app_type=desktop

# Instantiate for a future service (e.g., Slack)
bd mol pour mol-obtain-oauth-secret \
  --var provider=slack \
  --var service="Slack" \
  --var console_url="https://api.slack.com/apps" \
  --var apis="Web API" \
  --var env_client_id=PKB_SLACK_CLIENT_ID \
  --var env_client_secret=PKB_SLACK_CLIENT_SECRET \
  --var scopes="search:read" \
  --var app_type=web
```

## Troubleshooting

| Problem | Fix |
|---------|-----|
| "This app is blocked" during OAuth | Add your email as a **test user** in the consent screen |
| "credentials not configured" | Run `source .env && export PKB_GOOGLE_CLIENT_ID PKB_GOOGLE_CLIENT_SECRET` |
| Token expired | Re-run `./pkb auth` |
| Wrong account | Delete `~/.config/pkb/token.json` and re-run `./pkb auth` |
