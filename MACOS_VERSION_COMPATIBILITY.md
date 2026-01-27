# macOS Version Compatibility Guide

This document provides comprehensive information about macgo compatibility across different macOS versions, version-specific behaviors, testing strategies, and known quirks.

## Supported Versions

macgo officially supports and is tested on:

| macOS Version | Release Name | Version Number | Support Status | Testing Status |
|--------------|--------------|----------------|----------------|----------------|
| macOS 15     | Sequoia      | 15.x           | ✅ Fully Supported | ✅ CI Tested |
| macOS 14     | Sonoma       | 14.x           | ✅ Fully Supported | ✅ CI Tested |
| macOS 13     | Ventura      | 13.x           | ✅ Fully Supported | ✅ CI Tested |
| macOS 12     | Monterey     | 12.x           | ⚠️ Limited Support | ⚠️ Manual Testing Only |
| macOS 11     | Big Sur      | 11.x           | ⚠️ Limited Support | ⚠️ Manual Testing Only |

**Note:** macOS 12 (Monterey) GitHub Actions runners were deprecated in December 2024. macOS 13 (Ventura) runners are scheduled for retirement in December 2025.

### Minimum Requirements

- **macOS Version**: 11.0 (Big Sur) or later
- **Go Version**: 1.21 or later (1.24+ recommended)
- **Xcode Command Line Tools**: Required for code signing functionality

## Version-Specific Behaviors

### TCC (Transparency, Consent, and Control) Changes

#### macOS 15 (Sequoia)
- **TCC Database**: Enhanced privacy protections with stricter permission enforcement
- **System Settings**: Continued evolution of System Settings app (formerly System Preferences)
- **App Registration**: Improved bundle registration with LaunchServices
- **Known Issues**: None identified
- **Workarounds**: None required

#### macOS 14 (Sonoma)
- **TCC Database**: Improved permission prompt UI and timing
- **System Settings**: Further refinements to System Settings interface
- **App Registration**: More reliable LaunchServices integration
- **Known Issues**: Occasional timing issues with rapid permission requests
- **Workarounds**: Add small delays (100-500ms) between permission requests

#### macOS 13 (Ventura)
- **System Preferences → System Settings**: Major naming change
  - Application renamed from "System Preferences" to "System Settings"
  - macgo now includes alias support for both names
- **TCC Database**: Location and schema remain consistent
- **App Registration**: Requires proper bundle structure for TCC
- **Known Issues**:
  - System Settings name change breaks legacy scripts
  - Some AppleScript UI automation paths changed
- **Workarounds**:
  - Use macgo's alias support (automatically maps old names to new)
  - Update AppleScript paths for System Settings

#### macOS 12 (Monterey)
- **TCC Database**: Standard TCC behavior, stable schema
- **System Preferences**: Still using legacy "System Preferences" name
- **App Registration**: Requires well-formed bundles with proper Info.plist
- **Known Issues**:
  - ServicesLauncher may have occasional I/O forwarding delays
  - Some TCC prompts may not appear immediately
- **Workarounds**:
  - Use longer timeouts for permission prompts
  - Implement retry logic for TCC permission checks

#### macOS 11 (Big Sur)
- **TCC Database**: Earlier TCC implementation, mostly stable
- **System Preferences**: Uses "System Preferences" naming
- **App Registration**: Basic LaunchServices support
- **Known Issues**:
  - More frequent LaunchServices registration delays
  - Some entitlements may not be fully supported
- **Workarounds**:
  - Extended wait times for bundle registration
  - Verify TCC permissions manually after first launch

## Version Detection

macgo automatically detects the running macOS version. You can also check manually:

### Using `sw_vers` Command

```bash
sw_vers
# Output:
# ProductName:		macOS
# ProductVersion:	15.0
# BuildVersion:		24A335
```

### In Go Code

```go
package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func getMacOSVersion() (string, error) {
	cmd := exec.Command("sw_vers", "-productVersion")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func main() {
	version, err := getMacOSVersion()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("macOS Version: %s\n", version)
}
```

### Version Comparison Helper

