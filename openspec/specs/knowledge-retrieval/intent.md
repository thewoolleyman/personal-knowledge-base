# Knowledge Retrieval — Intent

The core value of the system: given a query, return relevant knowledge from all connected sources.

## Purpose

Knowledge retrieval is the primary interface for both human users and agent consumers. It searches across all connected knowledge sources (Obsidian vaults, Google Drive, Gmail, Notion) and returns unified, ranked results.

This is what agents primarily consume — whether via MCP, ACP, CLI, or HTTP API.

## Current Capabilities

- Full-text search across multiple knowledge sources simultaneously
- Results include title, source, URL, and 80-character text snippets
- Source filtering — query specific sources or search all
- Available via HTTP API (`/search` endpoint), CLI (`pkb search`), TUI, and web UI

## Domain Model

- **Query** — a search string submitted by a user or agent
- **Result** — a matched document with title, source, snippet, and URL
- **Source** — a connected knowledge repository (Obsidian, Google Drive, Gmail, Notion)
- **Context assembly** — the process of gathering and ranking results for agent consumption (future)

## Behavioral Narratives

### Searching knowledge
A user or agent submits a query. The system fans out to all configured connectors, collects results, and returns them ranked by relevance. Results include enough context (snippet, source, URL) to decide whether to open the full document.

### Filtering by source
A user or agent may restrict search to specific sources (e.g., only Obsidian, only Gmail). The system queries only the requested connectors and returns filtered results.
