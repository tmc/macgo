# TCC Helper

A tool to help manage macOS TCC (Transparency, Consent, and Control) permissions through System Settings automation.

## Overview

This tool **does not bypass TCC** - it helps automate the UI navigation to make granting permissions easier during testing and development. It can:

- Open System Settings to specific permission panes
- Provide guided instructions for granting permissions
- Automate UI interactions to click buttons (requires Accessibility permission)

## Usage

### List Available Services

```bash
./tcc-helper -list
```

Shows all supported TCC services (screen-recording, accessibility, camera, microphone, etc.)

### Open System Settings

```bash
./tcc-helper -service screen-recording -action open
```

Opens System Settings to the specified permission pane with instructions.

### Interactive Prompt

```bash
./tcc-helper -service accessibility -action prompt
```

Prompts for confirmation before opening System Settings, then waits for you to grant permission.

### Revoke Permission (UI Automation)

```bash
./tcc-helper -service screen-recording -action revoke -app "ScreenCaptureKit-Example"
```

Automatically revokes a permission from an app by:
1. Opening System Settings to the correct pane
2. Finding the app in the permission list
3. Selecting it and clicking the '-' button to remove it

This is useful for testing and cleanup. **Requires UI automation setup** (see below).

### Grant Permission (UI Automation)

```bash
# Grant by app name (searches common locations)
./tcc-helper -service screen-recording -action automate -app screen-capture

# Grant by full path
./tcc-helper -service screen-recording -action automate -path /Applications/MyApp.app
```

Automatically grants a permission to an app by:
1. Opening System Settings to the correct pane
2. Clicking the lock icon (if locked) - you'll need to authenticate
3. Clicking the '+' button to open the file picker
4. Navigating to the app using Cmd+Shift+G
5. Selecting and opening the app to grant permission

**When using -app**: The tool searches common locations (/Applications, /System/Applications, etc.) and the current directory.

**When using -path**: Provide the full path to the .app bundle.

**Note**: You may need to authenticate when the lock is clicked. The automation will pause for you to enter your password.
**Important**: UI automation requires special setup (see below).

### Test Accessibility Permission

```bash
./tcc-helper -action test-access
```

Tests whether the tool has the necessary Accessibility permission for UI automation.

### Inspect UI Structure (Debugging)

```bash
./tcc-helper -service screen-recording -action inspect-ui
```

Inspects the System Settings UI structure to help diagnose UI automation issues. This is useful when:
- UI automation is failing with "Could not find app list table" errors
- You need to understand how the UI is structured on your macOS version
- You're debugging path detection issues

The inspect-ui action provides detailed information about the UI hierarchy and tests multiple common UI paths.

## UI Automation Setup

UI automation requires **TWO** separate permissions to work:

1. **Accessibility Permission** - Allows the tool to control UI elements
2. **Automation Permission** - Allows AppleScript to send events to System Events

### Understanding the Permission Chain

When tcc-helper runs UI automation, the permission flow is:

```
Your Terminal (iTerm/Terminal.app)
  ↓ runs
osascript (AppleScript interpreter)
  ↓ sends Apple Events to
System Events (macOS system process)
  ↓ controls
System Settings UI
```

Both permissions must be granted for this chain to work.

### Step 1: Grant Accessibility Permission

The Accessibility permission allows osascript to actually manipulate UI elements.

#### Using tcc-helper to open settings:

```bash
./tcc-helper -service accessibility -action open
```

#### Manual steps:

1. System Settings > Privacy & Security > Accessibility
2. Click the lock icon (🔒) to unlock (requires authentication)
3. Click the '+' button
4. Press Cmd+Shift+G and type: `/usr/bin/osascript`
5. Click 'Open' to add osascript
6. **Ensure the checkbox next to osascript is checked**

**Important**: You must grant permission to `/usr/bin/osascript`, not to your terminal or the tcc-helper binary. This is because tcc-helper uses AppleScript (osascript) to perform UI automation.

### Step 2: Grant Automation Permission

The Automation permission allows your terminal to send Apple Events to System Events.

#### Automatic grant:

The first time you run a UI automation command (e.g., `./tcc-helper -service screen-recording -action automate -app screen-capture`), macOS will prompt you:

```
"iTerm" wants to control "System Events".
```

Click **OK** to grant this permission.

#### Manual verification:

1. System Settings > Privacy & Security > Automation
2. Look for your terminal (e.g., "iTerm" or "Terminal")
3. Expand the entry
4. **Ensure "System Events" is checked**

### Testing Your Setup

Run this command to verify both permissions are correctly configured:

```bash
./tcc-helper -action test-access
```

Expected output if properly configured:
```
✓ tcc-helper HAS Accessibility permission
  UI automation should work
```

If you see an error, follow the instructions provided to grant the missing permission.

### Quick Setup Checklist

