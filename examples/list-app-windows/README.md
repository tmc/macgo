# list-app-windows

List application windows with their CGWindowIDs for use with macOS `screencapture` command.

## Overview

This tool uses CGo to access Core Graphics APIs and retrieve actual window IDs (CGWindowID) that work with macOS `screencapture -l` command. Unlike AppleScript window IDs, these are the real Quartz window IDs used by the window server.

**Note:** This tool does NOT require macgo or any special permissions to list windows. The Core Graphics APIs for window enumeration are publicly accessible.

## Features

- **Real CGWindowIDs**: Gets actual Quartz window IDs, not AppleScript IDs
- **Works with screencapture**: Window IDs can be used directly with `screencapture -l <windowid>`
- **Filter by application**: Show windows only for specific apps
- **App name aliases**: Support for legacy app names (e.g., System Preferences → System Settings)
- **Window state handling**: Show visible windows only, or include minimized/hidden windows
- **Multiple output formats**: Table (default), grouped, or JSON
- **No external dependencies**: Pure CGo, no Swift or Python required

## Building

```bash
cd examples/list-app-windows
go build
```

## Usage

### List all windows

```bash
./list-app-windows
```

Output:
```
WINDOW_ID  OWNER           TITLE
1885       Control Center
69         iTerm
141        Brave Browser
```

### Filter by application

```bash
./list-app-windows -app Safari
./list-app-windows -app "Brave"
```

#### App Name Aliases

The tool supports app name aliases for better compatibility across macOS versions:

```bash
# Works on macOS Ventura+ (System Settings)
./list-app-windows -app "System Settings"

# Works on macOS Big Sur and earlier (System Preferences)
./list-app-windows -app "System Preferences"

# Both resolve to the correct app name for your macOS version
./list-app-windows -app "settings"
./list-app-windows -app "preferences"
```

**Supported aliases include:**
- `System Settings` ↔ `System Preferences` (bidirectional, with aliases: `settings`, `preferences`)
- `Music` ↔ `iTunes`
- `Calendar` ↔ `iCal`
- `Contacts` ↔ `Address Book`
- `TV` ↔ `Apple TV`
- `Books` ↔ `iBooks`
- Common lowercase variants (e.g., `safari` → `Safari`)

This improves UX when writing scripts that need to work across different macOS versions.

### Include minimized and hidden windows

By default, only visible windows on the current Space are shown. To include minimized windows, hidden windows, and windows on other Spaces:

```bash
./list-app-windows -include-offscreen
./list-app-windows -app Safari -include-offscreen
```

See [WINDOW_STATES.md](WINDOW_STATES.md) for detailed information about window state handling.

### Verbose output with bounds and layer

```bash
./list-app-windows -verbose
```

Output:
```
WINDOW_ID  OWNER_PID  OWNER  TITLE  LAYER  BOUNDS
1132       827        iTerm         0      0,51,2294x1349
```

### JSON output

```bash
./list-app-windows -json
```

Output:
```json
[
  {"window_id":1132,"owner_pid":827,"owner_name":"iTerm","window_name":"","layer":0,"x":0,"y":51,"width":2294,"height":1349}
]
```

## Integration with screen-capture

The recommended way to capture windows is using the `screen-capture` example from macgo, which handles permissions properly:

```bash
# Get the window ID
WINDOW_ID=$(./list-app-windows -app Safari | awk 'NR==2 {print $1}')

# Capture that window using screen-capture example
cd ../screen-capture
./screen-capture -window $WINDOW_ID -output safari-window.png
```

One-liner:

```bash
cd ../screen-capture && ./screen-capture -window $(cd ../list-app-windows && ./list-app-windows -app Safari | awk 'NR==2 {print $1}') -output screenshot.png
```

### Using screencapture directly

You can also use the macOS `screencapture` command directly:

```bash
WINDOW_ID=$(./list-app-windows -app Safari | awk 'NR==2 {print $1}')
screencapture -l $WINDOW_ID screenshot.png
```

Note: This requires Terminal/iTerm to have Screen Recording permission (see Permissions section).

## Permissions

### For list-app-windows

**No permissions required!** The tool uses public Core Graphics APIs (`CGWindowListCopyWindowInfo`) that don't require any special permissions to enumerate windows.

### For screencapture

The `screencapture` command requires **Screen Recording** permission:

1. Open **System Settings** > **Privacy & Security** > **Screen Recording**
2. Enable **Terminal** or **iTerm2** (whichever you're using)
3. Restart your terminal

Without this permission, `screencapture` will fail with:
```
could not create image from display
```

## Technical Details

### CGo Implementation

The tool uses CGo to call `CGWindowListCopyWindowInfo` from the Core Graphics framework:

```c
CFArrayRef windowList = CGWindowListCopyWindowInfo(
    kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
    kCGNullWindowID
);
```

### Window Information Retrieved

For each window:
- **kCGWindowNumber**: The actual CGWindowID (what screencapture needs)
- **kCGWindowOwnerPID**: Process ID of the owning application
- **kCGWindowOwnerName**: Application name
- **kCGWindowName**: Window title (may be empty)
- **kCGWindowLayer**: Window layer (0 = normal windows)
- **kCGWindowBounds**: Window position and size

### AppleScript vs CGWindowID

⚠️ **Important**: AppleScript window IDs are different from CGWindowIDs!

- AppleScript: `tell application "Safari" to get id of window 1` → returns app-specific ID (e.g., `93`)
- CGWindowID: From Core Graphics → returns Quartz window ID (e.g., `7493`)

Only CGWindowIDs work with `screencapture -l`.

## Examples

### Capture the frontmost window of an app

```bash
./list-app-windows -app "Brave Browser" | awk 'NR==2 {print $1}' | xargs -I {} screencapture -l {} brave-window.png
```

### Get window info as JSON for scripting

```bash
./list-app-windows -json | jq '.[] | select(.owner_name == "iTerm")'
```

### List all iTerm windows with their IDs

```bash
./list-app-windows -app iterm -verbose
```

## Troubleshooting

### "could not create image from window"

This means `screencapture` doesn't have Screen Recording permission. Grant it in System Settings.

### No windows shown for an app

The app might not have any visible windows, or the windows might be:
- Minimized to the Dock
- Hidden (Cmd+H)
- On a different Space/desktop

Try including offscreen windows:
```bash
./list-app-windows -app Safari -include-offscreen
```

See [WINDOW_STATES.md](WINDOW_STATES.md) for more details.

### Build errors

Make sure you have Xcode Command Line Tools installed:
```bash
xcode-select --install
```

## See Also

- [WINDOW_STATES.md](WINDOW_STATES.md) - Detailed guide on window state handling
- [screen-capture example](../screen-capture) - Full screen capture tool using macgo
- [Core Graphics documentation](https://developer.apple.com/documentation/coregraphics)
- [CGWindowListCopyWindowInfo](https://developer.apple.com/documentation/coregraphics/1455137-cgwindowlistcopywindowinfo)
