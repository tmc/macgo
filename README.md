# macgo

[![Go Reference](https://pkg.go.dev/badge/github.com/tmc/misc/macgo.svg)](https://pkg.go.dev/github.com/tmc/misc/macgo)
[![Go Report Card](https://goreportcard.com/badge/github.com/tmc/misc/macgo)](https://goreportcard.com/report/github.com/tmc/misc/macgo)

**macgo** automatically creates and launches macOS app bundles to enable TCC (Transparency, Consent, and Control) permissions for command-line Go programs.

## Problem Solved

On macOS, command-line binaries cannot access protected resources like camera, microphone, location, or user files because macOS requires applications to be properly bundled and signed to request these permissions. macgo bridges this gap by automatically wrapping your Go binary in a properly configured app bundle.

## Quick Start

### Simplest Usage

```go
package main

import _ "github.com/tmc/misc/macgo/auto/sandbox"

func main() {
    // Your code automatically runs with app sandbox enabled
}
```

### Basic Configuration

```go
package main

import "github.com/tmc/misc/macgo"

func init() {
    macgo.RequestEntitlements(
        macgo.EntAppSandbox,
        macgo.EntCamera,
        macgo.EntMicrophone,
    )
    macgo.Start()
}

func main() {
    // Your code runs with requested permissions
}
```

### Advanced Configuration

```go
package main

import "github.com/tmc/misc/macgo"

func init() {
    cfg := macgo.NewConfig()
    cfg.ApplicationName = "MyApp"
    cfg.BundleID = "com.example.myapp"
    cfg.RequestEntitlements(macgo.EntCamera, macgo.EntUserSelectedReadOnly)
    cfg.AddPlistEntry("LSUIElement", false) // Show in dock

    macgo.Configure(cfg)
    macgo.EnableImprovedSignalHandling() // Better Ctrl+C support
    macgo.Start()
}
```

## Available Entitlements

### TCC Permissions
- `EntCamera` - Camera access
- `EntMicrophone` - Microphone access
- `EntLocation` - Location services
- `EntAddressBook` - Contacts access
- `EntPhotos` - Photos library access
- `EntCalendars` - Calendar access
- `EntReminders` - Reminders access

### App Sandbox
- `EntAppSandbox` - Enable app sandbox
- `EntUserSelectedReadOnly` - Read access to user-selected files
- `EntUserSelectedReadWrite` - Read/write access to user-selected files
- `EntNetworkClient` - Outgoing network connections¹
- `EntNetworkServer` - Incoming network connections¹

### Hardware & Development
- `EntBluetooth` - Bluetooth device access
- `EntUSB` - USB device access
- `EntAllowJIT` - JIT compilation
- `EntDebugger` - Debugger attachment

¹ *Network entitlements only affect Objective-C/Swift APIs, not Go's standard networking*

## Environment Variables

```bash
export MACGO_APP_NAME="MyApp"
export MACGO_BUNDLE_ID="com.example.myapp"
export MACGO_CAMERA=1
export MACGO_MIC=1
export MACGO_SANDBOX=1
export MACGO_DEBUG=1

./myapp
```

## Auto-initialization Packages

```go
// Basic - no sandbox
import _ "github.com/tmc/misc/macgo/auto"

// With app sandbox
import _ "github.com/tmc/misc/macgo/auto/sandbox"

// With sandbox + user file read access
import _ "github.com/tmc/misc/macgo/auto/sandbox/readonly"

// With improved signal handling (better Ctrl+C)
import _ "github.com/tmc/misc/macgo/auto/sandbox/signalhandler"
```

## How It Works

1. **Detection**: Checks if already running in an app bundle
2. **Bundle Creation**: Creates `.app` bundle with entitlements and Info.plist
3. **Code Signing**: Automatically signs the bundle (ad-hoc by default)
4. **Relaunching**: Relaunches the process inside the app bundle
5. **I/O Forwarding**: Maintains stdin/stdout/stderr connectivity
6. **Signal Forwarding**: Forwards signals (including Ctrl+C) between processes

## Key Features

- **Zero Configuration**: Works out-of-the-box with defaults
- **Clean Architecture**: Modular design following Go best practices
- **Signal Handling**: Robust Ctrl+C and signal forwarding
- **Automatic Cleanup**: Temporary bundles cleaned up automatically
- **Code Signing**: Built-in ad-hoc signing with custom identity support
- **Environment Support**: Configure via environment variables
- **Debug Mode**: Detailed logging when `MACGO_DEBUG=1`

## Architecture

### Core Packages
- **`macgo`** - Main API and configuration
- **`bundle`** - App bundle creation and management
- **`security`** - Code signing and validation
- **`signal`** - Signal forwarding and process management
- **`entitlements`** - Entitlement definitions and constants
- **`process`** - Process launching and I/O handling

### Clean Interfaces
```go
type BundleCreator interface {
    Create(ctx context.Context, cfg *Config, execPath string) (string, error)
    Exists(cfg *Config, execPath string) (bool, error)
}

type SignalForwarder interface {
    Forward(ctx context.Context, target *os.Process) error
    Stop() error
}
```

## Examples

See `examples/` directory for complete examples:
- `examples/hello/` - Basic usage
- `examples/advanced/` - Advanced configuration
- `examples/signals/` - Signal handling
- `examples/sandbox/` - Sandboxed execution

## Requirements

- **macOS 10.15+** (Catalina or later)
- **Go 1.19+**
- **Xcode Command Line Tools** (for code signing)

## Cross-Platform Compatibility

macgo is designed to work gracefully on all platforms. While macgo functionality is macOS-specific, you can safely use it in cross-platform applications:

- **On macOS**: Full functionality - creates app bundles, handles TCC permissions
- **On other platforms**: All macgo functions are safe no-ops that do not affect execution
- **No build constraints needed**: Import and use macgo directly without `//go:build` tags

```go
package main

import "github.com/tmc/misc/macgo"

func main() {
    // Works on all platforms - no-op on non-macOS
    macgo.RequestEntitlements(macgo.EntCamera, macgo.EntMicrophone)
    macgo.Start()

    // Your cross-platform application code here
    // ...
}
```

Enable debug logging with `MACGO_DEBUG=1` to see platform-specific behavior.

## Installation

```bash
go get github.com/tmc/misc/macgo
```

## Testing

```bash
go test ./...                    # Run all tests
MACGO_DEBUG=1 go test -v ./...   # With debug output
go test -run TestBundle ./...    # Specific test
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure `go test ./...` passes
5. Run `go fmt ./...`
6. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.