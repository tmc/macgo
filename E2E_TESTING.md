# End-to-End Testing Guide

This document explains the comprehensive end-to-end (E2E) testing infrastructure for macgo.

## Overview

The E2E test suite (`e2e_test.go`) validates complete macgo workflows from configuration through bundle creation, code signing, and launch strategies. These tests verify the integration of all components including bundle management, TCC permissions, I/O forwarding, and launch mechanisms.

## Test Categories

### 1. Basic Workflow Tests

**TestE2E_BasicWorkflow** - Validates the fundamental end-to-end flow:
- Single permission requests (camera)
- Multiple permissions (camera + microphone)
- Custom entitlements
- Bundle creation without relaunch

**TestE2E_BundleCreationAndReuse** - Verifies bundle lifecycle:
- Initial bundle creation
- Bundle reuse on subsequent runs
- Bundle persistence with KeepBundle option

### 2. Code Signing Tests

**TestE2E_CodeSigningStrategies** - Tests different signing approaches:
- Ad-hoc signing (`-` identity)
- Automatic Developer ID detection
- No signing (unsigned bundles)

Signing tests gracefully skip in CI environments where certificates aren't available.

### 3. Configuration Tests

**TestE2E_BundleIdentifierGeneration** - Validates bundle ID generation:
- Automatic generation from module path
- Explicit bundle ID specification
- Verification that old `com.macgo` prefix is not used

**TestE2E_EnvironmentConfiguration** - Tests environment variable configuration:
- `MACGO_APP_NAME`, `MACGO_BUNDLE_ID`, `MACGO_DEBUG`
- `MACGO_CAMERA`, `MACGO_MICROPHONE` permission flags
- `Config.FromEnv()` functionality

**TestE2E_BuilderPattern** - Validates fluent configuration API:
- Builder method chaining
- Permission accumulation
- Debug and signing option configuration

### 4. Error Handling Tests

**TestE2E_ErrorHandling** - Tests error conditions:
- Invalid bundle ID formats
- App groups without required sandbox permission
- Configuration validation before execution

**TestE2E_ConfigValidation** - Tests config validation:
- Valid minimal configurations
- Valid comprehensive configurations
- Invalid bundle ID detection
- App groups dependency validation

### 5. Bundle Structure Tests

**TestE2E_BundleStructure** - Verifies created bundles:
- Correct directory structure (`Contents/`, `Contents/MacOS/`)
- Required `Info.plist` with all keys
- Permission usage descriptions for TCC
- Code signing verification

Required bundle structure:
```
MyApp.app/
├── Contents/
│   ├── Info.plist
│   ├── MacOS/
│   │   └── MyApp
│   └── (optional entitlements)
```

### 6. Launch Strategy Tests

**TestE2E_LaunchStrategySelection** - Tests execution modes:
- Force direct execution (`ForceDirectExecution`)
- Force LaunchServices (`ForceLaunchServices`)
- Automatic strategy selection

Launch strategy selection depends on:
- Requested permissions (TCC triggers LaunchServices)
- User configuration flags
- I/O forwarding requirements

### 7. Permission Tests

**TestE2E_PermissionValidation** - Validates TCC permission handling:
- Single permissions
- Multiple permissions
- Empty permission lists
- Permission dependency validation

### 8. Advanced Tests

**TestE2E_ContextCancellation** - Tests context cancellation:
- Graceful shutdown with context timeout
- Resource cleanup on cancellation

**TestE2E_ConcurrentStarts** - Tests concurrent execution:
- Multiple simultaneous `Start()` calls
- Bundle isolation per app name
- No race conditions in bundle creation

### 9. TCC Integration Tests (Optional)

**TestE2E_TCCIntegration** - Tests TCC permission reset:
- Requires `MACGO_E2E_TCC_TESTS=1` environment variable
- Requires Full Disk Access for `tccutil reset`
- Validates `MACGO_RESET_PERMISSIONS=1` functionality

**TestE2E_AppGroups** - Tests app groups configuration:
- App groups with sandbox permission
- Team ID substitution
- Requires valid Developer ID certificate

### 10. Real Launch Tests (Optional)

**TestE2E_RealExecutableTest** - Full integration test:
- Requires `MACGO_E2E_REAL_LAUNCH=1` environment variable
- Creates and compiles real Go executable
- Executes through complete macgo workflow
- Verifies actual bundle launch

## Running Tests

### Basic E2E Tests (No Relaunch)

Run all E2E tests with relaunch disabled:

```bash
go test -v -run "TestE2E_" -timeout 2m
```

These tests use `MACGO_NO_RELAUNCH=1` internally to prevent actual relaunching, focusing on bundle creation and configuration validation.

### TCC Integration Tests

Run with TCC reset capability (requires Full Disk Access):

```bash
MACGO_E2E_TCC_TESTS=1 go test -v -run "TestE2E_TCC"
```

### Real Launch Tests

Run with actual executable launch (most comprehensive):

```bash
MACGO_E2E_REAL_LAUNCH=1 go test -v -run "TestE2E_RealExecutable"
```

Warning: This creates temporary executables and launches them through macgo. May trigger TCC prompts.

### Benchmark Tests

Measure bundle creation performance:

```bash
go test -bench=BenchmarkE2E_BundleCreation -benchmem
```

## Test Environment Variables

The E2E test suite respects these environment variables:

