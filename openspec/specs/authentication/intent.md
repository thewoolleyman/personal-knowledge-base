# Authentication — Intent

Identity and access control for humans and agents.

## Purpose

Authentication manages how users and agents prove their identity and gain access to knowledge sources. Currently focused on Google OAuth for Drive/Gmail access and API tokens for Notion. As the system becomes more agentic, authentication will need to handle agent identity (MCP connections, API keys) alongside human OAuth flows.

## Current Capabilities

- Interactive OAuth token generation flow (`pkb authenticate`)
- Google OAuth for Drive and Gmail (shared credentials)
- Notion API token via environment variable
- Token storage at XDG config directory path
- Token refresh handling for expired Google OAuth tokens
- Works on Linux without GUI (no dependency on macOS `open`)

## Domain Model

- **OAuth flow** — the interactive browser-based authentication for Google APIs
- **Token** — stored credential (OAuth refresh token for Google, API key for Notion)
- **Token storage** — file-based storage at XDG-compliant path (`token.json`)
- **Credentials** — OAuth client ID/secret and API tokens configured in `.env`

## Behavioral Narratives

### First-time setup
A user runs `pkb authenticate`, which opens a browser for Google OAuth consent. The resulting token is stored locally. Notion requires manually adding an API token to `.env`. The `.env.example` file documents all required credential entries.

### Token expiration
When a Google OAuth token expires, the system uses the refresh token to obtain a new access token automatically. If the refresh token is also invalid, the user must re-authenticate.

### Agent authentication (future)
When agents connect via MCP or ACP, they will need their own identity mechanism. The specific approach (API keys, capability scopes, etc.) is yet to be determined and will evolve iteratively.
