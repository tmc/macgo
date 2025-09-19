# macgo v2 Examples

These examples demonstrate the simplified v2 API following Russ Cox's design principles.

## 🚀 Quick Start

All examples follow the same pattern:

1. **Simple approach** - One-liner for common cases
2. **Configured approach** - Full Config struct for complex cases
3. **Environment approach** - For deployment scenarios

## 📁 Available Examples

### 1. **[hello](./hello/)** - Simplest Example
```go
err := macgo.Request(macgo.Camera, macgo.Microphone)
```
- **Purpose:** Minimal working example
- **Lines:** ~30 (vs 79 in v1)
- **Shows:** Basic permission request

### 2. **[getting-started](./getting-started/)** - Core Patterns
```go
cfg := &macgo.Config{
    AppName: "MyApp",
    Permissions: []macgo.Permission{macgo.Camera},
}
err := macgo.Start(cfg)
```
- **Purpose:** Main patterns and approaches
- **Lines:** ~50 (vs 101 in v1)
- **Shows:** Configuration, context, alternatives

### 3. **[sandboxed-file-exec](./sandboxed-file-exec/)** - Sandbox & Files
```go
err := macgo.Request(macgo.Files) // Sandbox + file access
```
- **Purpose:** File access and sandbox restrictions
- **Lines:** ~60 (vs 84 in v1)
- **Shows:** Sandbox behavior, file access testing

### 4. **[camera-mic](./camera-mic/)** - Media Permissions
```go
err := macgo.Request(macgo.Camera, macgo.Microphone)
```
- **Purpose:** Camera and microphone access
- **Lines:** ~70 (new in v2)
- **Shows:** Media device access, permission testing

## 🔨 Advanced Examples

### 5. **[file-processor](./file-processor/)** - File Processing CLI
```go
cfg := &macgo.Config{
    AppName: "FileProcessor",
    Permissions: []macgo.Permission{macgo.Files},
    LSUIElement: true, // Hide from dock
}
```
- **Purpose:** Batch file processing with sandbox permissions
- **Shows:** CLI tool patterns, file transformations, error handling

### 6. **[screen-recorder](./screen-recorder/)** - Screen Recording
```go
permissions := []macgo.Permission{macgo.Screen, macgo.Files}
if withAudio { permissions = append(permissions, macgo.Microphone) }
if withCamera { permissions = append(permissions, macgo.Camera) }
```
- **Purpose:** Screen recording with optional audio/camera
- **Shows:** Conditional permissions, hardware acceleration, device enumeration

### 7. **[network-service](./network-service/)** - HTTP/WebSocket Server
```go
cfg := &macgo.Config{
    AppName: "NetworkService",
    Permissions: []macgo.Permission{macgo.Network},
    LSUIElement: background,
}
```
- **Purpose:** Network service with sandbox networking
- **Shows:** REST API, WebSocket, external connectivity testing

### 8. **[background-agent](./background-agent/)** - Background Service
```go
cfg := &macgo.Config{
    AppName: "BackgroundAgent",
    Permissions: []macgo.Permission{macgo.Files, macgo.Network},
    LSUIElement: true,
    LSBackgroundOnly: true,
}
```
- **Purpose:** Long-running background daemon service
- **Shows:** File monitoring, periodic tasks, launch agent configuration

### 9. **[dev-tools](./dev-tools/)** - Development Utilities
```go
cfg := &macgo.Config{
    AppName: "DevTools",
    Permissions: []macgo.Permission{macgo.Files, macgo.Network},
    LSUIElement: true,
}
```
- **Purpose:** Developer utilities for project analysis and building
- **Shows:** Language detection, build/test execution, IDE integration

### 10. **[media-processor](./media-processor/)** - Media Processing
```go
permissions := []macgo.Permission{macgo.Files, macgo.Network}
if liveCapture {
    permissions = append(permissions, macgo.Camera, macgo.Microphone)
}
```
- **Purpose:** Audio/video processing with hardware acceleration
- **Shows:** Hardware encoding, format conversion, live capture, batch processing

## 🔧 Running Examples

### Quick Test (No Relaunch)
```bash
cd v2/examples/hello
MACGO_NO_RELAUNCH=1 go run main.go
```

### Full Test (With Bundle Creation)
```bash
cd v2/examples/hello
go run main.go
```

### Build and Test
```bash
cd v2/examples/hello
go build -o hello-app
./hello-app
```

## 📊 Comparison with v1

| Example | v1 Lines | v2 Lines | Improvement |
|---------|----------|----------|-------------|
| hello | 79 | 30 | 62% less |
| getting-started | 101 | 50 | 50% less |
| sandboxed-file-exec | 84 | 60 | 29% less |
| camera-mic | N/A | 70 | New & simple |

## 🎯 Key v2 Benefits Shown

1. **No Global State** - All examples use explicit configuration
2. **Simple API** - 1-3 lines vs 10+ lines for setup
3. **Clear Intent** - Configuration visible at call site
4. **Easy Testing** - Pass different configs for different scenarios
5. **No Magic** - No init() functions or import side effects

## 🔄 Migration Guide

See [MIGRATION_GUIDE.md](../MIGRATION_GUIDE.md) for detailed migration instructions from v1 to v2.

## 📝 Example Template

New examples should follow this structure:

```go
package main

import macgo "github.com/tmc/misc/macgo/v2"

func main() {
    // Simple approach
    err := macgo.Request(macgo.Camera)
    if err != nil {
        log.Fatal(err)
    }

    // Your app code here...
}

// Alternative: Configured approach
func withConfig() {
    cfg := &macgo.Config{
        AppName: "MyApp",
        Permissions: []macgo.Permission{macgo.Camera},
        Debug: true,
    }

    err := macgo.Start(cfg)
    // ...
}

// Alternative: Environment approach
func withEnv() {
    // MACGO_CAMERA=1 MACGO_DEBUG=1 ./myapp
    err := macgo.Auto()
    // ...
}
```

All examples demonstrate these three patterns for maximum utility.