| Variable | Purpose | Default |
|----------|---------|---------|
| `MACGO_NO_RELAUNCH` | Skip relaunch (set by tests) | `1` in tests |
| `MACGO_E2E_TCC_TESTS` | Enable TCC integration tests | `0` (skip) |
| `MACGO_E2E_REAL_LAUNCH` | Enable real executable tests | `0` (skip) |
| `MACGO_DEBUG` | Enable debug output | `0` |

## Test Scenarios Covered

### Complete Workflow Coverage

1. **Configuration**: Environment variables, builder pattern, validation
2. **Bundle Creation**: Fresh creation, reuse, caching
3. **Code Signing**: Ad-hoc, Developer ID, unsigned
4. **Permissions**: Single, multiple, custom entitlements
5. **Launch Strategies**: Direct, LaunchServices, automatic
6. **I/O Forwarding**: Config file, pipes, redirection
7. **Error Handling**: Invalid configs, missing permissions
8. **TCC Integration**: Permission reset, app groups
9. **Concurrency**: Multiple simultaneous starts
10. **Context Management**: Cancellation, timeouts

### Fresh TCC Permissions (CI/VM)

For true fresh TCC state testing:

1. Use VM snapshots to reset TCC database
2. Set `MACGO_E2E_TCC_TESTS=1` and `MACGO_E2E_REAL_LAUNCH=1`
3. Run full test suite
4. Restore VM snapshot after tests

This validates:
- First-run TCC prompt behavior
- Permission request dialogs
- Bundle registration with TCC

## Test Architecture

### Test Isolation

Each test:
- Uses unique app names (`E2ETest*`)
- Uses unique bundle IDs (`com.test.e2e.*`)
- Cleans up temporary bundles (unless KeepBundle=true)
- Sets `MACGO_NO_RELAUNCH=1` to prevent os.Exit()

### Test Dependencies

E2E tests depend on these internal packages:
- `github.com/tmc/macgo` - Main API
- `github.com/tmc/macgo/internal/bundle` - Bundle creation
- `github.com/tmc/macgo/internal/system` - System utilities
- `github.com/tmc/macgo/internal/tcc` - TCC permission handling

### Test Helpers

The suite includes helper benchmarks:
- `BenchmarkE2E_BundleCreation` - Measures bundle creation performance

## CI Integration

### GitHub Actions / CI Recommendations

```yaml
- name: Run E2E Tests
  run: |
    # Basic tests (no TCC/launch)
    go test -v -run "TestE2E_" -timeout 2m

    # Optional: Real launch tests on macOS runners
    # MACGO_E2E_REAL_LAUNCH=1 go test -v -run "TestE2E_RealExecutable"
```

### Expected Test Results

With default settings (MACGO_NO_RELAUNCH=1):
- ✅ All basic workflow tests pass
- ✅ Bundle structure tests pass
- ✅ Configuration tests pass
- ⏭️ TCC integration tests skipped
- ⏭️ Real launch tests skipped

With TCC/launch enabled:
- ✅ All tests pass
- ⚠️ May trigger TCC permission dialogs
- ⚠️ Requires Full Disk Access for TCC reset

## Common Issues

### Certificate Not Found

```
auto_signing test skipped: no Developer ID Application certificate found
```

**Solution**: This is expected in CI. Tests gracefully skip when certificates aren't available.

### Full Disk Access Required

```
TCC reset failed: Full Disk Access required
```

**Solution**: TCC reset tests are optional. Run with `MACGO_E2E_TCC_TESTS=1` only in environments with FDA.

### Build Fails on Non-Darwin

```
TestE2E_* skipped: E2E tests only run on darwin
```

**Solution**: This is expected. E2E tests require macOS (darwin platform).

## Best Practices

### For Development

1. Run basic E2E tests frequently: `go test -v -run "TestE2E_"`
2. Use `MACGO_DEBUG=1` for detailed output
3. Keep bundles for inspection: Tests use KeepBundle where appropriate
4. Check bundle structure manually in `/tmp/*-E2ETest*.app`

### For CI/CD

1. Run basic E2E tests on all macOS runners
2. Skip TCC/launch tests in standard CI
3. Use VM snapshots for fresh TCC state testing
4. Monitor test performance with benchmarks

### For Contributors

1. Add E2E tests for new features
2. Ensure tests work with `MACGO_NO_RELAUNCH=1`
3. Provide clear skip messages for optional tests
4. Document any new environment variables

## Future Enhancements

Potential improvements for the E2E test suite:

1. **VM Integration**: Automated VM snapshot management
2. **TCC Mock**: Simulate TCC database for testing
3. **Launch Monitoring**: Verify app actually launches
4. **Performance Regression**: Track bundle creation times
5. **Error Injection**: Test failure recovery paths
6. **Multi-Version**: Test across macOS versions

## Related Documentation

- [MACOS_VERSION_COMPATIBILITY.md](MACOS_VERSION_COMPATIBILITY.md) - macOS version compatibility guide
- [TESTING.md](internal/launch/TESTING.md) - ServicesLauncher test documentation
- [README.md](README.md) - Main macgo documentation
- [examples/](examples/) - Example applications

## Summary

The E2E test suite provides comprehensive coverage of macgo workflows:
- ✅ 15+ test functions covering all major scenarios
- ✅ Bundle creation, signing, permissions, launch strategies
- ✅ Configuration validation and error handling
- ✅ Optional TCC integration and real launch tests
- ✅ Concurrent execution and context management
- ✅ Performance benchmarks

All tests pass in standard CI environments with appropriate skips for platform-specific or permission-requiring tests.
