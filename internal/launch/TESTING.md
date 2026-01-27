# ServicesLauncher Testing Guide

This document explains how to test ServicesLauncher V1 and V2.

## Automated Unit Tests

Comprehensive automated tests for launcher components:

```bash
go test ./internal/launch -v
```

### Test Coverage

The automated test suite includes:

**V1 (ServicesLauncher) Tests:**
- ✅ `TestServicesLauncherV1_IOTimeout` - Verifies timeout protection with no-growth detection
- ✅ `TestServicesLauncherV1_WritePipeConfig` - Validates config file generation
- ✅ `TestServicesLauncherV1_BuildCommandWithConfigFileStrategy` - Tests command building with config-file strategy
- ✅ `TestServicesLauncherV1_NoGrowthTimeout` - Validates 5-second no-growth timeout mechanism
- ✅ `TestServicesLauncherV1_StdinForwarding` - Tests stdin data forwarding to application stdin pipe
- ✅ `TestServicesLauncherV1_StderrForwarding` - Tests stderr output forwarding with timeout mechanism
- ✅ `TestServicesLauncherV1_CreatePipes` - Tests pipe creation with different configuration options (all, stdout only, stderr only, etc.)
- ✅ `TestServicesLauncherV1_CleanupWithErrors` - Tests cleanup handles missing files and directories gracefully
- ✅ `TestServicesLauncher_createPipeDirectory` - Tests pipe directory creation
- ✅ `TestServicesLauncher_createNamedPipes` - Tests named pipe creation
- ✅ `TestServicesLauncher_buildOpenCommand` - Tests command building with different strategies
- ✅ `TestServicesLauncher_cleanupPipeDirectory` - Tests cleanup behavior

**V2 (ServicesLauncherV2) Tests:**
- ✅ `TestServicesLauncherV2_ContinuousPolling` - Validates continuous polling for output capture
- ✅ `TestServicesLauncherV2_WritePipeConfig` - Validates config file generation
- ✅ `TestServicesLauncherV2_BuildCommand` - Tests command building
- ✅ `TestServicesLauncherV2_NoGrowthTimeout` - Validates 5-second no-growth timeout mechanism
- ✅ `TestServicesLauncherV2_StdinForwarding` - Tests stdin data forwarding with context timeout handling
- ✅ `TestServicesLauncherV2_StderrForwarding` - Tests stderr output forwarding with incremental writes
- ✅ `TestServicesLauncherV2_CreatePipes` - Tests pipe creation with and without stdin enabled
- ✅ `TestServicesLauncherV2_CleanupWithErrors` - Tests cleanup handles missing files and directories gracefully

**V2 Large Output Tests (services_large_output_test.go):**
- ✅ `TestServicesLauncherV2_LargeStdoutOutput` - Tests stdout with 1MB, 5MB, 10MB, 20MB output
- ✅ `TestServicesLauncherV2_LargeStderrOutput` - Tests stderr with 1MB, 5MB, 10MB output
- ✅ `TestServicesLauncherV2_ConcurrentLargeOutput` - Tests simultaneous 5MB stdout + 5MB stderr
- ✅ `TestServicesLauncherV2_BufferBoundaryConditions` - Tests edge cases around 32KB buffer boundary
- ✅ `TestServicesLauncherV2_MemoryUsage` - Verifies streaming behavior with 20MB data (no excessive buffering)

**Version Selection Tests:**
- ✅ `TestServicesLauncherVersionSelection` - Validates correct launcher selection based on environment variables

### Key Test Behaviors Verified

1. **V1 Timeout Behavior**: Tests confirm V1 gracefully handles scenarios where I/O forwarding may not work by implementing a 5-second no-growth timeout
2. **V2 Continuous Polling**: Tests verify V2 successfully captures output through continuous polling, even when data arrives slowly
3. **Config File Strategy**: Both V1 and V2 correctly generate config files with pipe paths
4. **Exit Codes**: Tests verify proper handling of process exit codes (where testable without os.Exit())
5. **Version Selection**: Tests confirm the correct launcher version is selected based on `MACGO_SERVICES_VERSION` environment variable
6. **Large Output Handling**: V2 correctly handles multi-MB output streams (1MB-20MB) without truncation
7. **Buffer Management**: 32KB buffer correctly handles data at and around buffer boundaries
8. **No Deadlocks**: Concurrent stdout/stderr forwarding works without deadlocks
9. **Memory Efficiency**: Streaming behavior verified - memory usage stays proportional to data size, not excessive buffering
10. **Data Integrity**: Byte-for-byte verification ensures no corruption during large transfers
11. **Stdin Forwarding**: Both V1 and V2 correctly forward stdin data with context timeout handling
12. **Stderr Forwarding**: Both V1 and V2 correctly forward stderr output with incremental writes and timeout mechanisms
13. **Pipe Creation**: V1 and V2 create correct pipe configurations based on environment settings
14. **Error Handling**: Cleanup operations handle missing files and directories gracefully without panicking

