# Protocol Interfaces — Intent

How consumers (humans and agents) talk to the system.

## Purpose

The same knowledge retrieval capabilities are exposed through multiple interfaces. Each interface is a different transport for the same underlying functionality. The system is designed to be consumed agentically — via MCP server, Agent Client Protocol (ACP), CLI, or HTTP API — as well as by humans via web UI, TUI, or CLI.

## Current Interfaces

- **HTTP API** — REST endpoint at `/search` with query and source parameters. The foundation that other interfaces build on.
- **CLI** — `pkb search <query>` command with source flags. Refactored to dogfood the HTTP API via an internal API client.
- **TUI** — Interactive terminal UI. Also dogfoods the HTTP API.
- **Web UI** — Embedded web interface served by `pkb serve`. Plain HTML/CSS/JavaScript with search and source selector. Collapsible source sections (planned).

## Planned Interfaces

- **MCP Server** — Model Context Protocol for agentic consumption. The primary way AI agents will access the knowledge base.
- **ACP** — Agent Client Protocol support for broader agent ecosystem compatibility.

## Domain Model

- **Interface** — a transport layer exposing retrieval capabilities (HTTP, CLI, MCP, etc.)
- **API client** — internal HTTP client (`internal/apiclient`) used by CLI and TUI to dogfood the API
- **Server** — the HTTP server (`internal/server`) that serves the API and web UI

## Behavioral Narratives

### API dogfooding
The CLI and TUI do not call connectors directly. They use the internal API client to hit the HTTP server, which in turn queries connectors. This ensures the API is always tested by real usage.

### Agent consumption
An agent (Open Brain, second brain, OpenClaw) connects via MCP or ACP, queries the knowledge base, and receives structured results suitable for context assembly. The agent treats PKB as a knowledge source.
