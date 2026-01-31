# Obsidian Vault → Google Drive One-Way Sync Setup

## Why do this?

Obsidian Sync is good for syncing between multiple devices, but if you put the vault on Google Drive, it seems to get confused with Google Drive's syncing and will create thousands of duplicate files. 

But I still want to get my vault somewhere that it is indexed and searchable via API, and Google Drive seems to be the best choice for that. 

I use a simple rsync scheduled job which does a unidirectional sync from the real Obsidian Vault to a directory on my Google Drive. 

## Summary

Automatically mirrors your Obsidian vault to Google Drive every 10 minutes using `rsync` and `launchd`.

## Paths

| Description | Path |
|-------------|------|
| Source (Obsidian vault) | `/Users/cwoolley/-Obsidian-Default-Vault/` |
| Destination (Google Drive) | `/Users/cwoolley/Google Drive/My Drive/Personal_Knowledge_Base_Mirrors/Obsidian_Default_Vault/` |

---

## 1. Create the plist file

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
        <string>/Users/cwoolley/Google Drive/My Drive/Personal_Knowledge_Base_Mirrors/Obsidian_Default_Vault/</string>
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

## 2. Load the launch agent

```bash
launchctl load ~/Library/LaunchAgents/com.user.rsync-obsidian-to-gdrive.plist
```

---

## 3. Verify it is enabled and running

```bash
launchctl list | grep rsync-obsidian
```

**Expected output:**
```
-       0       com.user.rsync-obsidian-to-gdrive
```

- First column: PID (or `-` if not currently running)
- Second column: last exit status (`0` = success)
- Third column: label

---

## 4. Manually trigger a sync (optional)

```bash
launchctl start com.user.rsync-obsidian-to-gdrive
```

---

## 5. Check logs

```bash
tail -f ~/.local/log/rsync-obsidian.log
```

---

## Updating the plist

If you need to modify the plist:

```bash
# Unload first
launchctl unload ~/Library/LaunchAgents/com.user.rsync-obsidian-to-gdrive.plist

# Edit the file
nano ~/Library/LaunchAgents/com.user.rsync-obsidian-to-gdrive.plist

# Reload
launchctl load ~/Library/LaunchAgents/com.user.rsync-obsidian-to-gdrive.plist
```

---

## Disabling the sync

```bash
launchctl unload ~/Library/LaunchAgents/com.user.rsync-obsidian-to-gdrive.plist
```

To re-enable, run the load command again.

---

## Notes

- Sync interval: **10 minutes** (600 seconds)
- `--delete` flag ensures destination mirrors source exactly (files deleted locally will be deleted in Google Drive)
- First sync may take longer depending on vault size; subsequent syncs are incremental
- Google Drive desktop app handles uploading the local mirror to the cloud
