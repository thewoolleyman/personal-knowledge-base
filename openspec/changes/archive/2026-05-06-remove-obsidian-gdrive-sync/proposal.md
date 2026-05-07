## Why

The Obsidian-to-Google-Drive rsync mirroring setup was a local Mac infrastructure convenience that is no longer needed. Removing it eliminates unnecessary background activity and dead documentation.

## What Changes

- Removed `~/Library/LaunchAgents/com.user.rsync-obsidian-to-gdrive.plist` (launchd agent unloaded and deleted)
- Removed `docs/obsidian-vault-mirroring-notes.md`
- Removed `docs/setup-obsidian-gdrive-mirror.sh`

All removals have already been executed. No code changes are required.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. The vault mirroring was never specced — it existed only as docs-level setup instructions outside the spec system.

## Impact

- `docs/` — two files deleted
- Local macOS launchd configuration — agent unloaded and plist deleted
- No application code, APIs, tests, or specs are affected
