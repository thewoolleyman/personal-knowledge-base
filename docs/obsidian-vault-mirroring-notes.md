# Obsidian Vault → Google Drive One-Way Sync Setup

## Why do this?

Obsidian Sync is good for syncing between multiple devices, but if you put the vault on Google Drive, it seems to get confused with Google Drive's syncing and will create thousands of duplicate files.

But I still want to get my vault somewhere that it is indexed and searchable via API, and Google Drive seems to be the best choice for that.

I use a simple rsync scheduled job which does a unidirectional sync from the real Obsidian Vault to a directory on my Google Drive.

### Why Google Drive?

- Native full-text search via Google Drive API (index-based, fast)
- Quota-based, not pay-per-call
- Already have it set up with local mount via Google Drive for Desktop

### Why rsync (not Arq, rclone, etc.)?

- **Arq won't work** — it stores files in deduplicated/chunked blob format, not readable files. The destination would be useless for API search.
- **rclone is unnecessary** — Google Drive is already mounted locally, so rsync to the local mount is simpler (no OAuth config, no extra tooling).
- Google Drive desktop app handles uploading the local mirror to the cloud.

## Summary

Automatically mirrors your Obsidian vault to Google Drive every 10 minutes using `rsync` and `launchd`.

## Paths

| Description | Path |
|-------------|------|
| Source (Obsidian vault) | `/Users/cwoolley/-Obsidian-Default-Vault/` |
| Destination (Google Drive) | `/Users/cwoolley/Library/CloudStorage/GoogleDrive-thewoolleyman@gmail.com/My Drive/Personal_Knowledge_Base_Mirrors/Obsidian_Default_Vault/` |

**Note:** Google Drive for Desktop uses the modern `~/Library/CloudStorage/GoogleDrive-<email>/My Drive` mount path. The old `~/Google Drive/My Drive` path is from the legacy Backup and Sync app.

---

## Automated Setup

Run the setup script which auto-detects the Google Drive path, tests rsync, and installs the launch agent:

```bash
bash docs/setup-obsidian-gdrive-mirror.sh
```

The script handles everything below automatically. The manual steps are documented here for reference.

---

## Manual Setup

### 1. Create the plist file

```bash
mkdir -p ~/.local/log

cat << 'EOF' > ~/Library/LaunchAgents/com.user.rsync-obsidian-to-gdrive.plist
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.user.rsync-obsidian-to-gdrive</string>

    <key>ProgramArguments</key>
    <array>
        <string>/usr/bin/rsync</string>
        <string>-av</string>
        <string>--delete</string>
        <string>/Users/cwoolley/-Obsidian-Default-Vault/</string>
        <string>/Users/cwoolley/Library/CloudStorage/GoogleDrive-thewoolleyman@gmail.com/My Drive/Personal_Knowledge_Base_Mirrors/Obsidian_Default_Vault/</string>
    </array>

    <key>StartInterval</key>
    <integer>600</integer>

    <key>RunAtLoad</key>
    <true/>

    <key>StandardOutPath</key>
    <string>/Users/cwoolley/.local/log/rsync-obsidian.log</string>

    <key>StandardErrorPath</key>
    <string>/Users/cwoolley/.local/log/rsync-obsidian.log</string>
</dict>
</plist>
EOF
```

---

### 2. Load the launch agent

```bash
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.user.rsync-obsidian-to-gdrive.plist
```

---

### 3. Verify it is enabled and running

```bash
launchctl print gui/$(id -u)/com.user.rsync-obsidian-to-gdrive
```

---

### 4. Manually trigger a sync (optional)

```bash
launchctl kickstart gui/$(id -u)/com.user.rsync-obsidian-to-gdrive
```

---

### 5. Check logs

```bash
tail -f ~/.local/log/rsync-obsidian.log
```

---

## Updating the plist

If you need to modify the plist:

```bash
# Unload first
launchctl bootout gui/$(id -u) ~/Library/LaunchAgents/com.user.rsync-obsidian-to-gdrive.plist

# Edit the file
nano ~/Library/LaunchAgents/com.user.rsync-obsidian-to-gdrive.plist

# Reload
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.user.rsync-obsidian-to-gdrive.plist
```

---

## Disabling the sync

```bash
launchctl bootout gui/$(id -u) ~/Library/LaunchAgents/com.user.rsync-obsidian-to-gdrive.plist
```

To re-enable:

```bash
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.user.rsync-obsidian-to-gdrive.plist
```

---

## Notes

- Sync interval: **10 minutes** (600 seconds)
- `--delete` flag ensures destination mirrors source exactly (files deleted locally will be deleted in Google Drive)
- First sync may take longer depending on vault size; subsequent syncs are incremental
- Google Drive desktop app handles uploading the local mirror to the cloud
- Vault currently has ~277 files
- Ensure Google Drive desktop app is set to sync (not just stream) the target folder