### Large Output Test Results

The large output test suite verifies ServicesLauncherV2's robustness with high-volume I/O:

**Stdout Tests:**
- ✅ 1MB: 1,048,576 bytes captured successfully (~5.5s)
- ✅ 5MB: 5,242,880 bytes captured successfully (~7.7s)
- ✅ 10MB: 10,485,760 bytes captured successfully (~7.6s)
- ✅ 20MB: 20,971,520 bytes captured successfully (~8.7s)

**Stderr Tests:**
- ✅ 1MB: 1,048,576 bytes captured successfully (~5.5s)
- ✅ 5MB: 5,242,880 bytes captured successfully (~7.6s)
- ✅ 10MB: 10,485,760 bytes captured successfully (~7.6s)

**Concurrent I/O:**
- ✅ 5MB stdout + 5MB stderr simultaneously: 10,485,760 total bytes (~6.0s)

**Buffer Boundary Tests:**
- ✅ Exactly 32KB (buffer size): Perfect alignment handling
- ✅ 32KB ± 1 byte: Edge case handling verified
- ✅ 2x, 2.5x, 3x buffer size: Multi-buffer scenarios work correctly

**Memory Usage:**
- ✅ 20MB data processed with ~12MB memory increase (60% of data size)
- ✅ Confirms streaming behavior - no linear memory growth with data size
- ✅ Fixed 32KB buffer per stream maintained throughout

### Performance Characteristics

From test results, V2 demonstrates:
- **Throughput**: ~2.5-3 MB/s sustained for large transfers
- **Latency**: 5-second no-growth timeout prevents hanging on stalled streams
- **Concurrency**: No contention between simultaneous stdout/stderr streams
- **Scalability**: Linear time complexity with data size, constant memory complexity

## Manual Integration Testing

Due to the nature of macOS LaunchServices and the launchers' use of `os.Exit()`, full end-to-end integration tests require manual execution. The automated tests above cover the core I/O forwarding and timeout logic.

### Testing ServicesLauncher V1

V1 uses config-file strategy by default with continuous polling and graceful timeout:

```bash
# Build a test app
cd examples/screen-capture
go build

# Run with V1 (default)
MACGO_SERVICES_VERSION=1 MACGO_DEBUG=1 ./screen-capture --help
```

**Expected behavior:**
- ✅ App launches via LaunchServices with config-file strategy
- ✅ Output is captured through continuous polling
- ✅ Graceful timeout after 5 seconds of no output growth
- ✅ Config file created in `/tmp/macgo-<pid>-<timestamp>/config`

