# macgo Examples

This directory contains example applications demonstrating various features of the macgo library.

## Quick Start Examples

### 📚 [getting-started](getting-started/)
Basic example showing the simplest way to use macgo with a single permission request.

```go
err := macgo.Request(macgo.Camera)
```

### 👋 [hello](hello/)
Classic "Hello, World!" as a macOS app bundle with debug logging to demonstrate the bundle creation process.

### 📹 [camera-mic](camera-mic/)
Request camera and microphone permissions for media applications. Shows how to request multiple permissions at once.

## File Access Examples

### 📁 [hello-files](hello-files/)
Demonstrates file system access with proper permissions. Creates a test file on the Desktop to verify access.

### 🖥️ [desktop-list](desktop-list/)
Advanced file listing with command-line flags for different signing strategies:
- `-debug` - Enable debug output
- `-ad-hoc` - Use ad-hoc signing
- `-auto-sign` - Auto-detect Developer ID
- `-sign <identity>` - Use specific signing identity

### 🔒 [sandboxed-file-exec](sandboxed-file-exec/)
Execute files within the macOS app sandbox. Demonstrates the difference between sandboxed and non-sandboxed file access.

## Code Signing Examples

### ✍️ [code-signing](code-signing/)
Comprehensive code signing demonstration:
- Ad-hoc signing for development
- Developer ID signing for distribution
- Custom signing identities
- Verification of signed bundles

### 🔐 [auto-signed](auto-signed/)
Automatic code signing using the `auto/signed` package. Detects and uses available Developer ID certificates.

## Advanced Examples

### 🍎 [osascript-wrapper](osascript-wrapper/)
Execute AppleScript from Go applications:
- Bundle AppleScript files within the app
- Execute scripts with proper permissions
- Handle script output and errors

### 🔄 [permission-reset-test](permission-reset-test/)
Test permission reset functionality using `tccutil`. Useful for development and testing workflows.

### 📸 [screen-capture](screen-capture/)
Screen recording and capture with proper entitlements. Demonstrates screen capture permissions.

## Test Examples

These examples are primarily for testing macgo's internal functionality:

- **comprehensive-io-test** - Test I/O handling in bundles
- **env-test** - Verify environment variable propagation
- **signal-test** - Test signal handling
- **stdio-test** - Test stdin/stdout/stderr redirection
- **test-signing** - Verify signing configurations

## Running Examples

### Direct Execution
Run any example directly with `go run`:

```bash
cd getting-started
go run .
```

### With Environment Variables
Configure behavior via environment:

```bash
# Enable debug output
MACGO_DEBUG=1 go run .

# Use ad-hoc signing
MACGO_AD_HOC_SIGN=1 go run .

# Keep bundle for inspection
MACGO_KEEP_BUNDLE=1 go run .

# Custom app name
MACGO_APP_NAME="My Test App" go run .
```

### Building Standalone Apps
Build as a regular executable:

```bash
go build -o myapp
./myapp
```

The executable will create its bundle on first run.

## Common Patterns

### Basic Permission Request
```go
package main

import (
    "log"
    "github.com/tmc/macgo"
)

func main() {
    err := macgo.Request(macgo.Camera)
    if err != nil {
        log.Fatal(err)
    }
    // Use camera...
}
```

### Multiple Permissions
```go
err := macgo.Request(
    macgo.Camera,
    macgo.Microphone,
    macgo.Files,
)
```

### Custom Configuration
```go
cfg := macgo.NewConfig().
    WithAppName("MyApp").
    WithBundleID("com.example.myapp").
    WithPermissions(macgo.Camera).
    WithAdHocSign().
    WithDebug()

err := macgo.Start(cfg)
```

### Using Auto Packages
```go
import (
    _ "github.com/tmc/macgo/auto/media"   // Camera + Microphone
    _ "github.com/tmc/macgo/auto/adhoc"   // Ad-hoc signing
    "github.com/tmc/macgo"
)

func main() {
    // Permissions and signing pre-configured
    macgo.Request()
}
```

## Troubleshooting

### Bundle Location
By default, bundles are created in:
- `$GOPATH/bin/` (if GOPATH is set)
- `/tmp/` (fallback location)
- Use `MACGO_KEEP_BUNDLE=1` to preserve bundles

### Code Signing Issues
- Ensure Xcode Command Line Tools are installed: `xcode-select --install`
- List available identities: `security find-identity -v -p codesigning`
- Use `-debug` flag for detailed signing output

### Permission Prompts
- Permissions are requested on first run
- Use permission-reset-test example to reset for testing
- Check System Settings → Privacy & Security for current status

### Debug Output
Enable debug mode to see:
- Bundle creation process
- Entitlements being added
- Code signing commands
- Relaunch behavior

```bash
MACGO_DEBUG=1 go run .
```

## Requirements

- Go 1.21 or later
- macOS 11.0 or later
- Xcode Command Line Tools (for code signing)