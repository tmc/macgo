# press-keys Example

This example demonstrates how to use `macgo` to build a macOS application that simulates keyboard events using Core Graphics and Accessibility APIs.

## Features

- **TCC Permission Handling**: Automatically requests Accessibility permissions required for synthetic input.
- **Window Focusing**: Locates and focuses applications or windows by name.
- **Event Simulation**: Sends keystrokes with modifier support (Command, Shift, Option, Control).

## Usage

Build and run the tool:

```bash
go build
./press-keys help
```

### Commands

1.  **send-text**: Type a string of text into the focused window.
    ```bash
    ./press-keys send-text "Hello World"
    ./press-keys send-text -window "TextEdit" "Automated input"
    ```

2.  **send-key**: Send a single keystroke with optional modifiers.
    ```bash
    ./press-keys send-key enter
    ./press-keys send-key -cmd s          # Save
    ./press-keys send-key -cmd -shift z   # Redo
    ```

## Mechanics

This example uses:
- `macgo.Config` with `.WithCustom("com.apple.security.temporary-exception.apple-events")` (note: actual accessibility permission is system-wide, but this entitlement is often relevant for automation).
- CGO to interface with `CGEventCreateKeyboardEvent`, `CGEventPost`, and `NSApplication` APIs for window management.

> **Note**: On the first run, macOS will prompt you to grant Accessibility permissions to the `press-keys` application in System Settings.