```go
package main

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseMacOSVersion parses a version string like "14.2.1" into major, minor, patch
func ParseMacOSVersion(version string) (major, minor, patch int, err error) {
	parts := strings.Split(version, ".")
	if len(parts) < 1 {
		return 0, 0, 0, fmt.Errorf("invalid version format")
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, err
	}

	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) > 2 {
		patch, _ = strconv.Atoi(parts[2])
	}

	return major, minor, patch, nil
}

// IsVenturaOrLater checks if running macOS 13 (Ventura) or later
func IsVenturaOrLater(version string) bool {
	major, _, _, err := ParseMacOSVersion(version)
	if err != nil {
		return false
	}
	return major >= 13
}
```

## Testing Across Versions

### GitHub Actions CI Matrix

macgo uses GitHub Actions with a matrix strategy to test across multiple macOS versions:

```yaml
strategy:
  matrix:
    include:
      - macos-version: '13'
        os: macos-13
        go-version: '1.24'

      - macos-version: '14'
        os: macos-14
        go-version: '1.24'

      - macos-version: '15'
        os: macos-15
        go-version: '1.24'
```

### Running Version-Specific Tests

#### Basic Test Suite (All Versions)

```bash
go test -v ./...
```

#### With Race Detection

```bash
go test -race -v ./...
```

#### E2E Tests (Version-Aware)

```bash
# Basic E2E tests (no relaunch)
go test -v -run "TestE2E_" -timeout 2m

# TCC integration tests (requires Full Disk Access)
MACGO_E2E_TCC_TESTS=1 go test -v -run "TestE2E_TCC"

# Real launch tests (most comprehensive)
MACGO_E2E_REAL_LAUNCH=1 go test -v -run "TestE2E_RealExecutable"
```

### Manual Testing Checklist

When testing on a specific macOS version:

#### 1. **Bundle Creation and Launch**
```bash
# Test basic bundle creation
cd examples/hello
go run .

# Verify bundle structure
ls -la ~/Library/Developer/macgo/bundles/
```

#### 2. **TCC Permission Handling**
```bash
# Test camera permission
cd examples/camera-mic
go run .

# Verify TCC prompt appears
# Check System Settings → Privacy & Security → Camera
```

#### 3. **Code Signing**
```bash
# Test ad-hoc signing
cd examples/code-signing
go run .

# Verify signature
codesign -dv --verbose=4 ~/Library/Developer/macgo/bundles/YourApp.app
```

#### 4. **I/O Forwarding**
```bash
# Test stdin/stdout/stderr forwarding
cd examples/stdin-test
echo "test input" | go run .
```

#### 5. **ServicesLauncher**
```bash
# Test LaunchServices integration
cd examples/hello
MACGO_SERVICES_VERSION=2 go run .
```

#### 6. **System Settings/Preferences Access**
```bash
# Test alias support (works on all versions)
cd examples/list-app-windows
./list-app-windows -app "System Preferences"  # Big Sur, Monterey
./list-app-windows -app "System Settings"      # Ventura+
./list-app-windows -app "settings"             # Alias (all versions)
```

### Version-Specific Test Scenarios

#### macOS 15 (Sequoia) Specific Tests

1. **Enhanced TCC Prompts**
   - Verify prompt UI improvements
   - Test rapid permission request sequences
   - Validate permission persistence

2. **App Registration**
   - Test LaunchServices registration speed
   - Verify bundle appears in System Settings immediately

#### macOS 14 (Sonoma) Specific Tests

1. **Permission Prompt Timing**
   - Add delays between multiple permission requests
   - Test permission state caching

2. **Bundle Lifecycle**
   - Test bundle reuse after system reboot
   - Verify code signature persistence

#### macOS 13 (Ventura) Specific Tests

1. **System Settings Name Change**
   - Test both "System Preferences" and "System Settings" names
   - Verify alias resolution works correctly

2. **AppleScript Compatibility**
   - Test UI automation with new System Settings paths
   - Verify tcc-helper works with updated UI structure

#### macOS 12 (Monterey) Specific Tests

1. **Legacy Behavior**
   - Test with "System Preferences" (old name)
   - Verify longer timeout handling

2. **I/O Forwarding**
   - Test ServicesLauncherV1 and V2
   - Validate config file I/O strategy

#### macOS 11 (Big Sur) Specific Tests

1. **Basic Compatibility**
   - Test minimal permission sets
   - Verify basic bundle creation

2. **Workaround Validation**
   - Test extended registration timeouts
   - Validate manual permission verification

## Known Version-Specific Quirks

### Quirk Matrix

