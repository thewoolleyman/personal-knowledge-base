## Why

Search results currently display as a flat list with no visual grouping. Users need to quickly scan results by source (Obsidian, Google Drive, Gmail, etc.). Grouping by source with collapsible sections makes the output scannable and organized.

## What Changes

- Group search output by source with section headers
- On the web UI: sections are expandable/collapsible, expanded by default
- Use plain JavaScript only — no external JS libraries
- CLI/TUI output should also group by source with headers

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `knowledge-retrieval`: Search results grouped by source with section headers
- `protocol-interfaces`: Web UI adds collapsible sections; CLI/TUI adds source grouping

## Impact

- Web UI templates/JavaScript
- CLI output formatting
- TUI rendering
- Search result data structure may need source grouping
