# Stdin Forwarding Test Suite for macgo

This test suite comprehensively validates stdin forwarding functionality in macgo, including interactive applications, EOF handling, line buffering, and terminal control characters.

## Test Results Summary

### ✅ Working Features

1. **Basic CLI Prompts** - Fully functional in both nobundle and bundle modes
   - Simple text input with prompts
   - Multiple sequential prompts
   - Yes/No prompts

2. **EOF Handling** - Correctly implemented
   - Proper EOF detection when pipe closes
   - Multiple line reading until EOF
   - Correct behavior after EOF (continues to return EOF)
   - Partial line handling before EOF

3. **Line Buffering** - Works correctly
   - Buffered line reading with `bufio.Reader`
   - Byte-by-byte reading with `io.ReadFull`
   - Scanner-based word tokenization

4. **Terminal Control Characters** - Properly forwarded
   - Tab characters (`\t`) preserved and counted correctly
   - Newline (`\n`) and carriage return (`\r`) handling
   - Control character detection
   - Unicode characters properly forwarded

5. **Multi-line Input** - Fully supported
   - Multiple lines with line-by-line processing
   - End marker detection
   - Line counting

6. **Stdin Properties Detection** - Accurate
   - Pipe detection works in nobundle mode
   - Bundle mode shows stdin as pipe/regular file (depending on strategy)
   - File descriptor information correct

## Test Programs

### stdin-test

A comprehensive test program with multiple test modes:

```bash
stdin-test [flags]

Flags:
  -prompt         Test CLI prompts (name, age, yes/no)
  -password       Test password-style input (simulated)
  -eof            Test EOF handling and detection
  -linebuffer     Test different buffering modes
  -control        Test terminal control characters (tabs, CR/LF)
  -multiline      Test multi-line input with END marker
  -timeout        Test input timeout (5 seconds)
  -verbose        Enable verbose output
```

### Running Individual Tests

#### Basic CLI Prompt Test
```bash
# Nobundle mode (direct execution)
echo -e "Alice\n25\ny" | MACGO_NOBUNDLE=1 stdin-test -prompt

# Bundle mode with stdin forwarding
echo -e "Bob\n30\ny" | MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1 stdin-test -prompt
```

#### EOF Handling Test
```bash
# Test EOF detection
echo -e "line1\nline2\nline3" | MACGO_NOBUNDLE=1 stdin-test -eof

# With bundle
echo -e "alpha\nbeta\ngamma" | MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1 stdin-test -eof
```

#### Control Characters Test
```bash
# Test tab characters
echo -e "hello\t\tworld\nmore\tdata" | MACGO_NOBUNDLE=1 stdin-test -control

# With bundle
echo -e "test\t\tdata\nmore" | MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1 stdin-test -control
```

#### Multi-line Input Test
```bash
# Test multi-line with END marker
echo -e "first\nsecond\nthird\nEND" | MACGO_NOBUNDLE=1 stdin-test -multiline

# With bundle
echo -e "A\nB\nC\nEND" | MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1 stdin-test -multiline
```

#### Password Input Test (Simulated)
```bash
# Test password matching
echo -e "secret123\nsecret123" | MACGO_NOBUNDLE=1 stdin-test -password

# With bundle
echo -e "mypass\nmypass" | MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1 stdin-test -password
```

### Automated Test Suite

Run the comprehensive automated test suite:

```bash
cd examples/stdin-test
./test-stdin.sh
```

The test suite includes:
- 20+ automated tests covering all scenarios
- Tests in nobundle mode (baseline)
- Tests with config-file strategy (default, working)
- Tests with env-vars strategy (may not work on all macOS versions)
- EOF edge cases (empty input, no trailing newline, long lines)
- Special characters (unicode, tabs, control chars)
- Concurrent I/O (stdin + stdout + stderr together)
- Timeout tests
- Pipe detection tests

## Implementation Details

### Stdin Forwarding Architecture

**Parent Process (macgo launcher):**
- Creates named pipe in `/tmp/macgo-PID-TIMESTAMP/stdin`
- Opens pipe in non-blocking mode initially
- Switches to blocking mode for data transfer
- Forwards stdin using `io.Copy` in goroutine
- Closes pipe on context cancellation or parent stdin EOF

**Child Process (bundled app):**
- Discovers pipe path via config file (default strategy)
- Opens pipe with `O_RDONLY` flag
- Replaces `os.Stdin` with opened pipe
- Reads normally from `os.Stdin`

