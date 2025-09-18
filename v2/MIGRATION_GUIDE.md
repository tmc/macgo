# Migration Guide: macgo v1 → v2

## 🎯 Quick Migration

### Basic Permission Request

**v1 (Complex)**
```go
import "github.com/tmc/misc/macgo"

func init() {
    macgo.RequestEntitlement(macgo.EntCamera)
    macgo.RequestEntitlement(macgo.EntMicrophone)
    macgo.EnableDebug()
}

func main() {
    macgo.Start()
    // your code
}
```

**v2 (Simple)**
```go
import macgo "github.com/tmc/misc/macgo/v2"

func main() {
    err := macgo.Request(macgo.Camera, macgo.Microphone)
    // your code
}
```

## 📋 Permission Mapping

| v1 Entitlement | v2 Permission | Notes |
|----------------|---------------|-------|
| `EntCamera` | `Camera` | Direct mapping |
| `EntMicrophone` | `Microphone` | Direct mapping |
| `EntLocation` | `Location` | Direct mapping |
| `EntAppSandbox`<br>`EntUserSelectedReadOnly`<br>`EntDownloadsReadOnly` | `Files` | Combined into single permission |
| `EntNetworkClient`<br>`EntNetworkServer` | `Network` | Combined into single permission |
| `EntPhotos` | Use `Custom` | `cfg.Custom = []string{"com.apple.security.personal-information.photos-library"}` |
| `EntAddressBook` | Use `Custom` | `cfg.Custom = []string{"com.apple.security.personal-information.addressbook"}` |
| All others | Use `Custom` | Add raw entitlement strings to `Custom` slice |

## 🔄 Configuration Migration

### App Configuration

**v1**
```go
func init() {
    macgo.SetAppName("MyApp")
    macgo.SetBundleID("com.example.myapp")
    macgo.SetIconFile("/path/to/icon.icns")
    macgo.EnableDebug()
    macgo.EnableKeepTemp()
    macgo.DisableRelaunch()
}
```

**v2**
```go
cfg := &macgo.Config{
    AppName:    "MyApp",
    BundleID:   "com.example.myapp",
    Debug:      true,
    KeepBundle: true,
}
// Note: Icon and disable-relaunch handled differently in v2
```

### Environment Variables

**v1**
```bash
MACGO_APP_NAME=MyApp \
MACGO_BUNDLE_ID=com.example.myapp \
MACGO_KEEP_TEMP=1 \
MACGO_SHOW_DOCK_ICON=1 \
./myapp
```

**v2**
```bash
MACGO_APP_NAME=MyApp \
MACGO_BUNDLE_ID=com.example.myapp \
MACGO_KEEP_BUNDLE=1 \
MACGO_CAMERA=1 \
MACGO_MICROPHONE=1 \
./myapp
```
```go
err := macgo.Auto() // Explicitly load from environment
```

## 🚫 Removed Features

These v1 features were removed for simplicity:

| v1 Feature | Why Removed | Alternative |
|------------|-------------|-------------|
| Auto-import packages | Magic behavior | Use explicit `macgo.Request()` |
| Global `init()` | Hidden state | Use explicit configuration |
| `IsInAppBundle()` | Implementation detail | Not needed in v2 |
| Signal handling config | Overcomplicated | Built-in proper handling |
| Custom icon support | Rarely used | Can add if needed |
| 30+ entitlement constants | Too many choices | 5 core + `Custom` |

## ✅ New v2 Benefits

1. **No Global State** - All configuration is explicit
2. **Context Support** - `StartContext()` for lifecycle management
3. **Builder Pattern** - `cfg.WithPermissions().WithDebug()`
4. **Simpler Testing** - Pass different configs for different tests
5. **Smaller Binary** - 97% less code to include

## 🔧 Step-by-Step Migration

1. **Update import**
   ```go
   // Old
   import "github.com/tmc/misc/macgo"

   // New
   import macgo "github.com/tmc/misc/macgo/v2"
   ```

2. **Remove init() function**
   - Move all configuration to main()
   - Convert to Config struct or Request() call

3. **Update permission requests**
   - Map v1 entitlements to v2 permissions (see table above)
   - Use `Custom` for less common entitlements

4. **Replace Start() call**
   ```go
   // Old
   macgo.Start()

   // New - choose one:
   err := macgo.Request(perms...)        // Simple
   err := macgo.Start(cfg)                // Configured
   err := macgo.Auto()                    // From environment
   ```

5. **Test thoroughly**
   - v2 has better error handling - check returned errors
   - Verify permissions work as expected

## 📝 Complete Example Migration

**v1 Full Example**
```go
package main

import (
    "fmt"
    "github.com/tmc/misc/macgo"
    _ "github.com/tmc/misc/macgo/auto/sandbox/signalhandler" // Magic!
)

func init() {
    macgo.SetAppName("PhotoEditor")
    macgo.SetBundleID("com.example.photoeditor")
    macgo.EnableDebug()
    macgo.RequestEntitlements(
        macgo.EntAppSandbox,
        macgo.EntCamera,
        macgo.EntPhotos,
        macgo.EntUserSelectedReadWrite,
        macgo.EntNetworkClient,
    )
}

func main() {
    macgo.Start()
    fmt.Println("Photo editor running...")
}
```

**v2 Equivalent**
```go
package main

import (
    "fmt"
    "log"
    macgo "github.com/tmc/misc/macgo/v2"
)

func main() {
    cfg := &macgo.Config{
        AppName:  "PhotoEditor",
        BundleID: "com.example.photoeditor",
        Permissions: []macgo.Permission{
            macgo.Camera,
            macgo.Files,   // Covers sandbox + file access
            macgo.Network, // Covers network client
        },
        Custom: []string{
            "com.apple.security.personal-information.photos-library",
        },
        Debug: true,
    }

    if err := macgo.Start(cfg); err != nil {
        log.Fatal(err)
    }

    fmt.Println("Photo editor running...")
}
```

## ❓ FAQ

**Q: Is v2 backward compatible?**
A: No, v2 is a complete redesign with breaking changes for simplicity.

**Q: Can I use v1 and v2 in the same project?**
A: Yes, with Go modules you can import both versions.

**Q: What if I need a v1 feature not in v2?**
A: Most features can be achieved with `Custom`. File an issue for missing critical features.

**Q: Is v2 production ready?**
A: Yes, it's simpler and more robust than v1 with better error handling.