# Examples Comparison: v1 vs v2

## 📊 Overall Metrics

| Metric | v1 Examples | v2 Examples | Improvement |
|--------|------------|-------------|-------------|
| Average Lines | 80-100 | 30-50 | **50% less** |
| Setup Lines | 20-30 (in init) | 3-5 | **85% less** |
| Concepts to Learn | 10+ | 3 | **70% less** |
| Global State | Yes (init) | No | **✅ Eliminated** |

## 🔄 Example-by-Example Comparison

### 1. **Hello World**

**v1 (79 lines):**
```go
func init() {
    debug.Init()
    macgo.EnableDebug()
    macgo.SetAppName("HelloMacgoApp")
    macgo.SetBundleID("com.example.hellomacgo")
    macgo.SetIconFile("/path/to/icon.icns")
    macgo.EnableImprovedSignalHandling()
    macgo.RequestEntitlements(
        macgo.EntAppSandbox,
        macgo.EntCamera,
        macgo.EntMicrophone,
    )
    macgo.Start()
}
```

**v2 (20 lines):**
```go
func main() {
    err := macgo.Request(macgo.Camera, macgo.Microphone)
    if err != nil {
        log.Fatal(err)
    }
    // Your app code...
}
```

**Improvement:** 75% less code, no init() magic

---

### 2. **Getting Started**

**v1 (101 lines):**
- 30+ lines in init() for configuration
- Multiple API calls for setup
- Debug package import
- Global state modifications

**v2 (50 lines):**
- 5 lines for complete configuration
- Single Start() call
- Explicit configuration struct
- No global state

**Key Difference:** Configuration is explicit and localized

---

### 3. **Sandboxed File Access**

**v1 (84 lines):**
```go
import _ "github.com/tmc/misc/macgo/auto/sandbox"
// Magic happens at import!
```

**v2 (50 lines):**
```go
cfg := &macgo.Config{
    Permissions: []macgo.Permission{macgo.Files},
}
err := macgo.Start(cfg)
```

**Key Difference:** No magic imports, explicit sandbox configuration

---

### 4. **Camera & Microphone**

**v1 Approach:**
```go
func init() {
    macgo.RequestEntitlement(macgo.EntCamera)
    macgo.RequestEntitlement(macgo.EntMicrophone)
    macgo.RequestEntitlement(macgo.EntAppSandbox)
    macgo.EnableImprovedSignalHandling()
    macgo.EnableDebug()
    macgo.Start()
}
```

**v2 Approach:**
```go
err := macgo.Request(macgo.Camera, macgo.Microphone)
```

**Improvement:** One line vs 6+ lines

---

## 🎯 Pattern Comparison

### Simple Permission Request

| v1 | v2 |
|----|-----|
| 15+ lines in init() | 1 line in main() |
| Global state | Local configuration |
| Multiple function calls | Single Request() call |
| Import side effects | Explicit execution |

### Full Configuration

| v1 | v2 |
|----|-----|
| Scattered across init() | Single Config struct |
| 10+ function calls | 1 Start() call |
| Hard to test | Easy to test |
| Hidden behavior | Visible at call site |

### Environment-Driven

| v1 | v2 |
|----|-----|
| Automatic in init() | Explicit Auto() call |
| Always reads env | Only when requested |
| Can't control timing | Full control |
| Magic behavior | Clear intention |

## 📝 Migration Examples

### Simplest Migration

**From v1:**
```go
import (
    "github.com/tmc/misc/macgo"
    _ "github.com/tmc/misc/macgo/auto"
)

func main() {
    // Permissions already set by auto import
    doWork()
}
```

**To v2:**
```go
import macgo "github.com/tmc/misc/macgo/v2"

func main() {
    macgo.Auto() // Explicit!
    doWork()
}
```

### Complex Migration

**From v1:**
```go
func init() {
    macgo.SetAppName("MyApp")
    macgo.SetBundleID("com.example.myapp")
    macgo.RequestEntitlement(macgo.EntAppSandbox)
    macgo.RequestEntitlement(macgo.EntCamera)
    macgo.RequestEntitlement(macgo.EntMicrophone)
    macgo.RequestEntitlement(macgo.EntLocation)
    macgo.RequestEntitlement(macgo.EntPhotos)
    macgo.RequestEntitlement(macgo.EntNetworkClient)
    macgo.RequestEntitlement(macgo.EntUserSelectedReadOnly)
    macgo.EnableDebug()
    macgo.Start()
}
```

**To v2:**
```go
func main() {
    cfg := &macgo.Config{
        AppName:  "MyApp",
        BundleID: "com.example.myapp",
        Permissions: []macgo.Permission{
            macgo.Camera,
            macgo.Microphone,
            macgo.Location,
            macgo.Files,   // Covers sandbox + file access
            macgo.Network, // Covers network access
        },
        Custom: []string{
            "com.apple.security.personal-information.photos-library",
        },
        Debug: true,
    }
    macgo.Start(cfg)
}
```

## 🏆 Why v2 Examples Are Better

1. **No Hidden Behavior:** Everything happens where you can see it
2. **Testable:** Pass different configs for different tests
3. **Composable:** Build configs programmatically
4. **Clear Intent:** One look shows what permissions are requested
5. **No Surprises:** No init() functions running at import
6. **Simpler Mental Model:** 3 concepts vs 10+

## 📚 Complete v2 Examples

Available in `/v2/examples/`:
- `hello/` - Simplest example
- `getting-started/` - Basic patterns
- `camera-mic/` - Media permissions
- `sandboxed-file-exec/` - Sandbox and file access

Each example is:
- ✅ Self-contained
- ✅ Under 100 lines
- ✅ Fully commented
- ✅ Shows alternatives
- ✅ No global state