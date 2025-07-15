# README Validation Report for macgo

## Summary
This report identifies documentation inconsistencies between the README.md and the actual macgo API implementation.

## Major Issues Found

### 1. Function Location Mismatch
**Issue**: The README documentation shows many functions as being part of the main `macgo` package when they actually exist only in the `entitlements` package.

**Affected Functions**:
- `SetCamera()`, `SetMic()`, `SetLocation()`, `SetContacts()`, `SetPhotos()`, `SetCalendar()`, `SetReminders()`
- `SetAppSandbox()`, `SetNetworkClient()`, `SetNetworkServer()`
- `SetBluetooth()`, `SetUSB()`, `SetAudioInput()`, `SetPrinting()`
- `SetAllowJIT()`, `SetAllowUnsignedMemory()`, `SetAllowDyldEnvVars()`, `SetDisableLibraryValidation()`, `SetDisableExecutablePageProtection()`, `SetDebugger()`
- `SetAllTCCPermissions()`, `SetAllDeviceAccess()`, `SetAllNetworking()`

**Location in README**: Lines 236-282 and 322-324 suggest these are `macgo.SetXXX()` functions
**Actual Location**: These functions exist only in the `entitlements` package as `entitlements.SetXXX()`

### 2. LSUIElement Default Value Inconsistency
**Issue**: The code has conflicting default values and incorrect comments for LSUIElement.

**In DefaultConfig** (macgo.go:157):
```go
"LSUIElement": false, // Hide dock icon and app menu by default
```
This comment is incorrect - LSUIElement=false means SHOW in dock, not hide.

**In NewConfig()** (macgo.go:495):
```go
"LSUIElement": true, // Hide dock icon and app menu by default
```
This comment is correct - LSUIElement=true means hide from dock.

**README Statement** (line 16): Claims default is `LSUIElement=true` (hidden from dock)
**Actual Default**: DefaultConfig has `LSUIElement=false` (shows in dock)

### 3. Missing Import Path in Example 3
**Issue**: README example 3 (lines 120-148) shows using `entitlements.SetPhotos()` but doesn't import the entitlements package properly.

**Missing Import**:
```go
"github.com/tmc/misc/macgo/entitlements"
```

### 4. Incorrect Environment Variable for Dock Icon
**Issue**: README line 278 states "MACGO_SHOW_DOCK_ICON=1" sets LSUIElement to true, but the code (macgo.go:277) actually sets it to true, which HIDES the dock icon.

The logic is inverted - MACGO_SHOW_DOCK_ICON=1 should set LSUIElement to false.

## Minor Issues

### 1. Deprecated Functions Still Documented
- `DisableAutoInit()` is documented in example 4 but is now a no-op (deprecated)
- `Initialize()` is shown in example 4 but `Start()` is the preferred method

### 2. Missing Documentation
- `EnableImprovedSignalHandling()` function exists and is mentioned in CLAUDE.md but not in README.md
- `SetIconFile()` function exists in api.go but is not documented in README

### 3. Unclear Network Entitlements Note
The note about network entitlements (lines 252-253) appears mid-table, breaking the table structure.

## Recommendations

### 1. Fix Function Documentation
Update the README tables to show the correct package for each function:

```markdown
| Camera     | `entitlements.SetCamera()`   | `macgo.EntCamera`   | `MACGO_CAMERA=1` |
```

Or add wrapper functions in the main macgo package to match the documentation.

### 2. Fix LSUIElement Defaults
- Change DefaultConfig to use `"LSUIElement": true` to match documentation
- Fix the comment on line 157 of macgo.go
- Or update README to reflect actual default behavior

### 3. Fix MACGO_SHOW_DOCK_ICON Logic
The init() function should be:
```go
if os.Getenv("MACGO_SHOW_DOCK_ICON") == "1" {
    DefaultConfig.PlistEntries["LSUIElement"] = false // false = show in dock
}
```

### 4. Update Examples
- Example 3: Add proper import for entitlements package
- Example 4: Remove `DisableAutoInit()` call and use `Start()` instead of `Initialize()`

### 5. Add Missing Documentation
- Document `EnableImprovedSignalHandling()` in the main API section
- Document `SetIconFile()` function
- Consider adding a section about the signalhandler auto-import packages

## Code That Works As Documented

The following aspects of the README are accurate:
- Basic usage patterns (examples 1 and 2)
- `RequestEntitlements()` and `RequestEntitlement()` functions
- `SetAppName()`, `SetBundleID()`, `EnableDockIcon()`
- Environment variable configuration (except MACGO_SHOW_DOCK_ICON)
- Entitlement constants (EntCamera, EntMicrophone, etc.)
- General architecture and workflow descriptions

## Test Results

All code examples from the README were tested for compilation:
- Example 1: Compiles with minor fix (unused variable)
- Example 2: Compiles successfully
- Example 3: Fails - missing import and functions don't exist in macgo package
- Example 4: Compiles but uses deprecated DisableAutoInit()

The convenience functions (`SetAllTCCPermissions`, etc.) and individual setter functions (`SetCamera`, etc.) do not exist in the main macgo package as documented.