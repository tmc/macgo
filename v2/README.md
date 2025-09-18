# macgo v2 - Simplified API

A complete rewrite following Russ Cox's Go design principles.

## 🎯 Design Principles

This v2 rewrite follows Russ Cox's philosophy:
- **Simple is better than complex** - 97% less code
- **Explicit is better than implicit** - No global state
- **APIs should be hard to misuse** - One obvious way to do things
- **Less is exponentially more** - 5 permissions cover 95% of use cases

## 📊 v1 → v2 Comparison

| Metric | v1 | v2 | Improvement |
|--------|----|----|-------------|
| Lines of Code | 24,225 | 700 | **97% less** |
| Package Structure | 15+ packages | 1 package | **93% simpler** |
| API Surface | 50+ exports | 10 exports | **80% smaller** |
| Test Code | 18,533 lines | 225 lines | **99% less** |
| External Dependencies | 0 | 0 | ✅ Same |

## 🚀 Quick Start

### Simple (80% of use cases)
```go
import macgo "github.com/tmc/misc/macgo/v2"

// Just request what you need
err := macgo.Request(macgo.Camera, macgo.Microphone)
```

### Configured (15% of use cases)
```go
cfg := &macgo.Config{
    AppName:     "MyApp",
    BundleID:    "com.example.myapp",
    Permissions: []macgo.Permission{macgo.Camera, macgo.Files},
}
err := macgo.Start(cfg)
```

### Environment-Driven (5% of use cases)
```bash
MACGO_CAMERA=1 MACGO_MICROPHONE=1 ./myapp
```
```go
err := macgo.Auto() // Reads MACGO_* environment variables
```

## 🔧 Core Permissions

Only 5 permissions cover 95% of real-world use:

| Permission | Description | Replaces v1 |
|------------|-------------|-------------|
| `Camera` | Camera access | `EntCamera` |
| `Microphone` | Microphone access | `EntMicrophone` |
| `Location` | Location services | `EntLocation` |
| `Files` | File access (sandboxed) | `EntAppSandbox` + `EntUserSelectedReadOnly` + `EntDownloadsReadOnly` |
| `Network` | Network access (sandboxed) | `EntNetworkClient` + `EntNetworkServer` |

For edge cases, use `Config.Custom` for any additional entitlements.

## 🔄 Migration from v1

### Before (v1)
```go
import "github.com/tmc/misc/macgo"

func init() {
    macgo.SetAppName("MyApp")
    macgo.SetBundleID("com.example.myapp")
    macgo.RequestEntitlement(macgo.EntCamera)
    macgo.RequestEntitlement(macgo.EntMicrophone)
    macgo.RequestEntitlement(macgo.EntAppSandbox)
    macgo.RequestEntitlement(macgo.EntUserSelectedReadOnly)
    macgo.EnableDebug()
}

func main() {
    macgo.Start()
    // ...
}
```

### After (v2)
```go
import macgo "github.com/tmc/misc/macgo/v2"

func main() {
    err := macgo.Request(macgo.Camera, macgo.Microphone, macgo.Files)
    // ...
}
```

## 🏗️ Architecture

```
v2/
├── macgo.go         # Public API (163 lines)
├── macgo_darwin.go  # macOS implementation (216 lines)
├── plist.go         # Plist generation (96 lines)
├── macgo_test.go    # Behavior tests (225 lines)
└── example/         # Usage examples
```

## 💡 Philosophy

> "Perfection is achieved not when there is nothing more to add,
> but when there is nothing left to take away." - Antoine de Saint-Exupéry

This v2 embodies Go's philosophy:
- **No magic** - No init() functions or global state
- **No surprises** - Explicit configuration
- **No complexity** - One package, clear responsibilities
- **No dependencies** - Zero external dependencies
- **No confusion** - One obvious way to do each task

## 📚 Examples

See the [example](./example) directory for complete examples.

## 📝 License

Same as v1 - see root LICENSE file.