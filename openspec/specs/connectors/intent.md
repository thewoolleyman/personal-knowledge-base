# Connectors — Intent

Individual source adapters for knowledge repositories.

## Purpose

Connectors are the extensibility story. Each connector knows how to authenticate with and query a specific knowledge source. They implement a common interface so the retrieval system can fan out queries without knowing source-specific details.

## Current Connectors

- **Obsidian** — Searches a local Obsidian vault by reading markdown files. Configured via vault path in `.env`. Supports subfolder search.
- **Google Drive** — Searches Google Drive documents via the Drive API. Requires OAuth credentials. Returns document titles, snippets, and links.
- **Gmail** — Searches Gmail messages via the Gmail API. Requires OAuth credentials (shared with Google Drive). Returns message subjects, snippets, and links.
- **Notion** — Searches Notion pages/databases via the Notion API. Requires an integration token.

## Domain Model

- **Connector interface** — the common contract all connectors implement (search method accepting query string, returning unified results)
- **Credentials** — source-specific authentication (OAuth tokens, API keys, vault paths) stored in `.env`
- **Unified result** — title, source name, snippet, URL

## Behavioral Narratives

### Building a new connector
A developer creates a new package under `internal/connectors/` implementing the connector interface. They add credentials to `.env.example`, write unit tests with mocked API responses, write an integration test that hits the real API, and add documentation to the README.

### Cross-platform considerations
The Obsidian connector reads local files and works on any OS. The `authenticate` CLI subcommand must work on Linux without a GUI (can't use macOS `open` tool). OAuth token paths follow XDG config directory conventions.
