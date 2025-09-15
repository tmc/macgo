# Cross-Platform Example

This example demonstrates how macgo works gracefully across different platforms.

## Running the Example

### On macOS
```bash
go run main.go
```

Expected output:
- Creates an app bundle with camera/microphone permissions
- Shows "✓ Running inside macOS app bundle with TCC permissions"

### On Other Platforms
```bash
go run main.go
```

Expected output:
- Shows "✓ Running as regular binary on [platform]"
- macgo functions are safe no-ops

### With Debug Output
```bash
MACGO_DEBUG=1 go run main.go
```

This will show detailed debug information about what macgo is doing, including platform-specific behavior.

## Simulated Non-Darwin Testing

To see how macgo behaves on non-macOS platforms:

```bash
go run test-non-darwin.go
```

This simulates the warning messages and behavior you would see on Linux, Windows, or other platforms.

## Key Points

1. **No build constraints needed** - The same code works on all platforms
2. **Graceful degradation** - macgo functions are no-ops on non-macOS
3. **Debug visibility** - Use `MACGO_DEBUG=1` to see what's happening
4. **Cross-platform safety** - No panics or errors on unsupported platforms

This approach allows you to write applications that can leverage macOS TCC permissions when available, while still working normally on other platforms.