| Issue | Affected Versions | Severity | Workaround |
|-------|------------------|----------|------------|
| System Settings name change | 13+ | Low | Use alias support |
| TCC prompt timing delays | 14 | Low | Add 100-500ms delays |
| LaunchServices registration slow | 11-12 | Medium | Extended timeouts |
| ServicesLauncher I/O delays | 12 | Low | Use V2 launcher |
| AppleScript UI paths changed | 13+ | Medium | Update UI automation |
| Permission persistence issues | 11 | Medium | Manual verification |

### Detailed Quirk Documentation

#### 1. System Settings Name Change (macOS 13+)

**Symptom:** Scripts using "System Preferences" fail to find the app on Ventura+

**Impact:** Low - handled automatically by macgo

**Solution:**
```go
// macgo automatically handles this via alias support
// Both names work on all versions:
listWindows("-app", "System Preferences")  // Works
listWindows("-app", "System Settings")     // Works
listWindows("-app", "settings")            // Works (alias)
```

#### 2. TCC Prompt Timing (macOS 14)

**Symptom:** Permission prompts may not appear immediately when requesting multiple permissions

**Impact:** Low - temporary delay only

**Solution:**
```go
// Add small delays between permission requests
config := &macgo.Config{
    Permissions: []macgo.Permission{macgo.Camera},
}
macgo.Start(config)
time.Sleep(500 * time.Millisecond)  // Allow TCC prompt to settle

config2 := &macgo.Config{
    Permissions: []macgo.Permission{macgo.Microphone},
}
macgo.Start(config2)
```

#### 3. LaunchServices Registration Delays (macOS 11-12)

**Symptom:** Bundle may not appear in System Settings immediately after creation

**Impact:** Medium - affects first-run UX

**Solution:**
```go
// Use longer timeouts for permission checks
config := &macgo.Config{
    Permissions: []macgo.Permission{macgo.Camera},
}
macgo.Start(config)

// Wait for bundle registration
time.Sleep(2 * time.Second)

// Then check permissions
// Implement retry logic with exponential backoff
```

#### 4. ServicesLauncher I/O Forwarding (macOS 12)

**Symptom:** Occasional delays in stdout/stderr forwarding

**Impact:** Low - output appears eventually

**Solution:**
```bash
# Use ServicesLauncherV2 (more reliable on Monterey)
export MACGO_SERVICES_VERSION=2
go run your-app
```

## Version Migration Guide

### Upgrading from macOS 12 → 13+

**Changes to be aware of:**
1. System Preferences → System Settings name change
2. Some AppleScript UI paths changed
3. TCC prompt UI improved

**Migration steps:**
1. Update any hardcoded "System Preferences" references to use macgo's alias support
2. Test AppleScript-based UI automation (if used)
3. Update documentation to reflect new names

### Upgrading from macOS 11 → 12+

**Changes to be aware of:**
1. Improved LaunchServices integration
2. More reliable TCC prompt behavior
3. Better ServicesLauncher performance

**Migration steps:**
1. Test with shorter timeouts (can remove extended waits)
2. Verify bundle registration is faster
3. Consider enabling ServicesLauncherV2

## Best Practices for Version Compatibility

### 1. Don't Hardcode App Names

❌ **Bad:**
```go
appName := "System Preferences"  // Breaks on Ventura+
```

✅ **Good:**
```go
// Use macgo's alias support or runtime detection
appName := "settings"  // Works on all versions
```

### 2. Use Appropriate Timeouts

❌ **Bad:**
```go
time.Sleep(100 * time.Millisecond)  // Too short for older versions
```

✅ **Good:**
```go
// Adjust timeouts based on version or use generous defaults
timeout := 2 * time.Second
if isOlderMacOS() {
    timeout = 5 * time.Second
}
time.Sleep(timeout)
```

### 3. Test on Multiple Versions

❌ **Bad:**
- Testing only on your development machine's OS version

✅ **Good:**
- Use CI matrix to test on multiple versions
- Manual testing on physical/VM machines with different OS versions
- Document any version-specific behavior

### 4. Handle Version-Specific Features Gracefully

❌ **Bad:**
```go
// Assuming feature always exists
useNewFeature()
```

✅ **Good:**
```go
// Check version before using version-specific features
if major, _, _, _ := ParseMacOSVersion(version); major >= 14 {
    useNewFeature()
} else {
    useLegacyApproach()
}
```