### Configuration

Enable stdin forwarding:
```bash
# Enable stdin forwarding
export MACGO_ENABLE_STDIN_FORWARDING=1

# Or enable all I/O forwarding
export MACGO_ENABLE_IO_FORWARDING=1

# Specify strategy (config-file is default and recommended)
export MACGO_IO_STRATEGY=config-file  # DEFAULT, WORKING
export MACGO_IO_STRATEGY=env-vars     # May not work with LaunchServices
```

### Key Behaviors

1. **EOF Propagation**: When parent's stdin reaches EOF, the pipe is closed, and child receives EOF
2. **Buffering**: Standard Go buffering applies (line buffered for terminals, block buffered for pipes)
3. **Control Characters**: All control characters are forwarded unchanged
4. **Unicode**: UTF-8 encoded text is forwarded correctly
5. **Binary Data**: Binary data can be forwarded (though this test suite focuses on text)

## Limitations and Known Issues

### Current Limitations

1. **No TTY Control**: The forwarded stdin is a pipe/file, not a TTY, so:
   - No support for raw mode / terminal escape sequences
   - No support for terminal size queries (TIOCGWINSZ)
   - No support for `termios` settings
   - Password input doesn't hide characters (no echo control)

2. **No Interactive Terminal Features**:
   - No readline support (arrow keys, history)
   - No job control signals (Ctrl+Z)
   - No terminal attributes

3. **LaunchServices Limitations**:
   - `open --env` doesn't pass environment variables reliably
   - env-vars strategy may not work on all macOS versions
   - Config-file strategy is the only reliable method

4. **Performance**:
   - Small latency added by pipe forwarding
   - Context switches between parent and child processes

### Workarounds

For applications that need true TTY support:
```bash
# Use nobundle mode for full TTY support
MACGO_NOBUNDLE=1 your-app

# Or use DirectLauncher (bypasses LaunchServices)
MACGO_LAUNCHER=direct your-app
```

For password input:
```go
// Use golang.org/x/term for proper password input
import "golang.org/x/term"

// This will work in nobundle mode but not with stdin forwarding
password, err := term.ReadPassword(int(os.Stdin.Fd()))
```

## Test Coverage

This test suite provides coverage for:

- ✅ CLI prompt input (sequential prompts)
- ✅ EOF detection and handling
- ✅ EOF edge cases (empty, partial lines, long lines)
- ✅ Line-buffered input
- ✅ Byte-by-byte input
- ✅ Scanner-based tokenization
- ✅ Tab characters
- ✅ Newline/carriage return
- ✅ Control character detection
- ✅ Unicode characters
- ✅ Multi-line input
- ✅ Timeout scenarios
- ✅ Pipe detection
- ✅ File mode detection
- ✅ Concurrent I/O (stdin + stdout + stderr)
- ✅ Both nobundle and bundle modes
- ✅ Config-file strategy
- ✅ Env-vars strategy

## Compatibility

**macOS Versions Tested:**
- macOS Sonoma 15.x (Darwin 25.0.0)

**Go Versions:**
- Go 1.21+

**Strategies:**
- Config-file: ✅ Working (default, recommended)
- Env-vars: ⚠️  May not work (LaunchServices limitation)
- Direct: ✅ Working (no stdin forwarding needed)

## Future Improvements

Potential enhancements for stdin forwarding:

1. **Pseudo-TTY Support**: Use `pty` package to provide true TTY
2. **Terminal Size Forwarding**: Forward SIGWINCH signals
3. **Raw Mode Support**: Allow applications to control terminal modes
4. **Readline Integration**: Provide readline support for bundled apps
5. **Signal Forwarding**: Forward Ctrl+C, Ctrl+Z to child
6. **Binary Mode**: Explicit binary vs text mode handling

## Related Files

- `main.go` - Test program implementation
- `test-stdin.sh` - Automated test suite
- `README.md` - This documentation
- `../../internal/launch/services.go` - Stdin forwarding implementation (V1)
- `../../internal/launch/services_v2.go` - Stdin forwarding implementation (V2)
- `../../macgo_darwin.go` - Child-side stdin redirection

## References

- [macgo Testing Documentation](../../docs/testing.md)
- [I/O Forwarding Strategies](../../docs/io-forwarding.md)
- [LaunchServices Limitations](../../docs/launchservices.md)
