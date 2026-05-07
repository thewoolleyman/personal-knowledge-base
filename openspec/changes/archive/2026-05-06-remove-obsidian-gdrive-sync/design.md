## Context

A macOS launchd agent (`com.user.rsync-obsidian-to-gdrive`) was running every 10 minutes to rsync `/Users/cwoolley/-Obsidian-Default-Vault/` into the Google Drive local mount. This was a local convenience setup, documented in two files under `docs/`. It was never part of the application itself and was not specced.

All removals have already been executed prior to this change being created.

## Goals / Non-Goals

**Goals:**
- Record that the vault mirroring setup has been permanently removed
- Confirm no application code, tests, or specs require updating

**Non-Goals:**
- Replacing the sync with any alternative mechanism
- Modifying the Obsidian connector (which reads the local vault for search — unrelated to mirroring)

## Decisions

No technical decisions required. The change is a pure deletion of external setup files.

## Risks / Trade-offs

- [Data availability] The Obsidian vault is no longer mirrored to Google Drive → Accepted; the sync is intentionally removed and not replaced.
