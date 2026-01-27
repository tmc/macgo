# safari-cli Demo

## What We Built

A production-ready Safari automation CLI with:
- ✅ Embedded Safari.sdef (8.7KB)
- ✅ macgo permissions integration
- ✅ Lazy permission requests
- ✅ Global install support
- ✅ Self-cleanup capabilities
- ✅ Full Cobra CLI with help

## Installation

```bash
$ make install
go build -o safari-cli
mkdir -p /Users/tmc/bin
cp safari-cli /Users/tmc/bin/
✓ Installed to /Users/tmc/bin/safari-cli
Make sure /Users/tmc/bin is in your PATH
```

## Usage Demo

### 1. Help Works Instantly (No Permissions Needed)

```bash
$ safari-cli --help
Control Safari via AppleScript with proper permissions management

Usage:
  safari-cli [command]

Available Commands:
  cleanup-permissions Remove Safari automation permissions for this tool
  get-title           Get the title of the front tab
  get-url             Get the URL of the front tab
  js                  Execute JavaScript in the front tab
  list-api            List Safari's AppleScript API
  list-tabs           List all open tabs
  open                Open a URL in Safari
  test                Run test commands to verify functionality
```

### 2. List API (No Permissions Needed)

```bash
$ safari-cli list-api
Safari suite:
  add reading list item          Add a new Reading List item
  do JavaScript                  Applies JavaScript code to a document
  email contents                 Emails the contents of a tab
  search the web                 Searches the web
  show bookmarks                 Shows Safari's bookmarks
```

### 3. First Command Requests Permissions

```bash
$ safari-cli open https://golang.org
# → macgo requests Safari Automation permission
# → User approves in System Settings
Opening https://golang.org...
✓ Opened
```

### 4. Subsequent Commands Work Seamlessly

```bash
$ safari-cli get-url
https://golang.org/

$ safari-cli get-title
Go Programming Language

$ safari-cli js "document.title"
Go Programming Language

$ safari-cli list-tabs
1. Go Programming Language - https://golang.org/
2. Example Domain - https://example.com/
```

### 5. Cleanup When Done

```bash
$ safari-cli cleanup-permissions
Cleaning up automation permissions...
✓ Permissions reset
```

## Key Features Demonstrated

### Embedded sdef
- No external dependencies
- 8.7KB Safari.sdef compiled into binary
- Parsed at runtime for API discovery

### macgo Integration
```go
cfg := &macgo.Config{
    AppName: "safari-cli",
    Custom: []string{
        "com.apple.security.automation.apple-events",
    },
}
macgo.Start(cfg)
```

### Lazy Permissions
```go
func execScript(script string) (string, error) {
    ensurePermissions() // Only called when executing
    // ...
}
```

- `--help` works instantly
- `list-api` works without permissions
- Permissions only requested on first actual Safari command

### Global Install
```bash
~/bin/safari-cli → accessible system-wide
```

### Self-Cleanup
```bash
safari-cli cleanup-permissions
# Runs: tccutil reset AppleEvents com.apple.Safari
```

## Comparison to Other Tools

| Feature | osascript | osascript-wrapper | sdef-to-cobra | safari-cli |
|---------|-----------|------------------|---------------|------------|
| Embedded API | ❌ | ❌ | ❌ | ✅ |
| Permissions | Manual | Manual | Manual | ✅ macgo |
| Global install | ✅ | Via PATH | Via PATH | ✅ Makefile |
| Self-cleanup | ❌ | ❌ | ❌ | ✅ |
| Lazy loading | ❌ | ❌ | ❌ | ✅ |
| Type safety | ❌ | ❌ | ✅ | ✅ |

## Complete Workflow

```
User
  ↓
safari-cli open https://golang.org
  ↓
ensurePermissions() (lazy)
  ↓
macgo.Start() (if first time)
  ↓
System Settings prompt
  ↓
User approves
  ↓
osascript executes
  ↓
Safari opens URL
  ↓
✓ Success
```

## Production Ready

This tool is ready for:
- Personal automation scripts
- CI/CD pipelines
- Developer workflows
- System administration
- Browser automation

All with proper permissions management and cleanup!