- [ ] Grant Accessibility permission to `/usr/bin/osascript`
- [ ] Grant Automation permission for your terminal to control System Events
- [ ] Run `./tcc-helper -action test-access` to verify
- [ ] Try a UI automation command like `./tcc-helper -service screen-recording -action revoke -app screen-capture`

### Why Both Permissions Are Needed

- **Accessibility**: Required for osascript to "see" and interact with UI elements (buttons, tables, etc.)
- **Automation**: Required for your terminal process to invoke osascript and send it commands to System Events

Without Accessibility, you'll get error `-25211` (not allowed assistive access).
Without Automation, you'll get error `-1743` (Not authorized to send Apple events).

### Troubleshooting Permission Issues

If `./tcc-helper -action test-access` reports missing permissions:

**Error -25211 (Accessibility missing)**:
- Grant Accessibility permission to `/usr/bin/osascript`
- After granting, you may need to quit and restart your terminal

**Error -1743 (Automation missing)**:
- Grant Automation permission for your terminal → System Events
- Run any UI automation command once to trigger the permission prompt
- Click "OK" when macOS asks if your terminal can control System Events

## How It Works

### Permission Detection

tcc-helper uses macgo to properly request TCC permissions by:
1. Creating a minimal .app bundle with proper Info.plist
2. Adding the required entitlements
3. Relaunching via LaunchServices so macOS recognizes it as an app

### UI Automation

UI automation uses AppleScript to interact with System Events:
```applescript
tell application "System Events"
    tell process "System Settings"
        -- Click buttons, interact with UI elements
    end tell
end tell
```

This requires the **app bundle** to have Accessibility permission, which is why setup is required.

## Limitations

- **No TCC Bypass**: This tool cannot grant permissions without user consent
- **Requires User Interaction**: Even with automation, unlocking settings requires authentication
- **macOS SIP**: System Integrity Protection prevents direct TCC database manipulation
- **Bundle Complexity**: The macgo bundle mechanism adds complexity to permission grants
- **Experimental**: UI automation may break with macOS UI changes

## Examples

```bash
# List all services
./tcc-helper -list

# Open screen recording settings
./tcc-helper -service screen-recording -action open

# Test if automation will work
./tcc-helper -action test-access

# Inspect System Settings UI structure (for debugging)
./tcc-helper -service screen-recording -action inspect-ui

# Revoke screen recording permission from an app
./tcc-helper -service screen-recording -action revoke -app "ScreenCaptureKit-Example"

# Grant screen recording permission to an app by name
./tcc-helper -service screen-recording -action automate -app "MyApp"

# Grant screen recording permission to an app by path
./tcc-helper -service screen-recording -action automate -path /Applications/MyApp.app
```

## Troubleshooting

### "osascript is not allowed assistive access"

This means the tcc-helper bundle doesn't have Accessibility permission yet. See "UI Automation Setup" above.

### Automation clicks wrong buttons

The UI automation uses button descriptions that may change between macOS versions. The script may need updates for your macOS version.

### Permission not granted after automation

Some permissions require additional steps beyond clicking '+'. The automation provides a starting point but you may need to complete the process manually.

## Building

tcc-helper uses automatic code signing to maintain a stable identity across rebuilds. This prevents the need to re-grant Accessibility permission every time you rebuild.

### Quick Build

```bash
# Using Make (recommended)
make

# Or using go build directly
go build -o tcc-helper
```

### Code Signing Behavior

tcc-helper automatically uses the best available signing method:

1. **Developer ID Application** (if available in keychain)
   - Provides the most stable identity
   - Permissions persist across rebuilds and even machine transfers
   - Requires a paid Apple Developer account

2. **Stable Ad-hoc Signature** (fallback)
   - Uses a fixed bundle identifier: `com.github.tmc.macgo.tcc-helper`
   - Permissions persist across rebuilds on the same machine
   - No developer account required
   - This is what most users will use during development

### Adding Developer ID Signing

If you have a Developer ID Application certificate:

```bash
# List available certificates
security find-identity -v -p codesigning

# The build will automatically find and use it
make

# Or specify explicitly
MACGO_CODE_SIGN_IDENTITY="Developer ID Application: Your Name (TEAMID)" make
```

### Makefile Targets

```bash
make              # Build with automatic signing
make dev          # Build with debug output
make clean        # Clean build artifacts
make install      # Install to ~/bin
make help         # Show detailed help
```

### Why This Matters

Without a stable code signature, macOS treats each rebuild as a "different app" and requires you to re-grant Accessibility permission every time. With stable signing:

- ✅ Permissions persist across rebuilds
- ✅ Faster development iteration
- ✅ No need to constantly re-authorize in System Settings

## Related Tools

- [macgo](https://github.com/tmc/macgo) - The library that enables proper TCC permission requests
- [screen-capture](../screen-capture) - Example tool that uses TCC screen recording permission
