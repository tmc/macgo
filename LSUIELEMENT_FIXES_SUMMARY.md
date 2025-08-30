# LSUIElement Logic Error Fixes Summary

## Critical Issues Fixed

### 1. **DefaultConfig LSUIElement Contradiction** (Line 157)
**Issue**: DefaultConfig was setting `LSUIElement: false` with comment "Hide dock icon" but LSUIElement=false actually SHOWS the dock icon.

**Fix**: Changed `DefaultConfig` to set `LSUIElement: true` with updated comment:
```go
// BEFORE (BUGGY):
"LSUIElement": false, // Hide dock icon and app menu by default

// AFTER (FIXED):
"LSUIElement": true, // Hide dock icon and app menu by default (true = hidden)
```

### 2. **Environment Variable Logic Error** (Line 277)
**Issue**: `MACGO_SHOW_DOCK_ICON=1` was setting `LSUIElement=true` which HIDES the dock icon, but the environment variable name suggests it should SHOW the dock icon.

**Fix**: Changed environment variable logic to set `LSUIElement=false` when showing dock icon:
```go
// BEFORE (BUGGY):
DefaultConfig.PlistEntries["LSUIElement"] = true

// AFTER (FIXED):
DefaultConfig.PlistEntries["LSUIElement"] = false
```

### 3. **Comment Clarity** (Line 495)
**Enhancement**: Added clarity to the comment in `NewConfig()` to make LSUIElement behavior explicit:
```go
// BEFORE:
"LSUIElement": true, // Hide dock icon and app menu by default

// AFTER:
"LSUIElement": true, // Hide dock icon and app menu by default (true = hidden)
```

### 4. **Test Expectations Updated** (Line 1611)
**Fix**: Updated test expectations in `macgo_test.go` to match the corrected behavior:
```go
// BEFORE (INCORRECT EXPECTATION):
if val, exists := DefaultConfig.PlistEntries["LSUIElement"]; !exists || val != false {
    t.Error("Expected LSUIElement to be false by default in DefaultConfig")
}

// AFTER (CORRECT EXPECTATION):
if val, exists := DefaultConfig.PlistEntries["LSUIElement"]; !exists || val != true {
    t.Error("Expected LSUIElement to be true by default in DefaultConfig (dock icon hidden)")
}
```

## LSUIElement Behavior Reference

**macOS LSUIElement Behavior**:
- `LSUIElement = true` → App does NOT appear in dock (hidden, runs in background)
- `LSUIElement = false` → App DOES appear in dock (visible, normal app behavior)

## Functions Working Correctly

The following functions were already working correctly and continue to work:

1. **`NewConfig()`** - Correctly sets `LSUIElement: true` to hide dock icon by default
2. **`EnableDockIcon()`** - Correctly sets `LSUIElement: false` to show dock icon
3. **API Comment in `api.go`** - Correctly states "By default, macgo applications run as background applications (LSUIElement=true)"

## Impact of Changes

### Before Fixes
- **DefaultConfig**: Showed dock icon by default (LSUIElement=false) despite comment saying "hide dock icon"
- **MACGO_SHOW_DOCK_ICON=1**: Hid dock icon (LSUIElement=true) despite variable name suggesting it should show
- **Inconsistent behavior**: Different functions had different default behaviors

### After Fixes
- **DefaultConfig**: Correctly hides dock icon by default (LSUIElement=true) matching comment
- **MACGO_SHOW_DOCK_ICON=1**: Correctly shows dock icon (LSUIElement=false) matching variable name
- **Consistent behavior**: All functions now behave consistently with LSUIElement semantics

## User Experience Impact

### For Default Users
- **Before**: Apps appeared in dock by default (despite documentation saying they wouldn't)
- **After**: Apps are hidden from dock by default (as documented and expected)

### For Environment Variable Users
- **Before**: `MACGO_SHOW_DOCK_ICON=1` would hide dock icon (counterintuitive)
- **After**: `MACGO_SHOW_DOCK_ICON=1` correctly shows dock icon (intuitive)

### For API Users
- **Before**: `DefaultConfig` and `NewConfig()` had different default behaviors
- **After**: All config creation methods now consistently hide dock icon by default

## Testing Coverage

Created comprehensive tests to verify:
1. Default behavior correctness
2. Environment variable behavior
3. Function consistency
4. Regression prevention

All tests pass, confirming that the fixes work correctly and don't break existing functionality.

## Files Modified

1. `/Volumes/tmc/go/src/github.com/tmc/misc/macgo/macgo.go` - Main logic fixes
2. `/Volumes/tmc/go/src/github.com/tmc/misc/macgo/macgo_test.go` - Test expectations updated

## Backward Compatibility

These changes may affect existing applications that were relying on the buggy behavior. However, since the buggy behavior was inconsistent with the documentation and user expectations, this is considered a critical bug fix rather than a breaking change.

Users who want to maintain the old behavior (showing dock icon by default) can:
1. Set `MACGO_SHOW_DOCK_ICON=1` environment variable
2. Use `EnableDockIcon()` function
3. Manually set `LSUIElement=false` in their configuration

## Summary

The LSUIElement logic errors have been successfully fixed, resulting in:
- ✅ Consistent behavior across all configuration methods
- ✅ Correct environment variable behavior
- ✅ Accurate comments and documentation
- ✅ Proper default behavior (dock icon hidden by default)
- ✅ Comprehensive test coverage
- ✅ No breaking changes to correct functionality