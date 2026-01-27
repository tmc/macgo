# Window State Handling in list-app-windows

This document explains how `list-app-windows` handles different window states, including minimized windows, hidden windows, and windows on different Spaces.

## Window States in macOS

macOS windows can be in several states:

1. **Visible (On-Screen)** - Windows currently visible on the active Space/desktop
2. **Minimized** - Windows minimized to the Dock
3. **Hidden** - Windows hidden via Cmd+H or application hide
4. **On Different Space** - Windows on other Mission Control Spaces/desktops
5. **System UI** - Special windows like Control Center, Dock, Window Server overlays

## Default Behavior

By default, `list-app-windows` only shows **visible windows** on the current Space:

```bash
list-app-windows
```

This uses the Core Graphics flag `kCGWindowListOptionOnScreenOnly` which filters to windows that are:
- Not minimized
- Not hidden
- On the current active Space
- Excluding desktop elements

## Including Offscreen Windows

To include minimized, hidden, and windows on other Spaces, use the `-include-offscreen` flag:

```bash
list-app-windows -include-offscreen
```

This removes the `kCGWindowListOptionOnScreenOnly` restriction, showing all windows including:
- ✅ Minimized windows
- ✅ Hidden windows (Cmd+H)
- ✅ Windows on other Spaces
- ✅ Background application windows
- ❌ Still excludes system UI by default

## Including System UI Windows

To also include system UI windows, combine `-include-offscreen` with `-all`:

```bash
list-app-windows -all -include-offscreen
```

This shows absolutely everything, including:
- ✅ All window states (visible, minimized, hidden, different Spaces)
- ✅ System UI (Control Center, Window Server, Dock overlays, etc.)

## Practical Examples

### Finding a Minimized Window

```bash
# Minimize a Safari window, then:
list-app-windows -app Safari                    # Won't show minimized window
list-app-windows -app Safari -include-offscreen # Will show minimized window
```

### Capturing a Minimized Window

```bash
# Get window ID of minimized window
WINDOW_ID=$(list-app-windows -app Terminal -include-offscreen | awk 'NR==2 {print $1}')

# Capture it (even though minimized)
screen-capture -window $WINDOW_ID -output minimized.png
```

### Finding Windows on Different Spaces

```bash
# Move a window to a different Space, then:
list-app-windows -app "Google Chrome"                    # Won't show
list-app-windows -app "Google Chrome" -include-offscreen # Will show
```

### Debugging Window Issues

```bash
# See EVERYTHING (useful for debugging)
list-app-windows -all -include-offscreen -json > all-windows.json
```

## Technical Details

### Core Graphics Window List Options

The tool uses `CGWindowListCopyWindowInfo` with these options:

**Default (visible only):**
```c
kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements
```

**With `-include-offscreen`:**
```c
kCGWindowListExcludeDesktopElements
```

### Window State Detection Limitations

macOS Core Graphics does **not** provide explicit window state flags. We cannot directly determine:
- Whether a window is minimized vs hidden vs on another Space
- Whether a window is fullscreen
- Whether a window is behind others (Z-order beyond basic ordering)

The only reliable detection is:
- **On-screen vs Off-screen** - Via `kCGWindowListOptionOnScreenOnly`
- **System UI filtering** - Via heuristics (app name, window size)

### Capturing Off-Screen Windows

The `screencapture -l <window_id>` command can successfully capture:
- ✅ Minimized windows (renders last visible state)
- ✅ Windows on other Spaces (renders actual content)
- ⚠️ Hidden windows (may capture last state or fail)
- ⚠️ Windows that have never been shown (may be blank)

## Troubleshooting

### "Application has no windows" but I see it

The window might be:
1. Minimized - Use `-include-offscreen`
2. On another Space - Use `-include-offscreen`
3. Hidden - Use `-include-offscreen`
4. A system UI element - Use `-all -include-offscreen`

### Window ID doesn't work with screencapture

1. Verify the window still exists:
   ```bash
   list-app-windows -include-offscreen | grep <window_id>
   ```

2. Check if it's a special window type that screencapture can't capture

3. Some windows change IDs when moved between Spaces

### Performance Considerations

Listing all windows including off-screen:
- Slightly slower (more windows to enumerate)
- Returns significantly more results
- May include thousands of windows with `-all -include-offscreen`

Filtering recommendations:
- Always use `-app` when you know the target
- Avoid `-all` unless debugging
- Use default (on-screen only) when possible

## See Also

- [Core Graphics Window Services](https://developer.apple.com/documentation/coregraphics/quartz_window_services)
- [screencapture man page](x-man-page://screencapture)
- `screen-capture` tool in this repository

## Commit History

- Added `-include-offscreen` flag for macgo-28 (Handle minimized and hidden windows)
- Original implementation used on-screen only filtering