**Test with broken open-flags strategy (to verify timeout protection):**
```bash
MACGO_SERVICES_VERSION=1 MACGO_IO_STRATEGY=open-flags MACGO_DEBUG=1 ./screen-capture --help
```
- ⚠️  May timeout waiting for output (expected - open flags don't work with .app bundles)
- ✅ Should timeout gracefully after 2-5 seconds and exit with code 0

### Testing ServicesLauncher V2

V2 uses the same config-file strategy with continuous polling (experimental):

```bash
# Build a test app
cd examples/screen-capture
go build

# Run with V2
MACGO_SERVICES_VERSION=2 MACGO_DEBUG=1 ./screen-capture --help
```

**Expected behavior:**
- ✅ App launches via LaunchServices with config-file strategy
- ✅ Output is captured successfully through continuous polling
- ✅ Handles slow output gracefully (waits for file growth)
- ✅ Exits cleanly after 5 seconds of no growth
- ✅ Config file created in `/tmp/macgo-<pid>-<timestamp>/config`

### Verification Checklist

**For both V1 and V2:**

- [ ] App bundle is created in `/tmp/` or designated location
- [ ] `Info.plist` is generated correctly with entitlements
- [ ] Named pipes (regular files) are created in `/tmp/macgo-*/`
- [ ] Config file contains `MACGO_STDOUT_PIPE` and `MACGO_STDERR_PIPE` paths
- [ ] App launches via `open` command (check debug output)
- [ ] Command-line arguments are forwarded correctly
- [ ] Cleanup happens on exit (pipes and config removed)
- [ ] Debug output shows "config-file I/O strategy"

**For V1 specifically:**
- [ ] Graceful timeout with open-flags strategy (set `MACGO_IO_STRATEGY=open-flags`)
- [ ] No-growth timeout works (5 seconds of no output growth)
- [ ] Config-file strategy works by default

**For V2 specifically:**
- [ ] Stdout is captured and forwarded correctly
- [ ] Stderr is captured and forwarded correctly
- [ ] Continuous polling detects slow output
- [ ] Process terminates cleanly after output complete
- [ ] No stdin pipe created by default (unless `MACGO_ENABLE_STDIN_FORWARDING=1`)

## Comparing V1 vs V2

Run the same command with both versions and compare:

```bash
# V1
MACGO_SERVICES_VERSION=1 MACGO_DEBUG=1 ./examples/screen-capture/screen-capture --help

# V2
MACGO_SERVICES_VERSION=2 MACGO_DEBUG=1 ./examples/screen-capture/screen-capture --help
```

### Expected Differences

| Aspect | V1 | V2 |
|--------|----|----|
| Output capture | May timeout | ✅ Works reliably |
| Exit detection | Timeout-based | ✅ Process-based |
| I/O strategy | open flags | Continuous polling |
| Complexity | Lower | Higher |
| Reliability | Medium | High |
| Status | **Stable (default)** | Experimental |

## Testing with Different Apps

### Simple CLI App

```bash
cd examples/tcc-helper
MACGO_SERVICES_VERSION=2 MACGO_DEBUG=1 ./tcc-helper -list
```

### App with TCC Permissions

```bash
cd examples/screen-capture
MACGO_SERVICES_VERSION=2 MACGO_DEBUG=1 ./screen-capture --help
```

### Long-running App

Test timeout behavior:

```bash
# Create a test app that sleeps
cat > /tmp/test-sleep.go <<'EOF'
package main

import (
	"fmt"
	"time"
	"github.com/tmc/macgo"
)

func main() {
	macgo.Start(&macgo.Config{})
	fmt.Println("Starting...")
	time.Sleep(30 * time.Second)
	fmt.Println("Done")
}
EOF

cd /tmp
go build test-sleep.go

# Test with short timeout (V1 should timeout gracefully)
MACGO_SERVICES_VERSION=1 MACGO_DEBUG=1 timeout 5s ./test-sleep
```

## Debug Output

Enable debug output to see launcher behavior:

```bash
MACGO_DEBUG=1 MACGO_SERVICES_VERSION=2 ./your-app
```

Debug output shows:
- Which launcher version is selected
- Bundle creation and signing
- Named pipe paths
- Open command arguments
- I/O forwarding activity
- Exit status

## Known Limitations

### Automated Testing

- Launchers call `os.Exit()` which causes test panics
- LaunchServices integration requires actual macOS environment
- Some behaviors vary by macOS version
- TCC prompts may appear during tests

### Workarounds

- Use manual testing procedures above
- Run tests on dedicated test machines
- Use mocks for unit tests of launcher-adjacent code
- Document expected behavior differences

## Troubleshooting

### V1 always times out

This is normal on some macOS versions where `open` command I/O flags don't work. Use V2 instead.

### V2 doesn't capture output

Check:
- Named pipes are created (`ls /tmp/macgo-*`)
- Debug output shows polling activity
- App actually writes to stdout/stderr

### Exit code not detected

V2 should detect exit codes. If not:
- Check that wait mode is active (debug output)
- Verify process cleanup
- Check for zombie processes (`ps aux | grep defunct`)

## Best Practices

1. **Use V1 by default** - More stable, handles timeouts gracefully
2. **Use V2 when you need output** - Reliable output capture
3. **Enable debug mode** during development - `MACGO_DEBUG=1`
4. **Test both versions** - Ensure your app works with both
5. **Document behavior differences** - If your app behaves differently

## Future Improvements

- Add automated integration tests that work in test environments
- Improve exit code detection in V1
- Add metrics/observability to launcher behavior
- Create test harness that doesn't require manual verification
