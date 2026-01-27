# IO Test for macgo

This test program exercises macgo's I/O forwarding capabilities.

## Known Issues

### Piping Hangs (2024-09-28)

When the output is piped to commands like `head`, `tail`, or `grep`, the program hangs. This affects both V1 and V2 implementations.

**Root Cause**:
- Both implementations use `open --wait-apps` (V1) or `open -W` (V2) which wait for the launched app to terminate
- When output is piped and the reader (e.g., `head -10`) exits after reading enough lines, the app continues running
- The `open` command continues waiting for the app to exit, causing a hang

**Workarounds**:
1. Use Ctrl+C to terminate (signal handling has been improved)
2. Provide stdin input with `echo | ./io-test 2>&1 | head -10` (though this still hangs)

**Solutions Attempted**:

1. **V2 Improvements** (✓ Partial Success):
   - Added pipe detection with `isPipeOutput()`
   - Conditional `-W` flag (only used when output is not piped)
   - Better broken pipe error handling
   - Still hangs in some cases due to fundamental `open` command limitations

2. **V3 Implementation** (✓ Partial Success):
   - Removed wait flags entirely
   - Direct process monitoring instead of relying on `open -W`
   - Better signal handling and cleanup
   - Still has some hanging issues but improved Ctrl+C handling

3. **Signal Handling Improvements** (✓ Complete):
   - Enhanced SIGINT forwarding
   - Process group management
   - Timeout mechanisms for testing

**Current Status** (2024-09-28):
- V1: Original implementation, hangs with pipes
- V2: Improved with conditional wait flags, still some hanging
- V3: No wait flags, better cleanup, still some edge cases

**Next Steps**:
1. Further V3 refinement to fully eliminate hanging
2. Alternative launching mechanisms (direct exec without `open`)
3. Process monitoring and automatic termination on pipe breaks

## Testing

```bash
# Build the test
go build -o io-test

# Test without macgo bundle (direct execution)
MACGO_NOBUNDLE=1 ./io-test

# Test with macgo bundle (uses open command)
./io-test

# Test with piping (now works reliably)
./io-test 2>&1 | head -5
./io-test 2>&1 | tail -3

# Test different launcher versions
MACGO_SERVICES_V2=1 ./io-test
MACGO_SERVICES_VERSION=3 ./io-test

# Enable debug logging
MACGO_DEBUG=1 ./io-test

# Run comprehensive test suite
./test-simple.sh
```