# macgo Auto Packages

The `auto` packages provide the simplest way to use macgo - just import and go! These packages use Go's `init()` function to automatically configure macgo with common permission sets.

## 🚀 Quick Start

Pick the auto package that matches your app's needs:

```go
// No permissions needed - just proper macOS app bundling
import _ "github.com/tmc/macgo/auto"

// File access for document processors, editors
import _ "github.com/tmc/macgo/auto/files"

// Camera access for video apps
import _ "github.com/tmc/macgo/auto/camera"

// Network access for web servers, API clients
import _ "github.com/tmc/macgo/auto/network"

// Media capture for screen recorders, conferencing
import _ "github.com/tmc/macgo/auto/media"

// Development tools permissions
import _ "github.com/tmc/macgo/auto/dev"

// Everything enabled (files + network + camera + mic)
import _ "github.com/tmc/macgo/auto/all"
```

That's it! No configuration needed.

## 📦 Available Packages

### Basic Packages

| Package | Permissions | Use Case |
|---------|-------------|----------|
| `auto` | None | Simple CLI tools, basic apps |
| `sandbox` | App Sandbox | Security-focused apps |
| `files` | File Access | Document processors, editors |
| `network` | Network | Web servers, API clients |
| `camera` | Camera | Video capture, photo apps |

### Specialized Packages

| Package | Permissions | Use Case |
|---------|-------------|----------|
| `media` | Camera + Mic + Screen | Screen recorders, conferencing |
| `dev` | Files + Network | Development tools, build systems |
| `all` | Files + Network + Camera + Mic | Full-featured multimedia apps |

## 🎯 Why Auto Packages?

**Manual Style (verbose)**:
```go
func init() {
    cfg := &macgo.Config{
        Permissions: []macgo.Permission{
            macgo.Sandbox,
            macgo.Files,
            macgo.Network,
            macgo.Camera,
            macgo.Microphone,
        },
    }
    macgo.Start(cfg)
}
```

**Auto Style (clean)**:
```go
import _ "github.com/tmc/macgo/auto/all"
```

## 🔧 Manual Configuration

If you need custom configuration, use the explicit API instead:

```go
import "github.com/tmc/macgo"

func main() {
    cfg := &macgo.Config{
        AppName: "MyCustomApp",
        Permissions: []macgo.Permission{macgo.Files, macgo.Network},
        LSUIElement: true, // Hide from dock
        Debug: true,
    }

    err := macgo.Start(cfg)
    // ... your app code
}
```

## 📋 Permission Reference

| Permission | Description | TCC Prompt |
|---------------|-------------|-------------|
| `macgo.Files` | User-selected file access | "Allow access to files you choose" |
| `macgo.Network` | Network client/server | No prompt (sandbox restriction) |
| `macgo.Camera` | Camera access | "Allow camera access" |
| `macgo.Microphone` | Microphone access | "Allow microphone access" |
| `macgo.Screen` | Screen recording | "Allow screen recording" |
| `macgo.Sandbox` | App sandbox isolation | No prompt (enhanced security) |

## ✨ Benefits of Auto Packages

1. **One Line**: Single import vs multiple init() calls
2. **No Global State**: Each import is isolated and predictable
3. **Error Handling**: Graceful degradation vs panics
4. **Cross-Platform**: Safe no-ops on non-macOS systems
5. **Explicit**: Clear intent from import path

## 🧪 Testing

```bash
# Test basic auto package
cd examples/hello
go run main.go

# Test with auto import
echo 'import _ "github.com/tmc/macgo/auto/files"' > test.go
echo 'func main() { println("Hello from auto!") }' >> test.go
go run test.go
```

The auto packages are designed for maximum simplicity while maintaining the power and flexibility of the explicit API when you need it.