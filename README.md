# macgo

A Go library for building macOS applications with proper permissions, entitlements, and bundle structure.

[![GoDoc](https://pkg.go.dev/badge/github.com/tmc/macgo)](https://pkg.go.dev/github.com/tmc/macgo)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Overview

macgo simplifies the process of creating macOS applications in Go by automatically handling:
- App bundle creation and management
- Permission requests (camera, microphone, files, etc.)
- Code signing (ad-hoc and Developer ID)
- Entitlements and sandboxing
- TCC (Transparency, Consent, and Control) integration

## Installation

```bash
go get github.com/tmc/macgo
```

## Quick Start

```go
package main

import (
    "log"
    "github.com/tmc/macgo"
)

func main() {
    // Request camera permission with automatic app bundle creation
    err := macgo.Request(macgo.Camera)
    if err != nil {
        log.Fatal(err)
    }

    // Your application code here
}
```

## Core Features

### Simple Permission Requests

Request macOS permissions with a single function call:

```go
// Request single permission
macgo.Request(macgo.Camera)

// Request multiple permissions
macgo.Request(macgo.Camera, macgo.Microphone, macgo.Files)
```

### Automatic Bundle Creation

macgo automatically creates a proper `.app` bundle structure with:
- Info.plist with required metadata
- Entitlements for requested permissions
- Proper executable location
- App icons (if provided)

### Code Signing Support

Built-in support for code signing:

```go
// Ad-hoc signing (development)
cfg := macgo.NewConfig().
    WithAppName("MyApp").
    WithAdHocSign()

// Developer ID signing (distribution)
cfg := macgo.NewConfig().
    WithAppName("MyApp").
    WithAutoSign() // Auto-detects Developer ID
```

### Environment Configuration

Configure via environment variables:

```bash
MACGO_APP_NAME=MyApp          # Application name
MACGO_BUNDLE_ID=com.example   # Bundle identifier
MACGO_AD_HOC_SIGN=1          # Enable ad-hoc signing
MACGO_AUTO_SIGN=1             # Auto-detect signing identity
MACGO_DEBUG=1                 # Debug output
MACGO_FORCE_DIRECT=1          # Force direct execution (bypass LaunchServices)
MACGO_FORCE_LAUNCH_SERVICES=1 # Force use of LaunchServices
```

## Available Permissions

- **Camera** (`macgo.Camera`) - Camera access
- **Microphone** (`macgo.Microphone`) - Microphone access
- **Location** (`macgo.Location`) - Location services
- **Files** (`macgo.Files`) - File system access
- **Network** (`macgo.Network`) - Network connections
- **Sandbox** (`macgo.Sandbox`) - App sandboxing

## Advanced Usage

### Custom Configuration

```go
cfg := macgo.NewConfig().
    WithAppName("MyApp").
    WithBundleID("com.example.myapp").
    WithPermissions(macgo.Camera, macgo.Microphone).
    WithAppGroups("group.com.example.shared").
    WithDebug()

err := macgo.Start(cfg)
```

### Auto Packages

Import auto packages for automatic configuration:

```go
import (
    _ "github.com/tmc/macgo/auto/media"   // Camera + Microphone
    _ "github.com/tmc/macgo/auto/adhoc"   // Ad-hoc signing
    "github.com/tmc/macgo"
)

func main() {
    // Permissions and signing are pre-configured
    macgo.Request()
}
```

## Package Structure

- **`macgo`** - Core library and main API
- **`bundle/`** - App bundle creation and management
- **`codesign/`** - Code signing utilities
- **`permissions/`** - Permission definitions and validation
- **`teamid/`** - Team ID detection for signing
- **`auto/`** - Auto-configuration packages
- **`examples/`** - Example applications
- **`internal/`** - Internal implementation packages

## Examples

See the [`examples/`](examples/) directory for complete examples:

- [`getting-started`](examples/getting-started/) - Basic usage
- [`camera-mic`](examples/camera-mic/) - Media permissions
- [`desktop-list`](examples/desktop-list/) - File access
- [`code-signing`](examples/code-signing/) - Signing examples
- [`sandboxed-file-exec`](examples/sandboxed-file-exec/) - Sandboxed file access
- [`press-keys`](examples/press-keys/) - Keyboard event simulation
- [`tee-see-see`](examples/tee-see-see/) - TCC database debugger and viewer
- [`lsregister-tool`](examples/lsregister-tool/) - Launch Services database tool

## Requirements

- Go 1.21 or later (1.24+ recommended)
- macOS 11.0 (Big Sur) or later
- Xcode Command Line Tools (for code signing)

**Version Compatibility:**
- macOS 15 (Sequoia): ✅ Fully supported and tested
- macOS 14 (Sonoma): ✅ Fully supported and tested
- macOS 13 (Ventura): ✅ Fully supported and tested
- macOS 12 (Monterey): ⚠️ Limited support (manual testing)
- macOS 11 (Big Sur): ⚠️ Limited support (manual testing)

For detailed version-specific behaviors, quirks, and testing strategies, see:
- [**macOS Version Compatibility Guide**](MACOS_VERSION_COMPATIBILITY.md) - Comprehensive version documentation

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Acknowledgments

This library leverages macOS native frameworks and tools to provide seamless integration with the operating system's security model.