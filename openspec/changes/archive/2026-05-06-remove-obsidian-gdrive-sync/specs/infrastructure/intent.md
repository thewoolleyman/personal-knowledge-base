## REMOVED Requirements

### Requirement: Obsidian vault mirroring to Google Drive

The rsync-based launchd agent that mirrored the local Obsidian vault to Google Drive has been removed. This was a local Mac convenience setup, not a system requirement.

**Reason:** No longer needed; the setup was external to the application and undocumented at the spec level.

**Migration:** None required. The Obsidian connector for knowledge search is unaffected and continues to read the local vault directly.

#### Scenario: No vault mirror agent running

- **WHEN** the system is running on the local Mac
- **THEN** no launchd agent named `com.user.rsync-obsidian-to-gdrive` SHALL exist