### 5. Document Version Requirements

✅ **In your README.md:**
```markdown
## Requirements

- macOS 13.0 (Ventura) or later for optimal experience
- macOS 11.0 (Big Sur) minimum supported version
- Known limitations on macOS 11-12: [link to documentation]
```

## Troubleshooting Version-Specific Issues

### Issue: TCC Prompt Not Appearing

**Possible Causes:**
1. Bundle not properly registered with LaunchServices (common on macOS 11-12)
2. TCC database corrupted (rare)
3. Insufficient wait time for registration (older versions)

**Diagnosis:**
```bash
# Check if bundle is registered
lsregister -dump | grep -i YourAppName

# Check TCC database
sqlite3 ~/Library/Application\ Support/com.apple.TCC/TCC.db "SELECT * FROM access;"
```

**Solutions:**
1. Add longer wait times on older macOS versions
2. Reset TCC permissions: `tccutil reset All com.your.bundleid`
3. Verify bundle structure is correct

### Issue: System Settings/Preferences Not Found

**Possible Causes:**
1. Using wrong app name for macOS version
2. App name lookup not using alias support

**Diagnosis:**
```bash
# Check which name is used on your system
sw_vers -productVersion
ps aux | grep -i "system.*settings\|system.*preferences"
```

**Solutions:**
1. Use macgo's alias support (automatically handles version differences)
2. Use generic aliases like "settings" or "preferences"

### Issue: Code Signing Failures

**Possible Causes:**
1. Missing or expired certificates
2. Version-specific codesign requirements

**Diagnosis:**
```bash
# Check available certificates
security find-identity -p codesigning -v

# Check codesign version
codesign --version
```

**Solutions:**
1. Ensure Xcode Command Line Tools are up to date
2. Use ad-hoc signing (`-`) for development
3. Check certificate validity period

## CI/CD Configuration

### GitHub Actions Example

Complete example from macgo's `.github/workflows/test.yml`:

```yaml
jobs:
  test:
    strategy:
      fail-fast: false
      matrix:
        include:
          - macos-version: '13'
            os: macos-13
          - macos-version: '14'
            os: macos-14
          - macos-version: '15'
            os: macos-15

    runs-on: ${{ matrix.os }}

    steps:
    - name: Display system information
      run: |
        sw_vers
        go version

    - name: Run tests
      run: go test -race -v ./...
```

### Local VM Testing

For comprehensive version testing:

1. **Set up VMs:**
   - Use UTM or Parallels for macOS VMs
   - Create snapshots for each major version (11, 12, 13, 14, 15)

2. **Test Script:**
```bash
#!/bin/bash
# test-all-versions.sh

versions=("11" "12" "13" "14" "15")

for version in "${versions[@]}"; do
    echo "Testing on macOS $version..."
    # Start VM, run tests, capture results
    # Restore snapshot for next test
done
```

## Future Considerations

### Upcoming Changes

- **macOS 16**: Expected in fall 2025
  - Monitor WWDC announcements for TCC changes
  - Test beta versions when available

- **GitHub Actions Runners**:
  - macOS 13 retirement scheduled for Dec 2025
  - Plan migration to macOS 14 as minimum CI version

### Deprecation Timeline

| Version | End of Life | Action Required |
|---------|-------------|-----------------|
| macOS 11 | 2024 | Consider removing from test matrix |
| macOS 12 | 2025 | Limited support only |
| macOS 13 | Dec 2025 | CI runners retiring |

## Summary

macgo provides robust cross-version compatibility with:

- ✅ **Automatic version detection and adaptation**
- ✅ **Built-in alias support for renamed system apps**
- ✅ **CI testing across macOS 13, 14, 15**
- ✅ **Documented version-specific quirks and workarounds**
- ✅ **Comprehensive testing strategies**
- ✅ **Migration guides for version upgrades**

For version-specific issues not covered here, please file an issue at:
https://github.com/tmc/macgo/issues

## Related Documentation

- [E2E_TESTING.md](E2E_TESTING.md) - End-to-end testing guide
- [internal/launch/TESTING.md](internal/launch/TESTING.md) - ServicesLauncher testing
- [README.md](README.md) - Main macgo documentation
- [.github/workflows/test.yml](.github/workflows/test.yml) - CI configuration
