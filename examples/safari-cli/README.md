# safari-cli

A macgo-enabled CLI tool for controlling Safari via AppleScript with proper permissions management.

## Features

- ✅ **Embedded sdef**: Safari.sdef bundled in binary
- ✅ **macgo integration**: Automatic Automation permissions handling
- ✅ **Self-cleanup**: Remove permissions when done
- ✅ **Global install**: Install to ~/bin for system-wide access
- ✅ **Lazy permissions**: Only requests access when executing commands

## Installation

```bash
make install
```

This installs `safari-cli` to `~/bin/safari-cli`. Make sure `~/bin` is in your PATH.

## Usage

### Open a URL
```bash
safari-cli open https://golang.org
```

### Get current tab info
```bash
safari-cli get-url
safari-cli get-title
```

### List all open tabs
```bash
safari-cli list-tabs
```

### Execute JavaScript
```bash
safari-cli js "document.title"
safari-cli js "alert('Hello from safari-cli')"
```

### List Safari's AppleScript API
```bash
safari-cli list-api
```

### Run tests
```bash
safari-cli test
```

### Clean up permissions
```bash
safari-cli cleanup-permissions
```

## How It Works

### 1. Embedded sdef
The Safari.sdef file is embedded at compile time using `//go:embed`:

```go
//go:embed Safari.sdef
var safariSdef []byte
```

This means the binary is completely self-contained.

### 2. Lazy Permission Requests
Permissions are only requested when you actually run a command:

```go
func ensurePermissions() {
    if permissionsRequested {
        return
    }
    // Request permissions via macgo
    macgo.Start(cfg)
}
```

This means `safari-cli --help` works instantly without any permission dialogs.

### 3. macgo Integration
Uses macgo to request Automation permissions for Safari:

```go
cfg := &macgo.Config{
    AppName: "safari-cli",
    Custom: []string{
        "com.apple.security.automation.apple-events",
    },
}
```

### 4. Self-Cleanup
Can remove its own permissions:

```bash
safari-cli cleanup-permissions
```

This runs `tccutil reset AppleEvents` to clear the TCC database entry.

## Example Session

```bash
$ safari-cli open https://example.com
Opening https://example.com...
✓ Opened

$ safari-cli get-url
https://example.com/

$ safari-cli get-title
Example Domain

$ safari-cli js "document.body.innerText"
Example Domain

This domain is for use in illustrative examples...

$ safari-cli list-tabs
1. Example Domain - https://example.com/
2. Google - https://www.google.com/
3. GitHub - https://github.com/

$ safari-cli cleanup-permissions
Cleaning up automation permissions...
✓ Permissions reset
```

## Architecture

```
Binary
  ├── main.go (CLI logic)
  ├── Safari.sdef (embedded)
  └── macgo (permissions)
       ↓
   osascript
       ↓
   Safari.app
```

## Commands Reference

| Command | Description |
|---------|-------------|
| `open [url]` | Open URL in new tab |
| `get-url` | Get current tab URL |
| `get-title` | Get current tab title |
| `list-tabs` | List all open tabs |
| `js [code]` | Execute JavaScript |
| `list-api` | Show Safari's API |
| `test` | Run test suite |
| `cleanup-permissions` | Remove permissions |

## Global Namespacing

The tool is designed to be installed globally:

```bash
make install  # → ~/bin/safari-cli
```

The binary name `safari-cli` is namespaced to avoid conflicts with:
- Safari's built-in commands
- Other automation tools
- System utilities

## Uninstall

```bash
make uninstall
safari-cli cleanup-permissions  # Before uninstalling
```

## Development

```bash
# Build
make build

# Test
make test

# Clean
make clean
```

## Related Tools

- **osascript-wrapper**: File-based AppleScript management
- **sdef-to-cobra**: Dynamic CLI generator from sdef
- **macgo**: macOS permissions framework

This tool combines all three approaches into a production-ready CLI.
