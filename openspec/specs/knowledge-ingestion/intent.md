# Knowledge Ingestion — Intent

How knowledge enters the system from external sources.

## Purpose

Knowledge ingestion is the pipeline that connects external knowledge repositories to the retrieval system. Each source has its own connector that knows how to authenticate, query, and return results in the unified format.

## Current Capabilities

- Connectors for Obsidian (local vault), Google Drive, Gmail, and Notion
- Each connector implements a common interface for search
- Connectors are configured via environment variables
- Live API integration — connectors hit real APIs, not cached/indexed copies

## Domain Model

- **Connector** — an adapter that speaks a source's API and returns unified results
- **Source configuration** — credentials and settings for a specific knowledge source (stored in `.env`)
- **Unified result** — the common format all connectors produce (title, source, snippet, URL)

## Behavioral Narratives

### Adding a new knowledge source
A developer implements a new connector satisfying the connector interface. They add configuration entries to `.env.example` and documentation to the README. The new source appears automatically in search results.

### Syncing and freshness
Currently, all searches are live — each query hits the source API in real-time. There is no local index or cache. This means results are always fresh but latency depends on source API speed.
