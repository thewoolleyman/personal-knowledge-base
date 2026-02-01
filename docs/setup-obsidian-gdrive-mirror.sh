#!/bin/bash
# Obsidian Vault → Google Drive Mirror Setup Script
# Run this on your Mac to set up the sync

set -e  # Exit on any error

# ============================================================================
# CONFIGURATION - Verify and edit these paths before running
# ============================================================================

# Source: Your Obsidian vault
SOURCE_DIR="/Users/cwoolley/-Obsidian-Default-Vault"

# Destination subdirectory (relative to Google Drive mount)
DEST_SUBDIR="Personal_Knowledge_Base_Mirrors/Obsidian_Default_Vault"

# ============================================================================
# AUTO-DETECT GOOGLE DRIVE PATH
# ============================================================================

echo "=== Obsidian → Google Drive Mirror Setup ==="
echo ""

# Find Google Drive mount point
# Modern Google Drive for Desktop uses ~/Library/CloudStorage/GoogleDrive-*/My Drive
# Old Backup and Sync used ~/Google Drive/My Drive

GDRIVE_BASE=""

# Check modern path first (Google Drive for Desktop)
MODERN_PATHS=$(ls -d ~/Library/CloudStorage/GoogleDrive-*/My\ Drive 2>/dev/null || true)
if [[ -n "$MODERN_PATHS" ]]; then
    # Take the first match
    GDRIVE_BASE=$(echo "$MODERN_PATHS" | head -n 1)
    echo "✓ Found Google Drive (modern path): $GDRIVE_BASE"
fi

# Fall back to legacy path
if [[ -z "$GDRIVE_BASE" ]]; then
    if [[ -d "$HOME/Google Drive/My Drive" ]]; then
        GDRIVE_BASE="$HOME/Google Drive/My Drive"
        echo "✓ Found Google Drive (legacy path): $GDRIVE_BASE"
    fi
fi

# Check if we found anything
if [[ -z "$GDRIVE_BASE" ]]; then
    echo "✗ ERROR: Could not find Google Drive mount!"
    echo ""
    echo "  Expected locations:"
    echo "    - ~/Library/CloudStorage/GoogleDrive-<email>/My Drive (modern)"
    echo "    - ~/Google Drive/My Drive (legacy)"
    echo ""
    echo "  Make sure Google Drive for Desktop is installed and running."
    exit 1
fi

DEST_DIR="$GDRIVE_BASE/$DEST_SUBDIR"

echo ""
echo "Configuration:"
echo "  Source:      $SOURCE_DIR"
echo "  Destination: $DEST_DIR"
echo ""

read -p "Does this look correct? (y/n) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted. Edit the CONFIGURATION section at the top of this script and re-run."
    exit 0
fi

echo ""

# ============================================================================
# PRE-FLIGHT CHECKS
# ============================================================================

echo "=== Pre-flight Checks ==="

# Check source exists
if [[ ! -d "$SOURCE_DIR" ]]; then
    echo "✗ ERROR: Source directory does not exist: $SOURCE_DIR"
    exit 1
fi
echo "✓ Source directory exists"

# Count files in source
SOURCE_COUNT=$(find "$SOURCE_DIR" -type f | wc -l | tr -d ' ')
echo "  → $SOURCE_COUNT files in source"

# Create destination if needed
if [[ ! -d "$DEST_DIR" ]]; then
    echo "→ Creating destination directory..."
    mkdir -p "$DEST_DIR"
    echo "✓ Created: $DEST_DIR"
else
    echo "✓ Destination directory exists"
fi

# Create log directory
LOG_DIR="$HOME/.local/log"
if [[ ! -d "$LOG_DIR" ]]; then
    mkdir -p "$LOG_DIR"
    echo "✓ Created log directory: $LOG_DIR"
else
    echo "✓ Log directory exists"
fi

# Create LaunchAgents directory if needed
LAUNCH_AGENTS_DIR="$HOME/Library/LaunchAgents"
if [[ ! -d "$LAUNCH_AGENTS_DIR" ]]; then
    mkdir -p "$LAUNCH_AGENTS_DIR"
    echo "✓ Created LaunchAgents directory"
else
    echo "✓ LaunchAgents directory exists"
fi

echo ""

# ============================================================================
# TEST RSYNC
# ============================================================================

echo "=== Testing rsync (dry-run) ==="

rsync -avn --delete "$SOURCE_DIR/" "$DEST_DIR/" 2>&1 | tail -5
echo "..."
echo ""

read -p "Run actual rsync now? (y/n) " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "→ Running rsync..."
    rsync -av --delete "$SOURCE_DIR/" "$DEST_DIR/"
    echo ""
    echo "✓ Initial sync complete"
else
    echo "→ Skipped initial sync"
fi

echo ""

# ============================================================================
# CREATE LAUNCHD PLIST
# ============================================================================

echo "=== Setting up launchd automation ==="

PLIST_PATH="$HOME/Library/LaunchAgents/com.user.rsync-obsidian-to-gdrive.plist"
LOG_PATH="$HOME/.local/log/rsync-obsidian.log"

# Unload existing if present (suppress errors)
launchctl bootout gui/$(id -u) "$PLIST_PATH" 2>/dev/null || true

cat > "$PLIST_PATH" << PLISTEOF
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
        <string>${SOURCE_DIR}/</string>
        <string>${DEST_DIR}/</string>
    </array>

    <key>StartInterval</key>
    <integer>600</integer>

    <key>RunAtLoad</key>
    <true/>

    <key>StandardOutPath</key>
    <string>${LOG_PATH}</string>

    <key>StandardErrorPath</key>
    <string>${LOG_PATH}</string>
</dict>
</plist>
PLISTEOF

echo "✓ Created plist: $PLIST_PATH"

# Load the agent
launchctl bootstrap gui/$(id -u) "$PLIST_PATH"
echo "✓ Loaded launch agent"

# Verify it's running
sleep 1
if launchctl print gui/$(id -u)/com.user.rsync-obsidian-to-gdrive &>/dev/null; then
    echo "✓ Launch agent is active"
else
    echo "⚠ Launch agent may not be running. Check with:"
    echo "  launchctl print gui/\$(id -u)/com.user.rsync-obsidian-to-gdrive"
fi

echo ""

# ============================================================================
# SUMMARY
# ============================================================================

echo "=== Setup Complete ==="
echo ""
echo "Your Obsidian vault will now sync to Google Drive every 10 minutes."
echo ""
echo "Useful commands:"
echo ""
echo "  # Check status"
echo "  launchctl print gui/\$(id -u)/com.user.rsync-obsidian-to-gdrive"
echo ""
echo "  # Trigger sync now"
echo "  launchctl kickstart gui/\$(id -u)/com.user.rsync-obsidian-to-gdrive"
echo ""
echo "  # View logs"
echo "  tail -f ~/.local/log/rsync-obsidian.log"
echo ""
echo "  # Stop/disable"
echo "  launchctl bootout gui/\$(id -u) ~/Library/LaunchAgents/com.user.rsync-obsidian-to-gdrive.plist"
echo ""
echo "  # Re-enable"
echo "  launchctl bootstrap gui/\$(id -u) ~/Library/LaunchAgents/com.user.rsync-obsidian-to-gdrive.plist"
echo ""
